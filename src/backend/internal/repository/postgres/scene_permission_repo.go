package postgres

import (
	"context"
	"fmt"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ScenePermissionRepository struct{ db *gorm.DB }

func NewScenePermissionRepository(db *gorm.DB) *ScenePermissionRepository {
	return &ScenePermissionRepository{db: db}
}

func (r *ScenePermissionRepository) FindBySceneID(ctx context.Context, workspaceID, sceneID uuid.UUID) (*domain.ScenePermission, error) {
	var permission domain.ScenePermission
	if err := r.db.WithContext(ctx).Where("workspace_id = ? AND scene_id = ?", workspaceID, sceneID).First(&permission).Error; err != nil {
		return nil, fmt.Errorf("failed to find scene permission: %w", err)
	}
	return &permission, nil
}

func (r *ScenePermissionRepository) FindActiveByToken(ctx context.Context, token string) (*domain.ScenePermission, error) {
	var permission domain.ScenePermission
	if err := r.db.WithContext(ctx).Where("token = ? AND active = true", token).First(&permission).Error; err != nil {
		return nil, fmt.Errorf("failed to find active scene permission: %w", err)
	}
	return &permission, nil
}

func (r *ScenePermissionRepository) Save(ctx context.Context, permission *domain.ScenePermission) error {
	if err := r.db.WithContext(ctx).Save(permission).Error; err != nil {
		return fmt.Errorf("failed to save scene permission: %w", err)
	}
	return nil
}
