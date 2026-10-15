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
	return r.db.WithContext(ctx).Create(asset).Error
}

func (r *sceneAssetRepository) GetByFileIDs(ctx domain.Context, workspaceID uuid.UUID, fileIDs []string) ([]domain.SceneAsset, error) {
	var assets []domain.SceneAsset
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND file_id IN ? AND deleted_at IS NULL", workspaceID, fileIDs).
		Find(&assets).Error
	if err != nil {
		return nil, err
	}
	return assets, nil
}

func (r *sceneAssetRepository) UpdatePresignedURL(ctx domain.Context, id uuid.UUID, url string, expiresAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.SceneAsset{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"presigned_url":  url,
			"url_expires_at": expiresAt,
		}).Error
}
