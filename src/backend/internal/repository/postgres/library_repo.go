package postgres

import (
	"devtool/internal/domain"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LibraryRepository struct{ db *gorm.DB }

func NewLibraryRepository(db *gorm.DB) *LibraryRepository { return &LibraryRepository{db: db} }

type libraryItemRow struct {
	ID              uuid.UUID
	WorkspaceID     uuid.UUID
	OwnerID         uuid.UUID
	Name            string
	CurrentRevision int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (libraryItemRow) TableName() string { return "library_items" }

type libraryRevisionRow struct {
	LibraryItemID uuid.UUID
	WorkspaceID   uuid.UUID
	Revision      int64
	Content       []byte
	ContentBytes  []byte
	ContentHash   string
	ByteSize      int64
	CreatedAt     time.Time
}

func (libraryRevisionRow) TableName() string { return "library_revisions" }

type libraryAssetRow struct {
	LibraryItemID uuid.UUID
	Revision      int64
	WorkspaceID   uuid.UUID
	SourceSceneID uuid.UUID
	AssetID       uuid.UUID
	FileID        string
}

func (libraryAssetRow) TableName() string { return "library_assets" }

func libraryItem(row libraryItemRow) domain.LibraryItem {
	return domain.LibraryItem{ID: row.ID, WorkspaceID: row.WorkspaceID, OwnerID: row.OwnerID, Name: row.Name, CurrentRevision: row.CurrentRevision, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func libraryRevision(row libraryRevisionRow) domain.LibraryRevision {
	return domain.LibraryRevision{LibraryItemID: row.LibraryItemID, WorkspaceID: row.WorkspaceID, Revision: row.Revision, Content: append([]byte(nil), row.Content...), ContentHash: row.ContentHash, ByteSize: row.ByteSize, CreatedAt: row.CreatedAt}
}

func (r *LibraryRepository) CreateRevision(ctx domain.Context, item domain.LibraryItem, revision domain.LibraryRevision, pins []domain.LibraryAssetPin) (domain.LibraryItem, domain.LibraryRevision, error) {
	db, err := requireCollectionTransaction(ctx, item.WorkspaceID)
	if err != nil {
		return domain.LibraryItem{}, domain.LibraryRevision{}, err
	}
	if revision.Revision == 1 {
		if err = db.Create(&libraryItemRow{ID: item.ID, WorkspaceID: item.WorkspaceID, OwnerID: item.OwnerID, Name: item.Name, CurrentRevision: 1}).Error; err != nil {
			return domain.LibraryItem{}, domain.LibraryRevision{}, err
		}
	} else {
		result := db.Model(&libraryItemRow{}).Where("workspace_id=? AND id=? AND owner_id=? AND current_revision=?", item.WorkspaceID, item.ID, item.OwnerID, revision.Revision-1).Updates(map[string]any{"name": item.Name, "current_revision": revision.Revision, "updated_at": gorm.Expr("clock_timestamp()")})
		if result.Error != nil {
			return domain.LibraryItem{}, domain.LibraryRevision{}, result.Error
		}
		if result.RowsAffected != 1 {
			return domain.LibraryItem{}, domain.LibraryRevision{}, domain.ErrRevisionConflict
		}
	}
	row := libraryRevisionRow{LibraryItemID: item.ID, WorkspaceID: item.WorkspaceID, Revision: revision.Revision, Content: revision.Content, ContentBytes: revision.Content, ContentHash: revision.ContentHash, ByteSize: revision.ByteSize}
	if err = db.Create(&row).Error; err != nil {
		return domain.LibraryItem{}, domain.LibraryRevision{}, err
	}
	for _, pin := range pins {
		var source domain.SceneAsset
		if err = db.Clauses(clause.Locking{Strength: "SHARE"}).Where("workspace_id=? AND scene_id=? AND id=? AND file_id=? AND deleted_at IS NULL", item.WorkspaceID, pin.SourceSceneID, pin.AssetID, pin.FileID).First(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.LibraryItem{}, domain.LibraryRevision{}, domain.ErrInaccessible
			}
			return domain.LibraryItem{}, domain.LibraryRevision{}, err
		}
		if err = db.Create(&libraryAssetRow{LibraryItemID: item.ID, Revision: revision.Revision, WorkspaceID: item.WorkspaceID, SourceSceneID: pin.SourceSceneID, AssetID: pin.AssetID, FileID: pin.FileID}).Error; err != nil {
			return domain.LibraryItem{}, domain.LibraryRevision{}, err
		}
	}
	stored, e := r.Item(ctx, item.WorkspaceID, item.ID)
	if e != nil {
		return domain.LibraryItem{}, domain.LibraryRevision{}, e
	}
	storedRevision, e := r.Revision(ctx, item.WorkspaceID, item.ID, revision.Revision)
	return stored, storedRevision, e
}
func (r *LibraryRepository) Item(ctx domain.Context, workspaceID, itemID uuid.UUID) (domain.LibraryItem, error) {
	var row libraryItemRow
	err := accessDB(ctx, r.db).Where("workspace_id=? AND id=?", workspaceID, itemID).First(&row).Error
	if err != nil {
		return domain.LibraryItem{}, accessError(err)
	}
	return libraryItem(row), nil
}
func (r *LibraryRepository) Revision(ctx domain.Context, workspaceID, itemID uuid.UUID, revision int64) (domain.LibraryRevision, error) {
	var row libraryRevisionRow
	err := accessDB(ctx, r.db).Where("workspace_id=? AND library_item_id=? AND revision=?", workspaceID, itemID, revision).First(&row).Error
	if err != nil {
		return domain.LibraryRevision{}, accessError(err)
	}
	return libraryRevision(row), nil
}
func (r *LibraryRepository) List(ctx domain.Context, workspaceID uuid.UUID, query string, limit int) ([]domain.LibraryItem, error) {
	rows := []libraryItemRow{}
	db := accessDB(ctx, r.db).Where("workspace_id=?", workspaceID)
	if query = strings.TrimSpace(query); query != "" {
		db = db.Where("lower(name) LIKE ?", "%"+strings.ToLower(query)+"%")
	}
	if err := db.Order("updated_at DESC,id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.LibraryItem, len(rows))
	for i, row := range rows {
		items[i] = libraryItem(row)
	}
	return items, nil
}
func (r *LibraryRepository) RevisionAssets(ctx domain.Context, workspaceID, itemID uuid.UUID, revision int64) ([]domain.LibraryAssetPin, error) {
	rows := []libraryAssetRow{}
	if err := accessDB(ctx, r.db).Where("workspace_id=? AND library_item_id=? AND revision=?", workspaceID, itemID, revision).Order("file_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	pins := make([]domain.LibraryAssetPin, len(rows))
	for i, row := range rows {
		pins[i] = domain.LibraryAssetPin{SourceSceneID: row.SourceSceneID, AssetID: row.AssetID, FileID: row.FileID}
	}
	return pins, nil
}
