package postgres

import (
	"context"
	"fmt"
	"sort"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SearchRepository struct {
	db *gorm.DB
}

func NewSearchRepository(db *gorm.DB) *SearchRepository {
	return &SearchRepository{db: db}
}

func (r *SearchRepository) Search(ctx context.Context, workspaceID uuid.UUID, query string, view string, limit int) ([]domain.SearchItem, error) {
	items := make([]domain.SearchItem, 0)

	folderQuery := accessDB(ctx, r.db).Model(&domain.Folder{}).Where("workspace_id = ?", workspaceID)
	if view == domain.SearchViewTrash {
		folderQuery = folderQuery.Unscoped().Where("deleted_at IS NOT NULL")
	} else {
		folderQuery = folderQuery.Where("deleted_at IS NULL")
		if view == domain.SearchViewStarred {
			folderQuery = folderQuery.Where("is_starred = ?", true)
		}
	}
	if query != "" {
		folderQuery = folderQuery.Where("name ILIKE ?", "%"+query+"%")
	}
	var folders []domain.Folder
	if err := folderQuery.Order("updated_at DESC").Limit(limit).Find(&folders).Error; err != nil {
		return nil, fmt.Errorf("failed to search folders: %w", err)
	}
	for _, folder := range folders {
		items = append(items, domain.SearchItem{
			ID:          folder.ID,
			WorkspaceID: folder.WorkspaceID,
			Type:        domain.SearchItemTypeFolder,
			Name:        folder.Name,
			IsStarred:   folder.IsStarred,
			DeletedAt:   deletedAt(folder.DeletedAt),
			UpdatedAt:   folder.UpdatedAt,
		})
	}

	sceneQuery := accessDB(ctx, r.db).Model(&domain.Scene{}).Where("workspace_id = ?", workspaceID)
	if view == domain.SearchViewTrash {
		sceneQuery = sceneQuery.Unscoped().Where("deleted_at IS NOT NULL")
	} else {
		sceneQuery = sceneQuery.Where("deleted_at IS NULL")
		if view == domain.SearchViewStarred {
			sceneQuery = sceneQuery.Where("is_starred = ?", true)
		}
	}
	if query != "" {
		sceneQuery = sceneQuery.Where("title ILIKE ?", "%"+query+"%")
	}
	var scenes []domain.Scene
	if err := sceneQuery.Order("updated_at DESC").Limit(limit).Find(&scenes).Error; err != nil {
		return nil, fmt.Errorf("failed to search scenes: %w", err)
	}
	for _, scene := range scenes {
		items = append(items, domain.SearchItem{
			ID:          scene.ID,
			WorkspaceID: scene.WorkspaceID,
			Type:        domain.SearchItemTypeScene,
			Name:        scene.Title,
			FolderID:    scene.FolderID,
			IsStarred:   scene.IsStarred,
			DeletedAt:   deletedAt(scene.DeletedAt),
			UpdatedAt:   scene.UpdatedAt,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func deletedAt(value gorm.DeletedAt) *time.Time {
	if !value.Valid {
		return nil
	}
	deleted := value.Time
	return &deleted
}
