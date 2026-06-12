package postgres

import (
	"context"
	"fmt"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SceneRepository struct {
	db *gorm.DB
}

func NewSceneRepository(db *gorm.DB) *SceneRepository {
	return &SceneRepository{db: db}
}

func (r *SceneRepository) Create(ctx context.Context, scene *domain.Scene) error {
	if err := r.db.WithContext(ctx).Create(scene).Error; err != nil {
		return fmt.Errorf("failed to create scene: %w", err)
	}
	return nil
}

func (r *SceneRepository) FindByID(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID) (*domain.Scene, error) {
	var scene domain.Scene
	if err := r.db.WithContext(ctx).First(&scene, "workspace_id = ? AND id = ?", workspaceID, id).Error; err != nil {
		return nil, fmt.Errorf("failed to find scene: %w", err)
	}
	return &scene, nil
}

func (r *SceneRepository) List(ctx context.Context, workspaceID uuid.UUID, folderID *uuid.UUID, deletedOnly bool) ([]domain.Scene, error) {
	var scenes []domain.Scene
	query := r.db.WithContext(ctx).Where("workspace_id = ?", workspaceID)
	if deletedOnly {
		query = query.Unscoped().Where("deleted_at IS NOT NULL")
	}
	if folderID == nil && !deletedOnly {
		query = query.Where("folder_id IS NULL")
	} else if folderID != nil {
		query = query.Where("folder_id = ?", *folderID)
	}
	if err := query.Order("updated_at DESC").Find(&scenes).Error; err != nil {
		return nil, fmt.Errorf("failed to list scenes: %w", err)
	}
	return scenes, nil
}

func (r *SceneRepository) Update(ctx context.Context, scene *domain.Scene) error {
	result := r.db.WithContext(ctx).Model(&domain.Scene{}).
		Where("workspace_id = ? AND id = ?", scene.WorkspaceID, scene.ID).
		Updates(map[string]interface{}{
			"title":           scene.Title,
			"folder_id":       scene.FolderID,
			"content":         scene.Content,
			"content_version": gorm.Expr("content_version + 1"),
			"updated_by":      scene.UpdatedBy,
			"updated_at":      gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to update scene: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("failed to update scene: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

func (r *SceneRepository) SoftDelete(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("workspace_id = ? AND id = ?", workspaceID, id).Delete(&domain.Scene{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete scene: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("failed to delete scene: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

func (r *SceneRepository) Restore(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Unscoped().Model(&domain.Scene{}).
		Where("workspace_id = ? AND id = ? AND deleted_at IS NOT NULL", workspaceID, id).
		Update("deleted_at", nil)
	if result.Error != nil {
		return fmt.Errorf("failed to restore scene: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("failed to restore scene: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

func (r *SceneRepository) SetStar(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID, isStarred bool) error {
	result := r.db.WithContext(ctx).Model(&domain.Scene{}).
		Where("workspace_id = ? AND id = ?", workspaceID, id).
		Update("is_starred", isStarred)
	if result.Error != nil {
		return fmt.Errorf("failed to update scene star: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("failed to update scene star: %w", gorm.ErrRecordNotFound)
	}
	return nil
}
