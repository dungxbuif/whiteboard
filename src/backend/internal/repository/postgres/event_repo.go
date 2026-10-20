package postgres

import (
	"bytes"
	"devtool/internal/domain"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	aclAuditSource     = "acl_audit"
	agentAuditSource   = "agent_audit"
	maxProjectionBatch = 100
)

type EventRepository struct{ db *gorm.DB }

func NewEventRepository(db *gorm.DB) *EventRepository { return &EventRepository{db: db} }

type workspaceEventRow struct {
	ID           int64
	WorkspaceID  uuid.UUID
	SceneID      *uuid.UUID
	ActorKind    domain.EventActorKind
	ActorID      string
	Kind         string
	SubjectID    string
	OccurredAt   time.Time
	SafeMetadata []byte `gorm:"type:jsonb"`
	Source       string
	SourceKey    string
}

func (workspaceEventRow) TableName() string { return "workspace_events" }

type workspaceEventOutboxRow struct {
	EventID     int64 `gorm:"primaryKey"`
	WorkspaceID uuid.UUID
	SceneID     *uuid.UUID
}

func (workspaceEventOutboxRow) TableName() string { return "workspace_event_outbox" }

type eventSourceCursor struct {
	Source     string `gorm:"primaryKey"`
	CursorTime time.Time
	CursorKey  string
	UpdatedAt  time.Time
}

func (eventSourceCursor) TableName() string { return "workspace_event_source_cursors" }

func eventFromRow(row workspaceEventRow) (domain.WorkspaceEvent, error) {
	metadata := map[string]any{}
	if len(row.SafeMetadata) > 0 {
		if err := json.Unmarshal(row.SafeMetadata, &metadata); err != nil {
			return domain.WorkspaceEvent{}, err
		}
	}
	return domain.WorkspaceEvent{ID: strconv.FormatInt(row.ID, 10), WorkspaceID: row.WorkspaceID, SceneID: row.SceneID, ActorKind: row.ActorKind, ActorID: row.ActorID, Kind: row.Kind, SubjectID: row.SubjectID, OccurredAt: row.OccurredAt, SafeMetadata: metadata, Source: row.Source, SourceKey: row.SourceKey}, nil
}

func (r *EventRepository) AppendEvent(ctx domain.Context, input domain.EventInput) (domain.WorkspaceEvent, error) {
	return r.appendWithDB(accessDB(ctx, r.db), input)
}

func (r *EventRepository) appendWithDB(db *gorm.DB, input domain.EventInput) (domain.WorkspaceEvent, error) {
	metadata := input.SafeMetadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return domain.WorkspaceEvent{}, err
	}
	when := input.OccurredAt
	if when.IsZero() {
		when = time.Now().UTC()
	}
	row := workspaceEventRow{WorkspaceID: input.WorkspaceID, SceneID: input.SceneID, ActorKind: input.ActorKind, ActorID: input.ActorID, Kind: input.Kind, SubjectID: input.SubjectID, OccurredAt: when, SafeMetadata: encoded, Source: input.Source, SourceKey: input.SourceKey}
	created := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "source"}, {Name: "source_key"}}, DoNothing: true}).Create(&row)
	if created.Error != nil {
		return domain.WorkspaceEvent{}, created.Error
	}
	if created.RowsAffected == 0 {
		var existing workspaceEventRow
		if err = db.Where("source=? AND source_key=?", input.Source, input.SourceKey).First(&existing).Error; err != nil {
			return domain.WorkspaceEvent{}, err
		}
		if existing.WorkspaceID != input.WorkspaceID || !sameUUIDPointer(existing.SceneID, input.SceneID) || existing.ActorKind != input.ActorKind || existing.ActorID != input.ActorID || existing.Kind != input.Kind || existing.SubjectID != input.SubjectID || !jsonEquivalent(existing.SafeMetadata, encoded) {
			return domain.WorkspaceEvent{}, domain.ErrEventSourceReused
		}
		return eventFromRow(existing)
	}
	outbox := workspaceEventOutboxRow{EventID: row.ID, WorkspaceID: row.WorkspaceID, SceneID: row.SceneID}
	if err = db.Create(&outbox).Error; err != nil {
		return domain.WorkspaceEvent{}, err
	}
	return eventFromRow(row)
}

func sameUUIDPointer(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func jsonEquivalent(left, right []byte) bool {
	var a, b any
	return json.Unmarshal(left, &a) == nil && json.Unmarshal(right, &b) == nil && bytes.Equal(mustCanonicalJSON(a), mustCanonicalJSON(b))
}

func mustCanonicalJSON(value any) []byte { encoded, _ := json.Marshal(value); return encoded }

func (r *EventRepository) ListWorkspaceEvents(ctx domain.Context, workspaceID uuid.UUID, cursor string, limit int) ([]domain.WorkspaceEvent, error) {
	rows := []workspaceEventRow{}
	query := accessDB(ctx, r.db).Where("workspace_id=?", workspaceID)
	if cursor != "" {
		query = query.Where("id>?", cursor)
	}
	if err := query.Order("id").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	events := make([]domain.WorkspaceEvent, 0, len(rows))
	for _, row := range rows {
		event, err := eventFromRow(row)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

type aclAuditProjection struct {
	ID          int64
	WorkspaceID uuid.UUID
	SceneID     *uuid.UUID
	ActorID     *uuid.UUID
	SubjectID   *uuid.UUID
	EventType   string
	ACLVersion  int64
	CreatedAt   time.Time
}

type agentAuditProjection struct {
	ID           uuid.UUID
	PrincipalID  string
	WorkspaceID  uuid.UUID
	SceneID      *uuid.UUID
	Tool         string
	Scope        string
	Outcome      string
	FailureClass string
	LatencyMs    int64
	CreatedAt    time.Time
}

func (r *EventRepository) ProjectAuditBatch(ctx domain.Context, limit int) (int, error) {
	if limit < 1 || limit > maxProjectionBatch {
		return 0, domain.ErrEventInput
	}
	projected := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		count, err := r.projectACL(tx, limit)
		if err != nil {
			return err
		}
		projected += count
		if projected == limit {
			return nil
		}
		count, err = r.projectAgent(tx, limit-projected)
		projected += count
		return err
	})
	return projected, err
}

func lockEventCursor(tx *gorm.DB, source string) (eventSourceCursor, error) {
	seed := eventSourceCursor{Source: source, CursorTime: time.Unix(0, 0).UTC(), CursorKey: ""}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
		return eventSourceCursor{}, err
	}
	var cursor eventSourceCursor
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cursor, "source=?", source).Error
	return cursor, err
}

func saveEventCursor(tx *gorm.DB, cursor eventSourceCursor) error {
	return tx.Model(&eventSourceCursor{}).Where("source=?", cursor.Source).Updates(map[string]any{"cursor_time": cursor.CursorTime, "cursor_key": cursor.CursorKey, "updated_at": time.Now().UTC()}).Error
}

func (r *EventRepository) projectACL(tx *gorm.DB, limit int) (int, error) {
	cursor, err := lockEventCursor(tx, aclAuditSource)
	if err != nil {
		return 0, err
	}
	rows := []aclAuditProjection{}
	lastID := int64(0)
	if cursor.CursorKey != "" {
		if lastID, err = strconv.ParseInt(cursor.CursorKey, 10, 64); err != nil {
			return 0, err
		}
	}
	if err = tx.Raw(`SELECT id,workspace_id,scene_id,actor_id,subject_id,event_type,acl_version,created_at FROM acl_audit_events WHERE id>? ORDER BY id LIMIT ?`, lastID, limit).Scan(&rows).Error; err != nil {
		return 0, err
	}
	for _, row := range rows {
		actorKind, actorID := domain.EventActorSystem, "system"
		if row.ActorID != nil {
			actorKind, actorID = domain.EventActorUser, row.ActorID.String()
		}
		subject := ""
		if row.SubjectID != nil {
			subject = row.SubjectID.String()
		}
		_, err = (&EventRepository{db: tx}).appendWithDB(tx, domain.EventInput{WorkspaceID: row.WorkspaceID, SceneID: row.SceneID, ActorKind: actorKind, ActorID: actorID, Kind: "acl." + row.EventType, SubjectID: subject, OccurredAt: row.CreatedAt, SafeMetadata: map[string]any{"eventType": row.EventType, "aclVersion": strconv.FormatInt(row.ACLVersion, 10)}, Source: aclAuditSource, SourceKey: strconv.FormatInt(row.ID, 10)})
		if err != nil {
			return 0, err
		}
		cursor.CursorTime, cursor.CursorKey = row.CreatedAt, strconv.FormatInt(row.ID, 10)
	}
	if len(rows) > 0 {
		if err = saveEventCursor(tx, cursor); err != nil {
			return 0, err
		}
	}
	return len(rows), nil
}

func (r *EventRepository) projectAgent(tx *gorm.DB, limit int) (int, error) {
	cursor, err := lockEventCursor(tx, agentAuditSource)
	if err != nil {
		return 0, err
	}
	rows := []agentAuditProjection{}
	if err = tx.Raw(`SELECT id,principal_id,workspace_id,scene_id,tool,scope,outcome,failure_class,latency_ms,created_at FROM agent_operation_audit WHERE (created_at,id::text)>(?,?) ORDER BY created_at,id::text LIMIT ?`, cursor.CursorTime, cursor.CursorKey, limit).Scan(&rows).Error; err != nil {
		return 0, err
	}
	for _, row := range rows {
		metadata := map[string]any{"tool": row.Tool, "scope": row.Scope, "outcome": row.Outcome, "latencyMs": row.LatencyMs}
		if row.FailureClass != "" {
			metadata["failureClass"] = row.FailureClass
		}
		_, err = (&EventRepository{db: tx}).appendWithDB(tx, domain.EventInput{WorkspaceID: row.WorkspaceID, SceneID: row.SceneID, ActorKind: domain.EventActorAgent, ActorID: row.PrincipalID, Kind: "agent." + row.Tool + "." + row.Outcome, OccurredAt: row.CreatedAt, SafeMetadata: metadata, Source: agentAuditSource, SourceKey: row.ID.String()})
		if err != nil {
			return 0, err
		}
		cursor.CursorTime, cursor.CursorKey = row.CreatedAt, row.ID.String()
	}
	if len(rows) > 0 {
		if err = saveEventCursor(tx, cursor); err != nil {
			return 0, err
		}
	}
	return len(rows), nil
}
