package postgres

import (
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CollectionRepository struct{ db *gorm.DB }

func NewCollectionRepository(db *gorm.DB) *CollectionRepository { return &CollectionRepository{db: db} }

type collectionRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	CreatorID   uuid.UUID
	Name        string
	Visibility  domain.CollectionVisibility
	Revision    int64
	SceneCount  int `gorm:"column:scene_count;->"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (collectionRow) TableName() string { return "collections" }

type collectionSceneRow struct {
	WorkspaceID  uuid.UUID
	CollectionID uuid.UUID
	SceneID      uuid.UUID
	Position     int
	CreatedAt    time.Time
}

func (collectionSceneRow) TableName() string { return "collection_scenes" }

func collectionFromRow(row collectionRow) domain.Collection {
	return domain.Collection{ID: row.ID, WorkspaceID: row.WorkspaceID, CreatorID: row.CreatorID, Name: row.Name, Visibility: row.Visibility, Revision: row.Revision, SceneCount: row.SceneCount, Scenes: []domain.CollectionScene{}, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func requireCollectionTransaction(ctx domain.Context, workspaceID uuid.UUID) (*gorm.DB, error) {
	tx, ok := ctx.Value(accessTransactionKey{}).(accessTransaction)
	if !ok || tx.workspace != workspaceID {
		return nil, domain.ErrAccessDenied
	}
	return tx.db.WithContext(ctx), nil
}

func (r *CollectionRepository) WithMutation(ctx domain.Context, principal domain.Principal, workspaceID, requestID uuid.UUID, payload []byte, fn func(domain.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	db, err := requireCollectionTransaction(ctx, workspaceID)
	if err != nil || requestID == uuid.Nil || principal.ID == "" || !json.Valid(payload) {
		return domain.StoredFeatureResponse{}, domain.ErrAccessDenied
	}
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])
	var existing featureReceipt
	result := db.First(&existing, "workspace_id=? AND request_id=?", workspaceID, requestID)
	if result.Error == nil {
		if existing.SceneID != nil || existing.Action != string(domain.FeatureCollectionManage) || existing.PrincipalKind != string(principal.Kind) || existing.PrincipalID != principal.ID || existing.PayloadHash != hash {
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
	receipt := featureReceipt{WorkspaceID: workspaceID, RequestID: requestID, Action: string(domain.FeatureCollectionManage), PrincipalKind: string(principal.Kind), PrincipalID: principal.ID, PayloadHash: hash, ResponseStatus: response.Status, ResponseBody: response.Body}
	if err = db.Create(&receipt).Error; err != nil {
		return domain.StoredFeatureResponse{}, err
	}
	return response, nil
}

func (r *CollectionRepository) Create(ctx domain.Context, value domain.Collection) (domain.Collection, error) {
	db, err := requireCollectionTransaction(ctx, value.WorkspaceID)
	if err != nil {
		return domain.Collection{}, err
	}
	row := collectionRow{ID: value.ID, WorkspaceID: value.WorkspaceID, CreatorID: value.CreatorID, Name: value.Name, Visibility: value.Visibility, Revision: value.Revision}
	if err = db.Create(&row).Error; err != nil {
		return domain.Collection{}, accessError(err)
	}
	return r.Get(ctx, value.WorkspaceID, value.ID, value.CreatorID, false)
}

func (r *CollectionRepository) visibleRow(ctx domain.Context, workspaceID, collectionID, userID uuid.UUID) (collectionRow, error) {
	var row collectionRow
	err := accessDB(ctx, r.db).Where("workspace_id=? AND id=? AND (visibility='workspace' OR creator_id=?)", workspaceID, collectionID, userID).First(&row).Error
	return row, accessError(err)
}

func (r *CollectionRepository) scenes(ctx domain.Context, workspaceID, collectionID, userID uuid.UUID) ([]domain.CollectionScene, error) {
	rows := []struct {
		ID        uuid.UUID
		PublicID  *string
		Title     string
		Position  int
		UpdatedAt time.Time
	}{}
	err := accessDB(ctx, r.db).Raw(`
		SELECT s.id,s.public_id,s.title,cs.position,s.updated_at
		FROM collection_scenes cs
		JOIN scenes s ON s.workspace_id=cs.workspace_id AND s.id=cs.scene_id AND s.deleted_at IS NULL
		JOIN workspace_members wm ON wm.workspace_id=s.workspace_id AND wm.user_id=?
		JOIN users u ON u.id=wm.user_id AND u.deleted_at IS NULL
		WHERE cs.workspace_id=? AND cs.collection_id=?
		ORDER BY cs.position,s.id
	`, userID, workspaceID, collectionID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]domain.CollectionScene, len(rows))
	for i, row := range rows {
		publicID := ""
		if row.PublicID != nil {
			publicID = *row.PublicID
		}
		items[i] = domain.CollectionScene{ID: row.ID, PublicID: publicID, Title: row.Title, Position: row.Position, UpdatedAt: row.UpdatedAt}
	}
	return items, nil
}

func (r *CollectionRepository) Get(ctx domain.Context, workspaceID, collectionID, userID uuid.UUID, owner bool) (domain.Collection, error) {
	_ = owner // Owners intentionally cannot read another user's private collection.
	row, err := r.visibleRow(ctx, workspaceID, collectionID, userID)
	if err != nil {
		return domain.Collection{}, err
	}
	value := collectionFromRow(row)
	value.Scenes, err = r.scenes(ctx, workspaceID, collectionID, userID)
	value.SceneCount = len(value.Scenes)
	return value, err
}

func (r *CollectionRepository) List(ctx domain.Context, workspaceID, userID uuid.UUID, owner bool, cursor *uuid.UUID, limit int) ([]domain.Collection, error) {
	_ = owner
	db := accessDB(ctx, r.db)
	var cursorRow collectionRow
	if cursor != nil {
		if err := db.Where("workspace_id=? AND id=? AND (visibility='workspace' OR creator_id=?)", workspaceID, *cursor, userID).First(&cursorRow).Error; err != nil {
			return nil, accessError(err)
		}
	}
	query := `
		SELECT c.*,
		 (SELECT count(*) FROM collection_scenes cs
		  JOIN scenes s ON s.workspace_id=cs.workspace_id AND s.id=cs.scene_id AND s.deleted_at IS NULL
		  JOIN workspace_members wm ON wm.workspace_id=s.workspace_id AND wm.user_id=?
		  JOIN users u ON u.id=wm.user_id AND u.deleted_at IS NULL
		  WHERE cs.workspace_id=c.workspace_id AND cs.collection_id=c.id) AS scene_count
		FROM collections c
		WHERE c.workspace_id=? AND (c.visibility='workspace' OR c.creator_id=?)`
	args := []any{userID, workspaceID, userID}
	if cursor != nil {
		query += " AND (c.created_at,c.id) < (?,?)"
		args = append(args, cursorRow.CreatedAt, cursorRow.ID)
	}
	query += " ORDER BY c.created_at DESC,c.id DESC LIMIT ?"
	args = append(args, limit)
	rows := []collectionRow{}
	if err := db.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Collection, len(rows))
	for i, row := range rows {
		items[i] = collectionFromRow(row)
	}
	return items, nil
}

func (r *CollectionRepository) mutationMiss(ctx domain.Context, workspaceID, collectionID, userID uuid.UUID) error {
	if _, err := r.visibleRow(ctx, workspaceID, collectionID, userID); err != nil {
		return err
	}
	return domain.ErrRevisionConflict
}

func (r *CollectionRepository) Update(ctx domain.Context, workspaceID, collectionID, userID uuid.UUID, expected int64, name string, visibility domain.CollectionVisibility) (domain.Collection, error) {
	db, err := requireCollectionTransaction(ctx, workspaceID)
	if err != nil {
		return domain.Collection{}, err
	}
	result := db.Model(&collectionRow{}).Where("workspace_id=? AND id=? AND revision=? AND (creator_id=? OR (visibility='workspace' AND ?='workspace'))", workspaceID, collectionID, expected, userID, visibility).Updates(map[string]any{"name": name, "visibility": visibility, "revision": gorm.Expr("revision+1"), "updated_at": gorm.Expr("clock_timestamp()")})
	if result.Error != nil {
		return domain.Collection{}, accessError(result.Error)
	}
	if result.RowsAffected != 1 {
		return domain.Collection{}, r.mutationMiss(ctx, workspaceID, collectionID, userID)
	}
	return r.Get(ctx, workspaceID, collectionID, userID, false)
}

func (r *CollectionRepository) Delete(ctx domain.Context, workspaceID, collectionID, userID uuid.UUID, expected int64, owner bool) error {
	db, err := requireCollectionTransaction(ctx, workspaceID)
	if err != nil {
		return err
	}
	result := db.Where("workspace_id=? AND id=? AND revision=? AND (visibility='workspace' OR creator_id=? OR ?)", workspaceID, collectionID, expected, userID, owner).Delete(&collectionRow{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var row collectionRow
	find := db.Where("workspace_id=? AND id=?", workspaceID, collectionID).First(&row)
	if find.Error != nil {
		return accessError(find.Error)
	}
	if row.Visibility == domain.CollectionPrivate && row.CreatorID != userID && !owner {
		return domain.ErrInaccessible
	}
	return domain.ErrRevisionConflict
}

func (r *CollectionRepository) ensureMutable(ctx domain.Context, workspaceID, collectionID, userID uuid.UUID, expected int64) (*gorm.DB, error) {
	db, err := requireCollectionTransaction(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	var row collectionRow
	if err = db.Where("workspace_id=? AND id=? AND (visibility='workspace' OR creator_id=?)", workspaceID, collectionID, userID).First(&row).Error; err != nil {
		return nil, accessError(err)
	}
	if row.Revision != expected {
		return nil, domain.ErrRevisionConflict
	}
	return db, nil
}

func (r *CollectionRepository) bump(ctx domain.Context, workspaceID, collectionID uuid.UUID, expected int64) error {
	db := accessDB(ctx, r.db)
	result := db.Model(&collectionRow{}).Where("workspace_id=? AND id=? AND revision=?", workspaceID, collectionID, expected).Updates(map[string]any{"revision": gorm.Expr("revision+1"), "updated_at": gorm.Expr("clock_timestamp()")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrRevisionConflict
	}
	return nil
}

func (r *CollectionRepository) AddScene(ctx domain.Context, workspaceID, collectionID, userID, sceneID uuid.UUID, expected int64) (domain.Collection, error) {
	db, err := r.ensureMutable(ctx, workspaceID, collectionID, userID, expected)
	if err != nil {
		return domain.Collection{}, err
	}
	var count int64
	if err = db.Model(&collectionSceneRow{}).Where("workspace_id=? AND collection_id=? AND scene_id=?", workspaceID, collectionID, sceneID).Count(&count).Error; err != nil {
		return domain.Collection{}, err
	}
	if count != 0 {
		return domain.Collection{}, domain.ErrRevisionConflict
	}
	var position int
	if err = db.Raw("SELECT COALESCE(max(position)+1,0) FROM collection_scenes WHERE workspace_id=? AND collection_id=?", workspaceID, collectionID).Scan(&position).Error; err != nil {
		return domain.Collection{}, err
	}
	if err = db.Create(&collectionSceneRow{WorkspaceID: workspaceID, CollectionID: collectionID, SceneID: sceneID, Position: position}).Error; err != nil {
		return domain.Collection{}, accessError(err)
	}
	if err = r.bump(ctx, workspaceID, collectionID, expected); err != nil {
		return domain.Collection{}, err
	}
	return r.Get(ctx, workspaceID, collectionID, userID, false)
}

func (r *CollectionRepository) RemoveScene(ctx domain.Context, workspaceID, collectionID, userID, sceneID uuid.UUID, expected int64) (domain.Collection, error) {
	db, err := r.ensureMutable(ctx, workspaceID, collectionID, userID, expected)
	if err != nil {
		return domain.Collection{}, err
	}
	result := db.Where("workspace_id=? AND collection_id=? AND scene_id=?", workspaceID, collectionID, sceneID).Delete(&collectionSceneRow{})
	if result.Error != nil {
		return domain.Collection{}, result.Error
	}
	if result.RowsAffected != 1 {
		return domain.Collection{}, domain.ErrInaccessible
	}
	if err = db.Exec(`WITH ranked AS (SELECT scene_id,row_number() OVER (ORDER BY position,scene_id)-1 AS next FROM collection_scenes WHERE workspace_id=? AND collection_id=?) UPDATE collection_scenes cs SET position=ranked.next FROM ranked WHERE cs.workspace_id=? AND cs.collection_id=? AND cs.scene_id=ranked.scene_id`, workspaceID, collectionID, workspaceID, collectionID).Error; err != nil {
		return domain.Collection{}, err
	}
	if err = r.bump(ctx, workspaceID, collectionID, expected); err != nil {
		return domain.Collection{}, err
	}
	return r.Get(ctx, workspaceID, collectionID, userID, false)
}

func (r *CollectionRepository) Reorder(ctx domain.Context, workspaceID, collectionID, userID uuid.UUID, expected int64, sceneIDs []uuid.UUID) (domain.Collection, error) {
	db, err := r.ensureMutable(ctx, workspaceID, collectionID, userID, expected)
	if err != nil {
		return domain.Collection{}, err
	}
	var current []uuid.UUID
	if err = db.Model(&collectionSceneRow{}).Where("workspace_id=? AND collection_id=?", workspaceID, collectionID).Pluck("scene_id", &current).Error; err != nil {
		return domain.Collection{}, err
	}
	if len(current) != len(sceneIDs) {
		return domain.Collection{}, domain.ErrRevisionConflict
	}
	sort.Slice(current, func(i, j int) bool { return current[i].String() < current[j].String() })
	want := append([]uuid.UUID(nil), sceneIDs...)
	sort.Slice(want, func(i, j int) bool { return want[i].String() < want[j].String() })
	for i := range current {
		if current[i] != want[i] {
			return domain.Collection{}, domain.ErrRevisionConflict
		}
	}
	if err = db.Model(&collectionSceneRow{}).Where("workspace_id=? AND collection_id=?", workspaceID, collectionID).Update("position", gorm.Expr("position+1000000")).Error; err != nil {
		return domain.Collection{}, err
	}
	for position, sceneID := range sceneIDs {
		if result := db.Model(&collectionSceneRow{}).Where("workspace_id=? AND collection_id=? AND scene_id=?", workspaceID, collectionID, sceneID).Update("position", position); result.Error != nil || result.RowsAffected != 1 {
			if result.Error != nil {
				return domain.Collection{}, result.Error
			}
			return domain.Collection{}, domain.ErrRevisionConflict
		}
	}
	if err = r.bump(ctx, workspaceID, collectionID, expected); err != nil {
		return domain.Collection{}, err
	}
	return r.Get(ctx, workspaceID, collectionID, userID, false)
}

var _ domain.CollectionRepository = (*CollectionRepository)(nil)
