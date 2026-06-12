package postgres

import (
	"context"
	"fmt"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderRepository struct {
	db *gorm.DB
}

func NewFolderRepository(db *gorm.DB) *FolderRepository {
	return &FolderRepository{
		db: db,
	}
}

func (r *FolderRepository) FindByID(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID) (*domain.Folder, error) {
	var f domain.Folder
	if err := r.db.WithContext(ctx).First(&f, "workspace_id = ? AND id = ?", workspaceID, id).Error; err != nil {
		return nil, fmt.Errorf("failed to find folder: %w", err)
	}
	return &f, nil
}

func (r *FolderRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Folder, error) {
	var folders []domain.Folder
	if err := r.db.WithContext(ctx).Find(&folders, "workspace_id = ?", workspaceID).Error; err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}
	return folders, nil
}

func (r *FolderRepository) ListByParent(ctx context.Context, workspaceID uuid.UUID, parentID *uuid.UUID) ([]domain.Folder, error) {
	var folders []domain.Folder
	query := r.db.WithContext(ctx).Where("workspace_id = ?", workspaceID)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", parentID)
	}

	if err := query.Find(&folders).Error; err != nil {
		return nil, fmt.Errorf("failed to list folders by parent: %w", err)
	}
	return folders, nil
}

func (r *FolderRepository) Save(ctx context.Context, f *domain.Folder) error {
	if err := r.db.WithContext(ctx).Save(f).Error; err != nil {
		return fmt.Errorf("failed to save folder: %w", err)
	}
	return nil
}

func (r *FolderRepository) MaxDepthByPath(ctx context.Context, workspaceID uuid.UUID, pathPrefix string) (int, error) {
	var maxDepth int
	if err := r.db.WithContext(ctx).Model(&domain.Folder{}).
		Where("workspace_id = ? AND path LIKE ?", workspaceID, pathPrefix+"%").
		Select("COALESCE(MAX(depth), 0)").
		Scan(&maxDepth).Error; err != nil {
		return 0, fmt.Errorf("failed to find maximum folder depth: %w", err)
	}
	return maxDepth, nil
}

func (r *FolderRepository) DeleteRecursive(ctx context.Context, workspaceID uuid.UUID, folderID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var folder domain.Folder
		if err := tx.First(&folder, "workspace_id = ? AND id = ?", workspaceID, folderID).Error; err != nil {
			return err
		}

		// Delete descendants using path prefix
		pathPrefix := folder.Path + "%"
		folderIDs := tx.Unscoped().Model(&domain.Folder{}).
			Select("id").
			Where("workspace_id = ? AND path LIKE ?", workspaceID, pathPrefix)
		if err := tx.Where("workspace_id = ? AND folder_id IN (?)", workspaceID, folderIDs).
			Delete(&domain.Scene{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workspace_id = ? AND path LIKE ?", workspaceID, pathPrefix).Delete(&domain.Folder{}).Error; err != nil {
			return err
		}

		// Delete the folder itself
		if err := tx.Delete(&folder).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *FolderRepository) RestoreRecursive(ctx context.Context, workspaceID uuid.UUID, folderID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var folder domain.Folder
		if err := tx.Unscoped().First(&folder, "workspace_id = ? AND id = ?", workspaceID, folderID).Error; err != nil {
			return fmt.Errorf("failed to find deleted folder: %w", err)
		}

		if err := tx.Unscoped().Model(&domain.Folder{}).
			Where("workspace_id = ? AND path LIKE ?", workspaceID, folder.Path+"%").
			Update("deleted_at", nil).Error; err != nil {
			return fmt.Errorf("failed to restore folder tree: %w", err)
		}
		folderIDs := tx.Unscoped().Model(&domain.Folder{}).
			Select("id").
			Where("workspace_id = ? AND path LIKE ?", workspaceID, folder.Path+"%")
		if err := tx.Unscoped().Model(&domain.Scene{}).
			Where("workspace_id = ? AND folder_id IN (?)", workspaceID, folderIDs).
			Update("deleted_at", nil).Error; err != nil {
			return fmt.Errorf("failed to restore folder scenes: %w", err)
		}
		return nil
	})
}

func (r *FolderRepository) UpdatePathRecursive(ctx context.Context, workspaceID uuid.UUID, oldPathPrefix, newPathPrefix string, depthDelta int) error {
	err := r.db.WithContext(ctx).Model(&domain.Folder{}).
		Where("workspace_id = ? AND path LIKE ?", workspaceID, oldPathPrefix+"%").
		Updates(map[string]interface{}{
			"path":  gorm.Expr("REPLACE(path, ?, ?)", oldPathPrefix, newPathPrefix),
			"depth": gorm.Expr("depth + ?", depthDelta),
		}).
		Error

	if err != nil {
		return fmt.Errorf("failed to update paths recursively: %w", err)
	}
	return nil
}
