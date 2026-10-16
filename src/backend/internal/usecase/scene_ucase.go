package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type sceneUsecase struct {
	repo       domain.SceneRepository
	folderRepo domain.FolderRepository
}

func NewSceneUsecase(repo domain.SceneRepository, folderRepo domain.FolderRepository) domain.SceneUsecase {
	return &sceneUsecase{repo: repo, folderRepo: folderRepo}
}

func (u *sceneUsecase) Create(ctx context.Context, workspaceID, userID uuid.UUID, title string, folderID *uuid.UUID, content json.RawMessage) (*domain.Scene, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("scene title is required")
	}
	if len(content) == 0 {
		content = json.RawMessage(`{}`)
	}
	if !json.Valid(content) {
		return nil, fmt.Errorf("scene content must be valid JSON")
	}
	if err := u.validateFolder(ctx, workspaceID, folderID); err != nil {
		return nil, err
	}

	scene := &domain.Scene{
		ID:             uuid.New(),
		WorkspaceID:    workspaceID,
		FolderID:       folderID,
		Title:          title,
		Content:        content,
		ContentVersion: 1,
		CreatedBy:      &userID,
		UpdatedBy:      &userID,
	}
	if err := u.repo.Create(ctx, scene); err != nil {
		return nil, err
	}
	return scene, nil
}

func (u *sceneUsecase) Get(ctx context.Context, workspaceID, id uuid.UUID) (*domain.Scene, error) {
	return u.repo.FindByID(ctx, workspaceID, id)
}

func (u *sceneUsecase) List(ctx context.Context, workspaceID uuid.UUID, folderID *uuid.UUID, deletedOnly bool) ([]domain.Scene, error) {
	return u.repo.List(ctx, workspaceID, folderID, deletedOnly)
}

func (u *sceneUsecase) Update(ctx context.Context, workspaceID, userID, id uuid.UUID, title *string, folderID **uuid.UUID, content *json.RawMessage) (*domain.Scene, error) {
	scene, err := u.repo.FindByID(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if content != nil && scene.WriterMode == "v2" {
		return nil, domain.ErrCollabRequired
	}
	scene.ContentPatch = content != nil
	if title != nil {
		trimmed := strings.TrimSpace(*title)
		if trimmed == "" {
			return nil, fmt.Errorf("scene title is required")
		}
		scene.Title = trimmed
	}
	if folderID != nil {
		if err := u.validateFolder(ctx, workspaceID, *folderID); err != nil {
			return nil, err
		}
		scene.FolderID = *folderID
	}
	if content != nil {
		if !json.Valid(*content) {
			return nil, fmt.Errorf("scene content must be valid JSON")
		}
		scene.Content = *content
	}
	scene.UpdatedBy = &userID
	if err := u.repo.Update(ctx, scene); err != nil {
		return nil, err
	}
	if content != nil {
		scene.ContentVersion++
	}
	return scene, nil
}

func (u *sceneUsecase) SetStar(ctx context.Context, workspaceID, id uuid.UUID, isStarred bool) error {
	return u.repo.SetStar(ctx, workspaceID, id, isStarred)
}

func (u *sceneUsecase) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return u.repo.SoftDelete(ctx, workspaceID, id)
}

func (u *sceneUsecase) Restore(ctx context.Context, workspaceID, id uuid.UUID) error {
	return u.repo.Restore(ctx, workspaceID, id)
}

func (u *sceneUsecase) validateFolder(ctx context.Context, workspaceID uuid.UUID, folderID *uuid.UUID) error {
	if folderID == nil {
		return nil
	}
	if _, err := u.folderRepo.FindByID(ctx, workspaceID, *folderID); err != nil {
		return fmt.Errorf("folder does not belong to workspace: %w", err)
	}
	return nil
}
