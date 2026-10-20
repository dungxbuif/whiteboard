package postgres

import (
	"database/sql"
	"devtool/internal/domain"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityRepository struct{ db *gorm.DB }

func NewActivityRepository(db *gorm.DB) *ActivityRepository { return &ActivityRepository{db: db} }

type activityRow struct {
	ID           int64
	WorkspaceID  uuid.UUID
	SceneID      *uuid.UUID
	ActorKind    domain.EventActorKind
	ActorName    string
	Kind         string
	SubjectID    string
	OccurredAt   time.Time
	SafeMetadata []byte `gorm:"type:jsonb"`
}

const activityActorJoin = `LEFT JOIN users actor ON actor.id=CASE
 WHEN e.actor_kind='user' AND e.actor_id ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
 THEN e.actor_id::uuid ELSE NULL END AND actor.deleted_at IS NULL`

func (r *ActivityRepository) authorizedActivity(ctx domain.Context, userID, workspaceID uuid.UUID, sceneID *uuid.UUID, owner bool) *gorm.DB {
	query := accessDB(ctx, r.db).Table("workspace_events AS e").
		Joins("JOIN workspaces w ON w.id=e.workspace_id AND w.deleted_at IS NULL").
		Joins("JOIN workspace_members m ON m.workspace_id=e.workspace_id AND m.user_id=?", userID).
		Joins("JOIN users principal_user ON principal_user.id=m.user_id AND principal_user.deleted_at IS NULL").
		Joins("LEFT JOIN scenes s ON s.workspace_id=e.workspace_id AND s.id=e.scene_id").
		Joins(activityActorJoin).
		Where("e.workspace_id=?", workspaceID).
		Where("e.scene_id IS NULL OR (s.id IS NOT NULL AND s.deleted_at IS NULL)")
	if sceneID != nil {
		query = query.Where("e.scene_id=?", *sceneID)
	}
	if !owner {
		query = query.Where("e.kind NOT LIKE 'security.%' AND e.kind NOT LIKE 'acl.%'")
	}
	return query
}

func activityFromRow(row activityRow) (domain.ActivityItem, error) {
	metadata := map[string]any{}
	if len(row.SafeMetadata) > 0 {
		if err := json.Unmarshal(row.SafeMetadata, &metadata); err != nil {
			return domain.ActivityItem{}, err
		}
	}
	return domain.ActivityItem{ID: strconv.FormatInt(row.ID, 10), WorkspaceID: row.WorkspaceID, SceneID: row.SceneID, ActorKind: row.ActorKind, ActorName: row.ActorName, Kind: row.Kind, SubjectID: row.SubjectID, OccurredAt: row.OccurredAt, SafeMetadata: metadata}, nil
}

func (r *ActivityRepository) ListActivity(ctx domain.Context, userID, workspaceID uuid.UUID, sceneID *uuid.UUID, owner bool, cursor *domain.ActivityCursor, limit int) ([]domain.ActivityItem, error) {
	query := r.authorizedActivity(ctx, userID, workspaceID, sceneID, owner).
		Select(`e.id,e.workspace_id,e.scene_id,e.actor_kind,
 CASE e.actor_kind WHEN 'user' THEN COALESCE(NULLIF(actor.name,''),'Former member') WHEN 'public' THEN 'Guest' WHEN 'agent' THEN 'Agent' ELSE 'System' END AS actor_name,
 e.kind,e.subject_id,e.occurred_at,e.safe_metadata`)
	if cursor != nil {
		id, err := strconv.ParseInt(cursor.ID, 10, 64)
		if err != nil || id < 1 {
			return nil, domain.ErrInvalidActivity
		}
		query = query.Where("(e.occurred_at,e.id) < (?,?)", cursor.OccurredAt, id)
	}
	rows := []activityRow{}
	if err := query.Order("e.occurred_at DESC,e.id DESC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.ActivityItem, 0, len(rows))
	for _, row := range rows {
		item, err := activityFromRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ActivityRepository) ActivityRetainedSince(ctx domain.Context, userID, workspaceID uuid.UUID, sceneID *uuid.UUID, owner bool) (time.Time, error) {
	var retained sql.NullTime
	if err := r.authorizedActivity(ctx, userID, workspaceID, sceneID, owner).Select("MIN(e.occurred_at)").Scan(&retained).Error; err != nil {
		return time.Time{}, err
	}
	if !retained.Valid {
		return time.Time{}, nil
	}
	return retained.Time.UTC(), nil
}

var _ domain.ActivityRepository = (*ActivityRepository)(nil)
