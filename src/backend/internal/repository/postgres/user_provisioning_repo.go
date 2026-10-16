package postgres

import (
	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserProvisioningRepository struct{ db *gorm.DB }

func NewUserProvisioningRepository(db *gorm.DB) *UserProvisioningRepository {
	return &UserProvisioningRepository{db: db}
}
func (r *UserProvisioningRepository) FindDefaultWorkspace(ctx domain.Context, user uuid.UUID) (*domain.Workspace, error) {
	var w domain.Workspace
	err := accessDB(ctx, r.db).Where("owner_id=?", user).First(&w).Error
	return &w, err
}
func (r *UserProvisioningRepository) CreateDefaultWorkspace(ctx domain.Context, w *domain.Workspace) error {
	return accessDB(ctx, r.db).Create(w).Error
}
