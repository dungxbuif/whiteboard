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
	if scene.PublicID == nil {
		id, err := domain.NewPublicID()
		if err != nil {
			return err
		}
		scene.PublicID = &id
	}
	if err := accessDB(ctx, r.db).Create(scene).Error; err != nil {
		return fmt.Errorf("failed to create scene: %w", err)
	}
	return nil
}

func (r *SceneRepository) HardDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return accessDB(ctx, r.db).Unscoped().Where("workspace_id=? AND id=?", workspaceID, id).Delete(&domain.Scene{}).Error
}

// PublishDuplicate makes a fully staged duplicate visible in one database write.
func (r *SceneRepository) PublishDuplicate(ctx context.Context, scene *domain.Scene) error {
	result := accessDB(ctx, r.db).Unscoped().Model(&domain.Scene{}).
		Where("workspace_id=? AND id=? AND deleted_at IS NOT NULL AND pending_duplicate = TRUE", scene.WorkspaceID, scene.ID).
		Updates(map[string]any{"content": scene.Content, "content_version": gorm.Expr("content_version + 1"), "deleted_at": nil, "pending_duplicate": false, "updated_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *SceneRepository) FindByID(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID) (*domain.Scene, error) {
	var scene domain.Scene
	if err := accessDB(ctx, r.db).First(&scene, "workspace_id = ? AND id = ?", workspaceID, id).Error; err != nil {
		return nil, fmt.Errorf("failed to find scene: %w", err)
	}
	return &scene, nil
}

func (r *SceneRepository) List(ctx context.Context, workspaceID uuid.UUID, folderID *uuid.UUID, deletedOnly bool) ([]domain.Scene, error) {
	var scenes []domain.Scene
	query := accessDB(ctx, r.db).Where("workspace_id = ?", workspaceID)
	if deletedOnly {
		query = query.Unscoped().Where("deleted_at IS NOT NULL AND pending_duplicate = FALSE")
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
	updates := map[string]interface{}{
		"title": scene.Title, "folder_id": scene.FolderID, "updated_by": scene.UpdatedBy, "updated_at": gorm.Expr("NOW()"),
	}
	query := accessDB(ctx, r.db).Model(&domain.Scene{}).Where("workspace_id = ? AND id = ?", scene.WorkspaceID, scene.ID)
	if scene.ContentPatch {
		query = query.Where("writer_mode = ?", "v1")
		updates["content"] = scene.Content
		updates["content_version"] = gorm.Expr("content_version + 1")
	}
	result := query.Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update scene: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		if scene.ContentPatch {
			return domain.ErrCollabRequired
		}
		return fmt.Errorf("failed to update scene: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

func (r *SceneRepository) SoftDelete(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID) error {
	result := accessDB(ctx, r.db).Where("workspace_id = ? AND id = ?", workspaceID, id).Delete(&domain.Scene{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete scene: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("failed to delete scene: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

func (r *SceneRepository) Restore(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID) error {
	result := accessDB(ctx, r.db).Unscoped().Model(&domain.Scene{}).
		Where("workspace_id = ? AND id = ? AND deleted_at IS NOT NULL AND pending_duplicate = FALSE", workspaceID, id).
		Update("deleted_at", nil)
	if result.Error != nil {
		return fmt.Errorf("failed to restore scene: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrInaccessible
	}
	return nil
}

func (r *SceneRepository) SetStar(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID, isStarred bool) error {
	result := accessDB(ctx, r.db).Model(&domain.Scene{}).
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
