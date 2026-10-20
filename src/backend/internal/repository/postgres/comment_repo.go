package postgres

import (
	"devtool/internal/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const commentMessagePageSize = 50

type CommentRepository struct{ db *gorm.DB }

func NewCommentRepository(db *gorm.DB) *CommentRepository { return &CommentRepository{db: db} }

type commentThreadRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	SceneID     uuid.UUID
	CreatorKind domain.PrincipalKind
	CreatorID   string
	CreatorName string
	ElementID   *string
	AnchorX     float64
	AnchorY     float64
	Status      string
	Revision    int64
	Subscribed  bool `gorm:"-"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (commentThreadRow) TableName() string { return "comment_threads" }

type commentRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	SceneID     uuid.UUID
	ThreadID    uuid.UUID
	AuthorKind  domain.PrincipalKind
	AuthorID    string
	AuthorName  string
	Body        string
	Revision    int64
	EditedAt    *time.Time
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (commentRow) TableName() string { return "comments" }

type commentSubscriptionRow struct {
	WorkspaceID uuid.UUID
	SceneID     uuid.UUID
	ThreadID    uuid.UUID
	UserID      uuid.UUID
}

func (commentSubscriptionRow) TableName() string { return "comment_subscriptions" }

func commentFromRow(row commentRow, principal domain.Principal) domain.Comment {
	return domain.Comment{
		ID: row.ID, WorkspaceID: row.WorkspaceID, SceneID: row.SceneID, ThreadID: row.ThreadID,
		AuthorKind: row.AuthorKind, AuthorID: row.AuthorID, AuthorName: row.AuthorName,
		Body: row.Body, Revision: row.Revision, Mine: row.AuthorKind == principal.Kind && row.AuthorID == principal.ID,
		EditedAt: row.EditedAt, DeletedAt: row.DeletedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func threadFromRow(row commentThreadRow) domain.CommentThread {
	return domain.CommentThread{
		ID: row.ID, WorkspaceID: row.WorkspaceID, SceneID: row.SceneID,
		CreatorKind: row.CreatorKind, CreatorID: row.CreatorID, CreatorName: row.CreatorName,
		Anchor: domain.CommentAnchor{ElementID: row.ElementID, X: row.AnchorX, Y: row.AnchorY},
		Status: row.Status, Revision: row.Revision, Subscribed: row.Subscribed,
		Messages: []domain.Comment{}, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func requireCommentTransaction(ctx domain.Context, workspaceID uuid.UUID) (*gorm.DB, error) {
	tx, ok := ctx.Value(accessTransactionKey{}).(accessTransaction)
	if !ok || tx.workspace != workspaceID {
		return nil, domain.ErrAccessDenied
	}
	return tx.db.WithContext(ctx), nil
}

func (r *CommentRepository) CreateThread(ctx domain.Context, thread domain.CommentThread, message domain.Comment, subscriber *uuid.UUID) error {
	db, err := requireCommentTransaction(ctx, thread.WorkspaceID)
	if err != nil {
		return err
	}
	row := commentThreadRow{ID: thread.ID, WorkspaceID: thread.WorkspaceID, SceneID: thread.SceneID, CreatorKind: thread.CreatorKind, CreatorID: thread.CreatorID, CreatorName: thread.CreatorName, ElementID: thread.Anchor.ElementID, AnchorX: thread.Anchor.X, AnchorY: thread.Anchor.Y, Status: thread.Status, Revision: thread.Revision}
	if err = db.Create(&row).Error; err != nil {
		return accessError(err)
	}
	comment := commentRow{ID: message.ID, WorkspaceID: message.WorkspaceID, SceneID: message.SceneID, ThreadID: message.ThreadID, AuthorKind: message.AuthorKind, AuthorID: message.AuthorID, AuthorName: message.AuthorName, Body: message.Body, Revision: message.Revision}
	if err = db.Create(&comment).Error; err != nil {
		return accessError(err)
	}
	if subscriber != nil {
		err = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&commentSubscriptionRow{WorkspaceID: thread.WorkspaceID, SceneID: thread.SceneID, ThreadID: thread.ID, UserID: *subscriber}).Error
	}
	return err
}

func (r *CommentRepository) AddReply(ctx domain.Context, message domain.Comment, subscriber *uuid.UUID) error {
	db, err := requireCommentTransaction(ctx, message.WorkspaceID)
	if err != nil {
		return err
	}
	var exists int
	result := db.Raw("SELECT 1 FROM comment_threads WHERE workspace_id=? AND scene_id=? AND id=?", message.WorkspaceID, message.SceneID, message.ThreadID).Scan(&exists)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrInaccessible
	}
	row := commentRow{ID: message.ID, WorkspaceID: message.WorkspaceID, SceneID: message.SceneID, ThreadID: message.ThreadID, AuthorKind: message.AuthorKind, AuthorID: message.AuthorID, AuthorName: message.AuthorName, Body: message.Body, Revision: message.Revision}
	if err = db.Create(&row).Error; err != nil {
		return accessError(err)
	}
	if subscriber != nil {
		err = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&commentSubscriptionRow{WorkspaceID: message.WorkspaceID, SceneID: message.SceneID, ThreadID: message.ThreadID, UserID: *subscriber}).Error
	}
	return err
}

func (r *CommentRepository) messagesPage(ctx domain.Context, ref domain.SceneRef, threadID uuid.UUID, principal domain.Principal, cursor *uuid.UUID, limit int) ([]domain.Comment, string, error) {
	db := accessDB(ctx, r.db).Where("workspace_id=? AND scene_id=? AND thread_id=?", ref.WorkspaceID, ref.SceneID, threadID)
	if cursor != nil {
		var cursorRow commentRow
		if err := db.Where("id=?", *cursor).First(&cursorRow).Error; err != nil {
			return nil, "", accessError(err)
		}
		db = accessDB(ctx, r.db).Where("workspace_id=? AND scene_id=? AND thread_id=? AND (created_at,id) > (?,?)", ref.WorkspaceID, ref.SceneID, threadID, cursorRow.CreatedAt, cursorRow.ID)
	}
	rows := []commentRow{}
	err := db.Order("created_at,id").Limit(limit + 1).Find(&rows).Error
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > limit {
		rows = rows[:limit]
		next = rows[len(rows)-1].ID.String()
	}
	items := make([]domain.Comment, len(rows))
	for i, row := range rows {
		items[i] = commentFromRow(row, principal)
	}
	return items, next, nil
}

func (r *CommentRepository) Thread(ctx domain.Context, ref domain.SceneRef, id uuid.UUID, principal domain.Principal) (domain.CommentThread, error) {
	return r.ThreadPage(ctx, ref, id, principal, nil, commentMessagePageSize)
}

func (r *CommentRepository) ThreadPage(ctx domain.Context, ref domain.SceneRef, id uuid.UUID, principal domain.Principal, cursor *uuid.UUID, limit int) (domain.CommentThread, error) {
	var row commentThreadRow
	db := accessDB(ctx, r.db)
	err := db.Where("workspace_id=? AND scene_id=? AND id=?", ref.WorkspaceID, ref.SceneID, id).First(&row).Error
	if err != nil {
		return domain.CommentThread{}, accessError(err)
	}
	if principal.Kind == domain.PrincipalUser && principal.UserID != nil {
		var count int64
		if err = db.Model(&commentSubscriptionRow{}).Where("workspace_id=? AND scene_id=? AND thread_id=? AND user_id=?", ref.WorkspaceID, ref.SceneID, id, *principal.UserID).Count(&count).Error; err != nil {
			return domain.CommentThread{}, err
		}
		row.Subscribed = count == 1
	}
	thread := threadFromRow(row)
	thread.Messages, thread.NextMessageCursor, err = r.messagesPage(ctx, ref, id, principal, cursor, limit)
	return thread, err
}

func (r *CommentRepository) Message(ctx domain.Context, ref domain.SceneRef, threadID, messageID uuid.UUID) (domain.Comment, error) {
	var row commentRow
	err := accessDB(ctx, r.db).Where("workspace_id=? AND scene_id=? AND thread_id=? AND id=?", ref.WorkspaceID, ref.SceneID, threadID, messageID).First(&row).Error
	if err != nil {
		return domain.Comment{}, accessError(err)
	}
	return commentFromRow(row, domain.Principal{}), nil
}

func (r *CommentRepository) UpdateMessage(ctx domain.Context, ref domain.SceneRef, threadID, messageID uuid.UUID, expected int64, body string, deleted bool) (domain.Comment, error) {
	db, err := requireCommentTransaction(ctx, ref.WorkspaceID)
	if err != nil {
		return domain.Comment{}, err
	}
	updates := map[string]any{"revision": gorm.Expr("revision+1"), "updated_at": gorm.Expr("clock_timestamp()")}
	if deleted {
		updates["body"] = ""
		updates["deleted_at"] = gorm.Expr("clock_timestamp()")
	} else {
		updates["body"] = body
		updates["edited_at"] = gorm.Expr("clock_timestamp()")
	}
	result := db.Model(&commentRow{}).Where("workspace_id=? AND scene_id=? AND thread_id=? AND id=? AND revision=? AND deleted_at IS NULL", ref.WorkspaceID, ref.SceneID, threadID, messageID, expected).Updates(updates)
	if result.Error != nil {
		return domain.Comment{}, result.Error
	}
	if result.RowsAffected != 1 {
		if _, findErr := r.Message(ctx, ref, threadID, messageID); findErr != nil {
			return domain.Comment{}, findErr
		}
		return domain.Comment{}, domain.ErrRevisionConflict
	}
	return r.Message(ctx, ref, threadID, messageID)
}

func (r *CommentRepository) UpdateThread(ctx domain.Context, ref domain.SceneRef, threadID uuid.UUID, expected int64, resolved *bool) (domain.CommentThread, error) {
	db, err := requireCommentTransaction(ctx, ref.WorkspaceID)
	if err != nil {
		return domain.CommentThread{}, err
	}
	updates := map[string]any{"revision": gorm.Expr("revision+1"), "updated_at": gorm.Expr("clock_timestamp()")}
	if resolved != nil {
		if *resolved {
			updates["status"] = domain.CommentStatusResolved
			updates["resolved_at"] = gorm.Expr("clock_timestamp()")
		} else {
			updates["status"] = domain.CommentStatusOpen
			updates["resolved_at"] = nil
		}
	}
	result := db.Model(&commentThreadRow{}).Where("workspace_id=? AND scene_id=? AND id=? AND revision=?", ref.WorkspaceID, ref.SceneID, threadID, expected).Updates(updates)
	if result.Error != nil {
		return domain.CommentThread{}, result.Error
	}
	if result.RowsAffected != 1 {
		if _, findErr := r.Thread(ctx, ref, threadID, domain.Principal{}); findErr != nil {
			return domain.CommentThread{}, findErr
		}
		return domain.CommentThread{}, domain.ErrRevisionConflict
	}
	return r.Thread(ctx, ref, threadID, domain.Principal{})
}

func (r *CommentRepository) SetSubscription(ctx domain.Context, ref domain.SceneRef, threadID, userID uuid.UUID, subscribed bool) error {
	db, err := requireCommentTransaction(ctx, ref.WorkspaceID)
	if err != nil {
		return err
	}
	row := commentSubscriptionRow{WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, ThreadID: threadID, UserID: userID}
	if subscribed {
		return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
	}
	return db.Where("workspace_id=? AND scene_id=? AND thread_id=? AND user_id=?", ref.WorkspaceID, ref.SceneID, threadID, userID).Delete(&commentSubscriptionRow{}).Error
}

func (r *CommentRepository) ListThreads(ctx domain.Context, ref domain.SceneRef, principal domain.Principal, cursor *uuid.UUID, limit int) ([]domain.CommentThread, error) {
	db := accessDB(ctx, r.db).Where("workspace_id=? AND scene_id=?", ref.WorkspaceID, ref.SceneID)
	if cursor != nil {
		var row commentThreadRow
		if err := db.Where("id=?", *cursor).First(&row).Error; err != nil {
			return nil, accessError(err)
		}
		db = accessDB(ctx, r.db).Where("workspace_id=? AND scene_id=? AND (created_at,id) < (?,?)", ref.WorkspaceID, ref.SceneID, row.CreatedAt, row.ID)
	}
	rows := []commentThreadRow{}
	if err := db.Order("created_at DESC,id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.CommentThread, 0, len(rows))
	for _, row := range rows {
		thread, err := r.Thread(ctx, ref, row.ID, principal)
		if err != nil {
			return nil, err
		}
		items = append(items, thread)
	}
	return items, nil
}

func (r *CommentRepository) RecentWrites(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, since time.Time) (int, error) {
	var count int64
	err := accessDB(ctx, r.db).Table("feature_mutation_receipts").Where("workspace_id=? AND scene_id=? AND principal_kind=? AND principal_id=? AND action IN ? AND created_at>=?", ref.WorkspaceID, ref.SceneID, principal.Kind, principal.ID, []domain.FeatureAction{domain.FeatureCommentRead, domain.FeatureCommentWrite}, since).Count(&count).Error
	return int(count), err
}

func (r *CommentRepository) DisplayName(ctx domain.Context, principal domain.Principal) (string, error) {
	if principal.Kind == domain.PrincipalPublic {
		if principal.DisplayName == "" {
			return "", domain.ErrAccessDenied
		}
		return principal.DisplayName, nil
	}
	if principal.Kind != domain.PrincipalUser || principal.UserID == nil {
		return "", domain.ErrAccessDenied
	}
	var name string
	if err := accessDB(ctx, r.db).Table("users").Where("id=? AND deleted_at IS NULL", *principal.UserID).Pluck("name", &name).Error; err != nil {
		return "", accessError(err)
	}
	if name == "" {
		return "", domain.ErrInaccessible
	}
	return name, nil
}

var _ domain.CommentRepository = (*CommentRepository)(nil)
