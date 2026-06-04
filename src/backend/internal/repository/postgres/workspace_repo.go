package postgres

import (
	"context"
	"fmt"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkspaceRepository struct {
	db *gorm.DB
}

func NewWorkspaceRepository(db *gorm.DB) *WorkspaceRepository {
	return &WorkspaceRepository{
		db: db,
	}
}

func (r *WorkspaceRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Workspace, error) {
	var w domain.Workspace
	if err := r.db.WithContext(ctx).First(&w, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to find workspace: %w", err)
	}
	return &w, nil
}

func (r *WorkspaceRepository) Save(ctx context.Context, w *domain.Workspace) error {
	if err := r.db.WithContext(ctx).Save(w).Error; err != nil {
		return fmt.Errorf("failed to save workspace: %w", err)
	}
	return nil
}

func (r *WorkspaceRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	var workspaces []domain.Workspace
	if err := r.db.WithContext(ctx).Find(&workspaces, "owner_id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}
	return workspaces, nil
}
