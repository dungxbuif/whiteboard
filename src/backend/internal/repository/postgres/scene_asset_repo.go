package postgres

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"devtool/internal/domain"
)

type sceneAssetRepository struct {
	db *gorm.DB
}

func NewSceneAssetRepository(db *gorm.DB) domain.SceneAssetRepository {
	return &sceneAssetRepository{db: db}
}

func (r *sceneAssetRepository) Create(ctx domain.Context, asset *domain.SceneAsset) error {
	return accessDB(ctx, r.db).Create(asset).Error
}

func (r *sceneAssetRepository) GetByFileIDs(ctx domain.Context, workspaceID, sceneID uuid.UUID, fileIDs []string) ([]domain.SceneAsset, error) {
	var assets []domain.SceneAsset
	err := accessDB(ctx, r.db).
		Where("workspace_id = ? AND scene_id = ? AND file_id IN ? AND deleted_at IS NULL", workspaceID, sceneID, fileIDs).
		Find(&assets).Error
	if err != nil {
		return nil, err
	}
	return assets, nil
}

func (r *sceneAssetRepository) UpdatePresignedURL(ctx domain.Context, id uuid.UUID, url string, expiresAt time.Time) error {
	return accessDB(ctx, r.db).
		Model(&domain.SceneAsset{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"presigned_url":  url,
			"url_expires_at": expiresAt,
		}).Error
}
