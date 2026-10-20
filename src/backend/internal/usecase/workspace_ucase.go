package usecase

import (
	"context"
	"fmt"
	"strings"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type workspaceUsecase struct {
	repo domain.WorkspaceRepository
}

func (u *workspaceUsecase) UpdateName(ctx context.Context, userID, workspaceID uuid.UUID, name string) (*domain.Workspace, error) {
	name = strings.TrimSpace(name)
	if len([]rune(name)) < 3 || len([]rune(name)) > 100 {
		return nil, domain.ErrInvalidWorkspaceName
	}
	workspace, err := u.repo.FindByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("find workspace: %w", err)
	}
	if workspace.OwnerID != userID {
		return nil, domain.ErrAccessDenied
	}
	if err := u.repo.UpdateName(ctx, workspaceID, name); err != nil {
		return nil, fmt.Errorf("update workspace name: %w", err)
	}
	workspace.Name = name
	return workspace, nil
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
