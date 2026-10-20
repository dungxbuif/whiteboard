package postgres

import (
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RetentionRepository struct {
	db                 *gorm.DB
	afterCandidateScan func()
}

func NewRetentionRepository(db *gorm.DB) *RetentionRepository { return &RetentionRepository{db: db} }

type retentionPolicyRow struct {
	WorkspaceID uuid.UUID `gorm:"type:uuid;primaryKey"`
	Revision    int64
	Policy      json.RawMessage `gorm:"type:jsonb"`
	PolicyHash  string
	EffectiveAt time.Time
	ScheduledAt time.Time
	UpdatedAt   time.Time
}

func (retentionPolicyRow) TableName() string { return "workspace_retention_policies" }

type retentionPreviewRow struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID           uuid.UUID
	PolicyRevision        int64
	Policy                json.RawMessage `gorm:"type:jsonb"`
	PolicyHash            string
	Counts                json.RawMessage `gorm:"type:jsonb"`
	EarliestDestructiveAt time.Time
	ExpiresAt             time.Time
	CreatedAt             time.Time
}

func (retentionPreviewRow) TableName() string { return "retention_previews" }

type retentionHoldRow struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID
	Kind        string
	RefID       string
	VersionID   *uuid.UUID
	UpdatedAt   time.Time
}

func (retentionHoldRow) TableName() string { return "retention_holds" }

type retentionReceiptRow struct {
	WorkspaceID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	RequestID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	PrincipalKind  string
	PrincipalID    string
	Action         string
	PayloadHash    string
	ResponseStatus int
	ResponseBody   json.RawMessage `gorm:"type:bytea"`
}

func (retentionReceiptRow) TableName() string { return "retention_mutation_receipts" }

func requireRetentionTransaction(ctx domain.Context, workspaceID uuid.UUID) (*gorm.DB, error) {
	tx, ok := ctx.Value(accessTransactionKey{}).(accessTransaction)
	if !ok || tx.workspace != workspaceID {
		return nil, domain.ErrAccessDenied
	}
	return tx.db.WithContext(ctx), nil
}

func defaultRetentionPolicy(workspaceID uuid.UUID) domain.RetentionPolicy {
	return domain.RetentionPolicy{WorkspaceID: workspaceID, Revision: 0, Policy: domain.DefaultRetentionPolicyInput(), PolicyHash: strings.Repeat("0", 64)}
}

func policyFromRow(row retentionPolicyRow) (domain.RetentionPolicy, error) {
	var input domain.RetentionPolicyInput
	if err := json.Unmarshal(row.Policy, &input); err != nil {
		return domain.RetentionPolicy{}, err
	}
	return domain.RetentionPolicy{WorkspaceID: row.WorkspaceID, Revision: row.Revision, Policy: input, PolicyHash: row.PolicyHash, EffectiveAt: row.EffectiveAt, ScheduledAt: row.ScheduledAt}, nil
}

func (r *RetentionRepository) CurrentPolicy(ctx domain.Context, workspaceID uuid.UUID) (domain.RetentionPolicy, error) {
	var row retentionPolicyRow
	err := accessDB(ctx, r.db).First(&row, "workspace_id=?", workspaceID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return defaultRetentionPolicy(workspaceID), nil
	}
	if err != nil {
		return domain.RetentionPolicy{}, err
	}
	return policyFromRow(row)
}

func (r *RetentionRepository) PreviewCounts(ctx domain.Context, workspaceID uuid.UUID, input domain.RetentionPolicyInput, now time.Time) (domain.RetentionCounts, error) {
	db := accessDB(ctx, r.db)
	out := domain.RetentionCounts{}
	if err := db.Raw(`SELECT count(*) FROM workspace_events e
		WHERE e.workspace_id=? AND e.occurred_at<?
		AND NOT EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=e.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='activity' AND (h.ref_id='*' OR h.ref_id=e.id::text)) OR (h.kind='scene' AND h.ref_id=e.scene_id::text)))
		AND NOT EXISTS (SELECT 1 FROM user_notifications n WHERE n.workspace_id=e.workspace_id AND n.event_id=e.id AND (n.created_at>=? OR EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=n.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='notification' AND (h.ref_id='*' OR h.ref_id=n.id::text)) OR (h.kind='scene' AND h.ref_id=n.scene_id::text)))))`, workspaceID, now.AddDate(0, 0, -input.ActivityDays), now.AddDate(0, 0, -input.NotificationDays)).Scan(&out.Activity).Error; err != nil {
		return out, err
	}
	if err := db.Raw(`SELECT count(*) FROM user_notifications n WHERE n.workspace_id=? AND n.created_at<? AND NOT EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=n.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='notification' AND (h.ref_id='*' OR h.ref_id=n.id::text)) OR (h.kind='scene' AND h.ref_id=n.scene_id::text)))`, workspaceID, now.AddDate(0, 0, -input.NotificationDays)).Scan(&out.Notifications).Error; err != nil {
		return out, err
	}
	if err := db.Raw(`SELECT count(*) FROM (SELECT id,workspace_id,scene_id,row_number() OVER(PARTITION BY scene_id ORDER BY created_at DESC,id DESC) AS n FROM scene_versions WHERE workspace_id=? AND kind='automatic' AND pinned=false) v WHERE v.n>? AND NOT EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=v.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='version' AND (h.ref_id='*' OR h.ref_id=v.id::text OR h.version_id=v.id)) OR (h.kind='scene' AND h.ref_id=v.scene_id::text)))`, workspaceID, input.AutomaticVersionLimit).Scan(&out.Versions).Error; err != nil {
		return out, err
	}
	if input.ManualVersionDays != nil {
		if err := db.Raw(`SELECT count(*) FROM scene_versions v WHERE v.workspace_id=? AND v.kind='manual' AND v.pinned=false AND v.created_at<? AND NOT EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=v.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='version' AND (h.ref_id='*' OR h.ref_id=v.id::text OR h.version_id=v.id)) OR (h.kind='scene' AND h.ref_id=v.scene_id::text)))`, workspaceID, now.AddDate(0, 0, -*input.ManualVersionDays)).Scan(&out.ManualVersions).Error; err != nil {
			return out, err
		}
	}
	if input.CommentsDays != nil {
		if err := db.Raw(`SELECT count(*) FROM comments c WHERE c.workspace_id=? AND c.deleted_at IS NULL AND c.created_at<? AND NOT EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=c.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='comment' AND (h.ref_id='*' OR h.ref_id=c.id::text)) OR (h.kind='scene' AND h.ref_id=c.scene_id::text)))`, workspaceID, now.AddDate(0, 0, -*input.CommentsDays)).Scan(&out.Comments).Error; err != nil {
			return out, err
		}
	}
	return out, nil
}

func (r *RetentionRepository) SavePreview(ctx domain.Context, value domain.RetentionPreview) error {
	policy, err := json.Marshal(value.Policy)
	if err != nil {
		return err
	}
	counts, err := json.Marshal(value.Counts)
	if err != nil {
		return err
	}
	row := retentionPreviewRow{ID: value.ID, WorkspaceID: value.WorkspaceID, PolicyRevision: value.PolicyRevision, Policy: policy, PolicyHash: value.PolicyHash, Counts: counts, EarliestDestructiveAt: value.EarliestDestructiveAt, ExpiresAt: value.ExpiresAt, CreatedAt: value.CreatedAt}
	return accessDB(ctx, r.db).Create(&row).Error
}

func (r *RetentionRepository) Preview(ctx domain.Context, workspaceID, id uuid.UUID) (domain.RetentionPreview, error) {
	var row retentionPreviewRow
	if err := accessDB(ctx, r.db).First(&row, "workspace_id=? AND id=?", workspaceID, id).Error; err != nil {
		return domain.RetentionPreview{}, accessError(err)
	}
	var input domain.RetentionPolicyInput
	var counts domain.RetentionCounts
	if err := json.Unmarshal(row.Policy, &input); err != nil {
		return domain.RetentionPreview{}, err
	}
	if err := json.Unmarshal(row.Counts, &counts); err != nil {
		return domain.RetentionPreview{}, err
	}
	return domain.RetentionPreview{ID: row.ID, WorkspaceID: row.WorkspaceID, PolicyRevision: row.PolicyRevision, Policy: input, PolicyHash: row.PolicyHash, Counts: counts, EarliestDestructiveAt: row.EarliestDestructiveAt, ExpiresAt: row.ExpiresAt, CreatedAt: row.CreatedAt}, nil
}

func (r *RetentionRepository) WithMutation(ctx domain.Context, principal domain.Principal, workspaceID, requestID uuid.UUID, action string, payload []byte, fn func(domain.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	db, err := requireRetentionTransaction(ctx, workspaceID)
	if err != nil || requestID == uuid.Nil || principal.ID == "" || !json.Valid(payload) || (action != "retention.schedule" && action != "retention.hold") {
		return domain.StoredFeatureResponse{}, domain.ErrAccessDenied
	}
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])
	var existing retentionReceiptRow
	lookup := db.First(&existing, "workspace_id=? AND request_id=?", workspaceID, requestID)
	if lookup.Error == nil {
		if existing.PrincipalKind != string(principal.Kind) || existing.PrincipalID != principal.ID || existing.Action != action || existing.PayloadHash != hash {
			return domain.StoredFeatureResponse{}, domain.ErrRequestIDReused
		}
		return domain.StoredFeatureResponse{Status: existing.ResponseStatus, Body: existing.ResponseBody}, nil
	}
	if !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
		return domain.StoredFeatureResponse{}, lookup.Error
	}
	response, err := fn(ctx)
	if err != nil {
		return domain.StoredFeatureResponse{}, err
	}
	if response.Status < 200 || response.Status > 299 || !json.Valid(response.Body) {
		return domain.StoredFeatureResponse{}, domain.ErrInvalidRetention
	}
	err = db.Create(&retentionReceiptRow{WorkspaceID: workspaceID, RequestID: requestID, PrincipalKind: string(principal.Kind), PrincipalID: principal.ID, Action: action, PayloadHash: hash, ResponseStatus: response.Status, ResponseBody: response.Body}).Error
	return response, err
}

func (r *RetentionRepository) Schedule(ctx domain.Context, workspaceID uuid.UUID, preview domain.RetentionPreview, expected int64, now time.Time) (domain.RetentionPolicy, error) {
	db, err := requireRetentionTransaction(ctx, workspaceID)
	if err != nil {
		return domain.RetentionPolicy{}, err
	}
	policyJSON, err := json.Marshal(preview.Policy)
	if err != nil {
		return domain.RetentionPolicy{}, err
	}
	effectiveAt := preview.EarliestDestructiveAt
	if minimum := now.Add(domain.MinimumRetentionActivationDelay); effectiveAt.Before(minimum) {
		effectiveAt = minimum
	}
	var current retentionPolicyRow
	find := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "workspace_id=?", workspaceID)
	if errors.Is(find.Error, gorm.ErrRecordNotFound) {
		if expected != 0 {
			return domain.RetentionPolicy{}, domain.ErrRevisionConflict
		}
		current = retentionPolicyRow{WorkspaceID: workspaceID, Revision: 1, Policy: policyJSON, PolicyHash: preview.PolicyHash, EffectiveAt: effectiveAt, ScheduledAt: now, UpdatedAt: now}
		if err = db.Create(&current).Error; err != nil {
			return domain.RetentionPolicy{}, err
		}
	} else if find.Error != nil {
		return domain.RetentionPolicy{}, find.Error
	} else {
		if current.Revision != expected {
			return domain.RetentionPolicy{}, domain.ErrRevisionConflict
		}
		current.Revision++
		current.Policy = policyJSON
		current.PolicyHash = preview.PolicyHash
		current.EffectiveAt = effectiveAt
		current.ScheduledAt = now
		current.UpdatedAt = now
		if err = db.Save(&current).Error; err != nil {
			return domain.RetentionPolicy{}, err
		}
	}
	// A revision-specific job is safe to retry; the worker re-reads the current row.
	payload, _ := json.Marshal(map[string]any{"workspaceId": workspaceID, "policyRevision": current.Revision})
	if err = db.Exec(`INSERT INTO feature_jobs(workspace_id,kind,payload,idempotency_key,available_at) VALUES (?, 'retention.purge', ?::jsonb, ?, ?) ON CONFLICT (workspace_id,kind,idempotency_key) DO NOTHING`, workspaceID, string(payload), fmt.Sprintf("policy:%d", current.Revision), effectiveAt).Error; err != nil {
		return domain.RetentionPolicy{}, err
	}
	return policyFromRow(current)
}

func (r *RetentionRepository) SetHold(ctx domain.Context, ref domain.RetentionReference, versionID *uuid.UUID, held bool, now time.Time) (domain.RetentionHold, error) {
	db, err := requireRetentionTransaction(ctx, ref.WorkspaceID)
	if err != nil {
		return domain.RetentionHold{}, err
	}
	var row retentionHoldRow
	q := db.Where("workspace_id=? AND kind=? AND ref_id=? AND version_key=coalesce(?,'00000000-0000-0000-0000-000000000000'::uuid)", ref.WorkspaceID, ref.Kind, ref.ID, versionID)
	find := q.First(&row)
	if held {
		if errors.Is(find.Error, gorm.ErrRecordNotFound) {
			row = retentionHoldRow{ID: uuid.New(), WorkspaceID: ref.WorkspaceID, Kind: ref.Kind, RefID: ref.ID, VersionID: versionID, UpdatedAt: now}
			if err = db.Create(&row).Error; err != nil {
				return domain.RetentionHold{}, err
			}
		} else if find.Error != nil {
			return domain.RetentionHold{}, find.Error
		} else {
			row.UpdatedAt = now
			if err = db.Model(&row).Update("updated_at", now).Error; err != nil {
				return domain.RetentionHold{}, err
			}
		}
	} else {
		if find.Error == nil {
			if err = db.Delete(&row).Error; err != nil {
				return domain.RetentionHold{}, err
			}
		} else if !errors.Is(find.Error, gorm.ErrRecordNotFound) {
			return domain.RetentionHold{}, find.Error
		}
		if row.ID == uuid.Nil {
			row = retentionHoldRow{ID: uuid.New(), WorkspaceID: ref.WorkspaceID, Kind: ref.Kind, RefID: ref.ID, VersionID: versionID, UpdatedAt: now}
		}
	}
	return domain.RetentionHold{ID: row.ID, Reference: ref, VersionID: versionID, Held: held, UpdatedAt: now}, nil
}

type retentionPurgeAuditRow struct {
	WorkspaceID    uuid.UUID
	PolicyRevision int64
	Kind           string
	SubjectID      string
	PurgedAt       time.Time
}

func (retentionPurgeAuditRow) TableName() string { return "retention_purge_audit" }

func hasRetentionHold(tx *gorm.DB, w uuid.UUID, kind, id string, sceneID *uuid.UUID) (bool, error) {
	var n int64
	query := tx.Table("retention_holds").Where("workspace_id=? AND ((kind='all' AND ref_id='*') OR (kind=? AND (ref_id='*' OR ref_id=? OR version_id::text=?)))", w, kind, id, id)
	if sceneID != nil {
		query = query.Or("workspace_id=? AND kind='scene' AND ref_id=?", w, sceneID.String())
	}
	err := query.Count(&n).Error
	return n > 0, err
}

func eventHasRemainingNotification(tx *gorm.DB, w uuid.UUID, eventID int64) (bool, error) {
	var n int64
	err := tx.Table("user_notifications").Where("workspace_id=? AND event_id=?", w, eventID).Count(&n).Error
	return n > 0, err
}
func addPurgeAudit(tx *gorm.DB, w uuid.UUID, revision int64, kind, id string, now time.Time) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&retentionPurgeAuditRow{WorkspaceID: w, PolicyRevision: revision, Kind: kind, SubjectID: id, PurgedAt: now}).Error
}

func (r *RetentionRepository) PurgeBatch(ctx domain.Context, now time.Time, limit int) (domain.PurgeResult, error) {
	if limit < 1 || limit > 1000 {
		return domain.PurgeResult{}, domain.ErrInvalidRetention
	}
	result := domain.PurgeResult{Items: []domain.RetentionPurgeItem{}}
	var due []retentionPolicyRow
	if err := r.db.WithContext(ctx).Table("workspace_retention_policies p").Select("p.*").Joins("JOIN workspace_feature_policies f ON f.workspace_id=p.workspace_id AND f.retention_enabled=true").Where("p.effective_at<=?", now).Order("p.effective_at,p.workspace_id").Limit(limit).Scan(&due).Error; err != nil {
		return result, err
	}
	if r.afterCandidateScan != nil {
		r.afterCandidateScan()
	}
	for _, candidate := range due {
		if len(result.Items) >= limit {
			break
		}
		workspaceResult := domain.PurgeResult{Items: []domain.RetentionPurgeItem{}}
		err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var lockedWorkspace struct{ ID uuid.UUID }
			if err := tx.Raw("SELECT id FROM workspaces WHERE id=? AND deleted_at IS NULL FOR UPDATE", candidate.WorkspaceID).Scan(&lockedWorkspace).Error; err != nil {
				return err
			}
			if lockedWorkspace.ID == uuid.Nil {
				return nil
			}
			var row retentionPolicyRow
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "workspace_id=?", candidate.WorkspaceID).Error; err != nil {
				return err
			}
			if row.Revision != candidate.Revision || row.EffectiveAt.After(now) {
				return nil
			}
			var retentionEnabled bool
			if err := tx.Raw("SELECT retention_enabled FROM workspace_feature_policies WHERE workspace_id=? FOR UPDATE", candidate.WorkspaceID).Scan(&retentionEnabled).Error; err != nil {
				return err
			}
			if !retentionEnabled {
				return nil
			}
			policy, err := policyFromRow(row)
			if err != nil {
				return err
			}
			workspaceResult.Scanned++
			remaining := limit - len(result.Items) - len(workspaceResult.Items)
			// Notifications are removed before activity because events cascade to inbox rows.
			var notifications []struct {
				ID      uuid.UUID
				SceneID uuid.UUID
			}
			if remaining > 0 {
				if err := tx.Raw(`SELECT n.id,n.scene_id FROM user_notifications n
					WHERE n.workspace_id=? AND n.created_at<?
					AND NOT EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=n.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='notification' AND (h.ref_id='*' OR h.ref_id=n.id::text)) OR (h.kind='scene' AND h.ref_id=n.scene_id::text)))
					ORDER BY n.created_at,n.id LIMIT ?`, row.WorkspaceID, now.AddDate(0, 0, -policy.Policy.NotificationDays), remaining).Scan(&notifications).Error; err != nil {
					return err
				}
				for _, notification := range notifications {
					held, holdErr := hasRetentionHold(tx, row.WorkspaceID, "notification", notification.ID.String(), &notification.SceneID)
					if holdErr != nil {
						return holdErr
					}
					if held {
						continue
					}
					if err := tx.Exec("DELETE FROM user_notifications WHERE workspace_id=? AND id=?", row.WorkspaceID, notification.ID).Error; err != nil {
						return err
					}
					if err := addPurgeAudit(tx, row.WorkspaceID, row.Revision, "notification", notification.ID.String(), now); err != nil {
						return err
					}
					workspaceResult.Purged.Notifications++
					workspaceResult.Items = append(workspaceResult.Items, domain.RetentionPurgeItem{Kind: "notification", ID: notification.ID.String()})
				}
			}
			remaining = limit - len(result.Items) - len(workspaceResult.Items)
			if remaining > 0 {
				var events []struct {
					ID      int64
					SceneID *uuid.UUID
				}
				if err := tx.Raw(`SELECT e.id,e.scene_id FROM workspace_events e
					WHERE e.workspace_id=? AND e.occurred_at<?
					AND NOT EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=e.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='activity' AND (h.ref_id='*' OR h.ref_id=e.id::text)) OR (h.kind='scene' AND h.ref_id=e.scene_id::text)))
					AND NOT EXISTS (SELECT 1 FROM user_notifications n WHERE n.workspace_id=e.workspace_id AND n.event_id=e.id)
					ORDER BY e.occurred_at,e.id LIMIT ?`, row.WorkspaceID, now.AddDate(0, 0, -policy.Policy.ActivityDays), remaining).Scan(&events).Error; err != nil {
					return err
				}
				for _, event := range events {
					key := strconv.FormatInt(event.ID, 10)
					held, holdErr := hasRetentionHold(tx, row.WorkspaceID, "activity", key, event.SceneID)
					if holdErr != nil {
						return holdErr
					}
					hasNotification, notificationErr := eventHasRemainingNotification(tx, row.WorkspaceID, event.ID)
					if notificationErr != nil {
						return notificationErr
					}
					if held || hasNotification {
						continue
					}
					if err := tx.Exec("DELETE FROM workspace_events WHERE workspace_id=? AND id=?", row.WorkspaceID, event.ID).Error; err != nil {
						return err
					}
					if err := addPurgeAudit(tx, row.WorkspaceID, row.Revision, "activity", key, now); err != nil {
						return err
					}
					workspaceResult.Purged.Activity++
					workspaceResult.Items = append(workspaceResult.Items, domain.RetentionPurgeItem{Kind: "activity", ID: key})
				}
			}
			remaining = limit - len(result.Items) - len(workspaceResult.Items)
			if remaining > 0 {
				var versions []struct {
					ID      uuid.UUID
					SceneID uuid.UUID
				}
				if err := tx.Raw(`SELECT v.id,v.scene_id FROM (SELECT id,workspace_id,scene_id,row_number() OVER(PARTITION BY scene_id ORDER BY created_at DESC,id DESC) n FROM scene_versions WHERE workspace_id=? AND kind='automatic' AND pinned=false) v
					WHERE v.n>? AND NOT EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=v.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='version' AND (h.ref_id='*' OR h.ref_id=v.id::text OR h.version_id=v.id)) OR (h.kind='scene' AND h.ref_id=v.scene_id::text)))
					ORDER BY v.id LIMIT ?`, row.WorkspaceID, policy.Policy.AutomaticVersionLimit, remaining).Scan(&versions).Error; err != nil {
					return err
				}
				for _, version := range versions {
					held, holdErr := hasRetentionHold(tx, row.WorkspaceID, "version", version.ID.String(), &version.SceneID)
					if holdErr != nil {
						return holdErr
					}
					if held {
						continue
					}
					if err := tx.Exec("DELETE FROM version_assets WHERE workspace_id=? AND version_id=?", row.WorkspaceID, version.ID).Error; err != nil {
						return err
					}
					if err := tx.Exec("DELETE FROM scene_versions WHERE workspace_id=? AND id=? AND kind='automatic' AND pinned=false", row.WorkspaceID, version.ID).Error; err != nil {
						return err
					}
					if err := addPurgeAudit(tx, row.WorkspaceID, row.Revision, "version", version.ID.String(), now); err != nil {
						return err
					}
					workspaceResult.Purged.Versions++
					workspaceResult.Items = append(workspaceResult.Items, domain.RetentionPurgeItem{Kind: "version", ID: version.ID.String()})
				}
			}
			remaining = limit - len(result.Items) - len(workspaceResult.Items)
			if remaining > 0 && policy.Policy.ManualVersionDays != nil {
				var versions []struct {
					ID      uuid.UUID
					SceneID uuid.UUID
				}
				if err := tx.Raw(`SELECT v.id,v.scene_id FROM scene_versions v
					WHERE v.workspace_id=? AND v.kind='manual' AND v.pinned=false AND v.created_at<?
					AND NOT EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=v.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='version' AND (h.ref_id='*' OR h.ref_id=v.id::text OR h.version_id=v.id)) OR (h.kind='scene' AND h.ref_id=v.scene_id::text)))
					ORDER BY v.created_at,v.id LIMIT ?`, row.WorkspaceID, now.AddDate(0, 0, -*policy.Policy.ManualVersionDays), remaining).Scan(&versions).Error; err != nil {
					return err
				}
				for _, version := range versions {
					held, holdErr := hasRetentionHold(tx, row.WorkspaceID, "version", version.ID.String(), &version.SceneID)
					if holdErr != nil {
						return holdErr
					}
					if held {
						continue
					}
					if err := tx.Exec("DELETE FROM version_assets WHERE workspace_id=? AND version_id=?", row.WorkspaceID, version.ID).Error; err != nil {
						return err
					}
					if err := tx.Exec("DELETE FROM scene_versions WHERE workspace_id=? AND id=? AND kind='manual' AND pinned=false", row.WorkspaceID, version.ID).Error; err != nil {
						return err
					}
					if err := addPurgeAudit(tx, row.WorkspaceID, row.Revision, "version", version.ID.String(), now); err != nil {
						return err
					}
					workspaceResult.Purged.ManualVersions++
					workspaceResult.Items = append(workspaceResult.Items, domain.RetentionPurgeItem{Kind: "manual_version", ID: version.ID.String()})
				}
			}
			remaining = limit - len(result.Items) - len(workspaceResult.Items)
			if remaining > 0 && policy.Policy.CommentsDays != nil {
				var comments []struct {
					ID      uuid.UUID
					SceneID uuid.UUID
				}
				if err := tx.Raw(`SELECT c.id,c.scene_id FROM comments c
					WHERE c.workspace_id=? AND c.deleted_at IS NULL AND c.created_at<?
					AND NOT EXISTS (SELECT 1 FROM retention_holds h WHERE h.workspace_id=c.workspace_id AND ((h.kind='all' AND h.ref_id='*') OR (h.kind='comment' AND (h.ref_id='*' OR h.ref_id=c.id::text)) OR (h.kind='scene' AND h.ref_id=c.scene_id::text)))
					ORDER BY c.created_at,c.id LIMIT ?`, row.WorkspaceID, now.AddDate(0, 0, -*policy.Policy.CommentsDays), remaining).Scan(&comments).Error; err != nil {
					return err
				}
				for _, comment := range comments {
					held, holdErr := hasRetentionHold(tx, row.WorkspaceID, "comment", comment.ID.String(), &comment.SceneID)
					if holdErr != nil {
						return holdErr
					}
					if held {
						continue
					}
					if err := tx.Table("comments").Where("workspace_id=? AND id=? AND deleted_at IS NULL", row.WorkspaceID, comment.ID).Updates(map[string]any{"body": "", "author_name": "Redacted", "deleted_at": now, "updated_at": now}).Error; err != nil {
						return err
					}
					if err := addPurgeAudit(tx, row.WorkspaceID, row.Revision, "comment", comment.ID.String(), now); err != nil {
						return err
					}
					workspaceResult.Purged.Comments++
					workspaceResult.Items = append(workspaceResult.Items, domain.RetentionPurgeItem{Kind: "comment", ID: comment.ID.String()})
				}
			}
			return nil
		})
		if err != nil {
			return result, err
		}
		result.Scanned += workspaceResult.Scanned
		result.Purged.Activity += workspaceResult.Purged.Activity
		result.Purged.Notifications += workspaceResult.Purged.Notifications
		result.Purged.Versions += workspaceResult.Purged.Versions
		result.Purged.ManualVersions += workspaceResult.Purged.ManualVersions
		result.Purged.Comments += workspaceResult.Purged.Comments
		result.Items = append(result.Items, workspaceResult.Items...)
	}
	return result, nil
}

var _ domain.RetentionRepository = (*RetentionRepository)(nil)
