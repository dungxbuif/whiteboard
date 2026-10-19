package usecase

import (
	"context"
	"fmt"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type workspaceUsecase struct {
	repo domain.WorkspaceRepository
}

func NewWorkspaceUsecase(repo domain.WorkspaceRepository) domain.WorkspaceUsecase {
	return &workspaceUsecase{
		repo: repo,
	}
}

func (u *workspaceUsecase) Create(ctx context.Context, userID uuid.UUID, name string) (*domain.Workspace, error) {
	workspace := &domain.Workspace{
		ID:      uuid.New(),
		Name:    name,
		OwnerID: userID,
		Role:    domain.WorkspaceRoleOwner,
	}

	if err := u.repo.Save(ctx, workspace); err != nil {
		return nil, fmt.Errorf("failed to save workspace: %w", err)
	}

	return workspace, nil
}

func (u *workspaceUsecase) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	workspaces, err := u.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	return workspaces, nil
}
