package postgres

import (
	"devtool/internal/domain"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotificationRepository struct{ db *gorm.DB }

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

type notificationRow struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	WorkspaceID   uuid.UUID
	SceneID       uuid.UUID
	EventID       int64
	Kind          string
	ActorName     string
	ThreadID      uuid.UUID
	ReadAt        *time.Time
	ReadRequestID *uuid.UUID
	CreatedAt     time.Time
}

func (notificationRow) TableName() string { return "user_notifications" }

func notificationFromRow(row notificationRow) domain.Notification {
	return domain.Notification{ID: row.ID, UserID: row.UserID, WorkspaceID: row.WorkspaceID, SceneID: row.SceneID, EventID: row.EventID, Kind: row.Kind, ActorName: row.ActorName, ThreadID: row.ThreadID, ReadAt: row.ReadAt, CreatedAt: row.CreatedAt}
}

func (r *NotificationRepository) Event(ctx domain.Context, id int64) (domain.WorkspaceEvent, error) {
	var row workspaceEventRow
	if err := accessDB(ctx, r.db).Where("id=?", id).First(&row).Error; err != nil {
		return domain.WorkspaceEvent{}, accessError(err)
	}
	return eventFromRow(row)
}

func (r *NotificationRepository) Subscribers(ctx domain.Context, ref domain.SceneRef, threadID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := accessDB(ctx, r.db).Model(&commentSubscriptionRow{}).
		Where("workspace_id=? AND scene_id=? AND thread_id=?", ref.WorkspaceID, ref.SceneID, threadID).
		Pluck("user_id", &ids).Error
	return ids, err
}

func (r *NotificationRepository) ActorName(ctx domain.Context, event domain.WorkspaceEvent) (string, error) {
	if event.ActorKind == domain.EventActorPublic {
		return "Guest", nil
	}
	actorID, err := uuid.Parse(event.ActorID)
	if err != nil {
		return "System", nil
	}
	var name string
	if err = accessDB(ctx, r.db).Raw("SELECT name FROM users WHERE id=?", actorID).Scan(&name).Error; err != nil {
		return "", err
	}
	if name == "" {
		name = "Former member"
	}
	return name, nil
}

func (r *NotificationRepository) Create(ctx domain.Context, value domain.Notification) error {
	db, err := requireCommentTransaction(ctx, value.WorkspaceID)
	if err != nil {
		return err
	}
	row := notificationRow{ID: value.ID, UserID: value.UserID, WorkspaceID: value.WorkspaceID, SceneID: value.SceneID, EventID: value.EventID, Kind: value.Kind, ActorName: value.ActorName, ThreadID: value.ThreadID}
	return db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "event_id"}}, DoNothing: true}).Create(&row).Error
}

func (r *NotificationRepository) List(ctx domain.Context, userID uuid.UUID, cursor *uuid.UUID, limit int) ([]domain.Notification, error) {
	db := accessDB(ctx, r.db).Where("user_id=?", userID)
	if cursor != nil {
		var cursorRow notificationRow
		if err := db.Where("id=?", *cursor).First(&cursorRow).Error; err != nil {
			return nil, accessError(err)
		}
		db = accessDB(ctx, r.db).Where("user_id=? AND (created_at,id) < (?,?)", userID, cursorRow.CreatedAt, cursorRow.ID)
	}
	var rows []notificationRow
	if err := db.Order("created_at DESC,id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Notification, len(rows))
	for index, row := range rows {
		items[index] = notificationFromRow(row)
	}
	return items, nil
}

func (r *NotificationRepository) Get(ctx domain.Context, userID, id uuid.UUID) (domain.Notification, error) {
	var row notificationRow
	if err := accessDB(ctx, r.db).Where("user_id=? AND id=?", userID, id).First(&row).Error; err != nil {
		return domain.Notification{}, accessError(err)
	}
	return notificationFromRow(row), nil
}

func (r *NotificationRepository) MarkRead(ctx domain.Context, userID, id, requestID uuid.UUID) error {
	notification, err := r.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	db, err := requireCommentTransaction(ctx, notification.WorkspaceID)
	if err != nil {
		return err
	}
	result := db.Model(&notificationRow{}).Where("user_id=? AND id=? AND (read_request_id IS NULL OR read_request_id=?)", userID, id, requestID).
		Updates(map[string]any{"read_at": gorm.Expr("COALESCE(read_at,clock_timestamp())"), "read_request_id": requestID})
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrRequestIDReused
		}
		return accessError(result.Error)
	}
	if result.RowsAffected != 1 {
		return domain.ErrRequestIDReused
	}
	return nil
}
