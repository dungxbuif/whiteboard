package postgres

import (
	"context"
	"devtool/internal/domain"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FeatureRepository struct {
	db *gorm.DB
}

func NewFeatureRepository(db *gorm.DB) *FeatureRepository {
	return &FeatureRepository{db: db}
}

func (r *FeatureRepository) Policy(ctx context.Context, workspaceID uuid.UUID) (domain.WorkspaceFeaturePolicy, error) {
	var policy domain.WorkspaceFeaturePolicy
	err := accessDB(ctx, r.db).First(&policy, "workspace_id=?", workspaceID).Error
	return policy, accessError(err)
}

func (r *FeatureRepository) PublicCommentsAllowed(ctx context.Context, principal domain.Principal, ref domain.SceneRef) (bool, error) {
	if principal.Kind != domain.PrincipalPublic {
		return false, domain.ErrAccessDenied
	}
	grantID, err := uuid.Parse(principal.GrantID)
	if err != nil {
		return false, domain.ErrAccessDenied
	}
	var row struct {
		AllowComments bool
		Revision      int64
	}
	result := accessDB(ctx, r.db).Raw(`
		SELECT allow_comments,revision
		FROM share_links
		WHERE id=? AND workspace_id=? AND scene_id=? AND active=true
		  AND (expires_at IS NULL OR expires_at>clock_timestamp())
	`, grantID, ref.WorkspaceID, ref.SceneID).Scan(&row)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected != 1 || strconv.FormatInt(row.Revision, 10) != principal.GrantVersion {
		return false, domain.ErrInaccessible
	}
	return row.AllowComments, nil
}

func (r *FeatureRepository) UpdatePolicy(ctx context.Context, workspaceID uuid.UUID, expectedRevision int64, patch domain.WorkspaceFeaturePolicyPatch) (domain.WorkspaceFeaturePolicy, error) {
	updates := map[string]any{}
	setBool := func(column string, value *bool) {
		if value != nil {
			updates[column] = *value
		}
	}
	setBool("member_comments", patch.MemberComments)
	setBool("member_history", patch.MemberHistory)
	setBool("comments_enabled", patch.CommentsEnabled)
	setBool("activity_enabled", patch.ActivityEnabled)
	setBool("versions_enabled", patch.VersionsEnabled)
	setBool("libraries_enabled", patch.LibrariesEnabled)
	setBool("collections_enabled", patch.CollectionsEnabled)
	setBool("presentation_enabled", patch.PresentationEnabled)
	setBool("screen_share_enabled", patch.ScreenShareEnabled)
	setBool("enterprise_identity_enabled", patch.EnterpriseIdentityEnabled)
	setBool("retention_enabled", patch.RetentionEnabled)
	if len(updates) == 0 {
		return domain.WorkspaceFeaturePolicy{}, domain.ErrAccessDenied
	}
	db := accessDB(ctx, r.db)
	result := db.Model(&domain.WorkspaceFeaturePolicy{}).
		Where("workspace_id=? AND revision=?", workspaceID, expectedRevision).
		Updates(updates)
	if result.Error != nil {
		return domain.WorkspaceFeaturePolicy{}, result.Error
	}
	if result.RowsAffected != 1 {
		return domain.WorkspaceFeaturePolicy{}, domain.ErrRevisionConflict
	}
	return r.Policy(ctx, workspaceID)
}

type featureReceipt struct {
	WorkspaceID    uuid.UUID       `gorm:"type:uuid;primaryKey"`
	RequestID      uuid.UUID       `gorm:"type:uuid;primaryKey"`
	SceneID        *uuid.UUID      `gorm:"type:uuid"`
	Action         string          `gorm:"not null"`
	PrincipalKind  string          `gorm:"not null"`
	PrincipalID    string          `gorm:"not null"`
	PayloadHash    string          `gorm:"not null"`
	ResponseStatus int             `gorm:"not null"`
	ResponseBody   json.RawMessage `gorm:"type:bytea;not null"`
}

func (featureReceipt) TableName() string {
	return "feature_mutation_receipts"
}

func (r *FeatureRepository) WithMutation(ctx context.Context, principal domain.Principal, ref domain.SceneRef, action domain.FeatureAction, requestID uuid.UUID, payloadHash string, fn func(context.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	transaction, ok := ctx.Value(accessTransactionKey{}).(accessTransaction)
	if !ok || transaction.workspace != ref.WorkspaceID || requestID == uuid.Nil || principal.ID == "" || len(payloadHash) != domain.FeatureRequestHashLength {
		return domain.StoredFeatureResponse{}, domain.ErrAccessDenied
	}
	if _, err := hex.DecodeString(payloadHash); err != nil {
		return domain.StoredFeatureResponse{}, domain.ErrAccessDenied
	}
	db := accessDB(ctx, r.db)
	var existing featureReceipt
	result := db.First(&existing, "workspace_id=? AND request_id=?", ref.WorkspaceID, requestID)
	if result.Error == nil {
		sameScene := (existing.SceneID == nil && ref.SceneID == uuid.Nil) ||
			(existing.SceneID != nil && *existing.SceneID == ref.SceneID)
		if !sameScene || existing.Action != string(action) ||
			existing.PrincipalKind != string(principal.Kind) ||
			existing.PrincipalID != principal.ID ||
			existing.PayloadHash != payloadHash {
			return domain.StoredFeatureResponse{}, domain.ErrRequestIDReused
		}
		return domain.StoredFeatureResponse{Status: existing.ResponseStatus, Body: existing.ResponseBody}, nil
	}
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return domain.StoredFeatureResponse{}, result.Error
	}
	response, err := fn(ctx)
	if err != nil {
		return domain.StoredFeatureResponse{}, err
	}
	if response.Status < 100 || response.Status > 599 || !json.Valid(response.Body) {
		return domain.StoredFeatureResponse{}, domain.ErrAccessDenied
	}
	var sceneID *uuid.UUID
	if ref.SceneID != uuid.Nil {
		id := ref.SceneID
		sceneID = &id
	}
	receipt := featureReceipt{
		WorkspaceID: ref.WorkspaceID, RequestID: requestID, SceneID: sceneID,
		Action: string(action), PrincipalKind: string(principal.Kind),
		PrincipalID: principal.ID, PayloadHash: payloadHash,
		ResponseStatus: response.Status, ResponseBody: response.Body,
	}
	if err := db.Create(&receipt).Error; err != nil {
		return domain.StoredFeatureResponse{}, err
	}
	return response, nil
}
