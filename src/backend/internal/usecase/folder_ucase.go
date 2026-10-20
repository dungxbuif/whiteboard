package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type folderUsecase struct {
	repo domain.FolderRepository
}

func NewFolderUsecase(repo domain.FolderRepository) domain.FolderUsecase {
	return &folderUsecase{
		repo: repo,
	}
}

func (u *folderUsecase) Create(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID, name string, parentID *uuid.UUID) (*domain.Folder, error) {
	id := uuid.New()
	strippedID := strings.ReplaceAll(id.String(), "-", "")

	depth := 0
	path := domain.FolderPathSeparator + strippedID + domain.FolderPathSeparator

	if parentID != nil {
		parent, err := u.repo.FindByID(ctx, workspaceID, *parentID)
		if err != nil {
			return nil, fmt.Errorf("parent folder not found: %w", err)
		}

		if parent.Depth >= domain.MaxFolderDepth-1 {
			return nil, fmt.Errorf("maximum folder depth reached (%d)", domain.MaxFolderDepth)
		}

		depth = parent.Depth + 1
		path = parent.Path + strippedID + domain.FolderPathSeparator
	}

	folder := &domain.Folder{
		ID:          id,
		WorkspaceID: workspaceID,
		ParentID:    parentID,
		Name:        name,
		Path:        path,
		Depth:       depth,
		CreatedBy:   &userID,
	}

	if err := u.repo.Save(ctx, folder); err != nil {
		return nil, fmt.Errorf("failed to save folder: %w", err)
	}

	return folder, nil
}

func (u *folderUsecase) Move(ctx context.Context, workspaceID uuid.UUID, folderID uuid.UUID, newParentID *uuid.UUID) error {
	folder, err := u.repo.FindByID(ctx, workspaceID, folderID)
	if err != nil {
		return fmt.Errorf("folder not found: %w", err)
	}

	oldPath := folder.Path
	oldDepth := folder.Depth
	strippedID := strings.ReplaceAll(folder.ID.String(), "-", "")

	var newPath string
	var newDepth int

	if newParentID == nil {
		newPath = domain.FolderPathSeparator + strippedID + domain.FolderPathSeparator
		newDepth = 0
	} else {
		if *newParentID == folder.ID {
			return fmt.Errorf("%w: cannot move folder to itself", domain.ErrInvalidFolderUpdate)
		}

		parent, err := u.repo.FindByID(ctx, workspaceID, *newParentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, domain.ErrInaccessible) {
				return domain.ErrInaccessible
			}
			return fmt.Errorf("new parent not found: %w", err)
		}

		if strings.Contains(parent.Path, domain.FolderPathSeparator+strippedID+domain.FolderPathSeparator) {
			return fmt.Errorf("%w: cannot move folder to its own descendant", domain.ErrInvalidFolderUpdate)
		}

		if parent.Depth >= domain.MaxFolderDepth-1 {
			return fmt.Errorf("%w: target parent depth exceeds limit (%d)", domain.ErrInvalidFolderUpdate, domain.MaxFolderDepth)
		}

		newPath = parent.Path + strippedID + domain.FolderPathSeparator
		newDepth = parent.Depth + 1
	}

	maxDepth, err := u.repo.MaxDepthByPath(ctx, workspaceID, oldPath)
	if err != nil {
		return fmt.Errorf("failed to inspect folder subtree: %w", err)
	}
	if newDepth+(maxDepth-oldDepth) >= domain.MaxFolderDepth {
		return fmt.Errorf("%w: moving folder would exceed maximum depth (%d)", domain.ErrInvalidFolderUpdate, domain.MaxFolderDepth)
	}

	folder.ParentID = newParentID
	folder.Path = newPath
	folder.Depth = newDepth

	if err := u.repo.Save(ctx, folder); err != nil {
		return fmt.Errorf("failed to update folder: %w", err)
	}

	return u.repo.UpdatePathRecursive(ctx, workspaceID, oldPath, newPath, newDepth-oldDepth)
}

func (u *folderUsecase) Update(ctx context.Context, workspaceID, folderID uuid.UUID, name *string, newParentID **uuid.UUID) error {
	if name == nil && newParentID == nil {
		return domain.ErrInvalidFolderUpdate
	}
	var trimmed string
	if name != nil {
		trimmed = strings.TrimSpace(*name)
		if len([]rune(trimmed)) < 1 || len([]rune(trimmed)) > 100 {
			return domain.ErrInvalidFolderUpdate
		}
	}
	if newParentID != nil {
		if err := u.Move(ctx, workspaceID, folderID, *newParentID); err != nil {
			return err
		}
	}
	if name != nil {
		folder, err := u.repo.FindByID(ctx, workspaceID, folderID)
		if err != nil {
			return fmt.Errorf("folder not found: %w", err)
		}
		folder.Name = trimmed
		if err := u.repo.Save(ctx, folder); err != nil {
			return fmt.Errorf("failed to update folder name: %w", err)
		}
	}
	return nil
}

func (u *folderUsecase) Delete(ctx context.Context, workspaceID uuid.UUID, folderID uuid.UUID) error {
	return u.repo.DeleteRecursive(ctx, workspaceID, folderID)
}

func (u *folderUsecase) Restore(ctx context.Context, workspaceID uuid.UUID, folderID uuid.UUID) error {
	return u.repo.RestoreRecursive(ctx, workspaceID, folderID)
}

func (u *folderUsecase) ListAll(ctx context.Context, workspaceID uuid.UUID) ([]domain.Folder, error) {
	return u.repo.ListByWorkspace(ctx, workspaceID)
}

func (u *folderUsecase) List(ctx context.Context, workspaceID uuid.UUID, parentID *uuid.UUID) ([]domain.Folder, error) {
	return u.repo.ListByParent(ctx, workspaceID, parentID)
}

func (u *folderUsecase) ToggleStar(ctx context.Context, workspaceID uuid.UUID, folderID uuid.UUID) error {
	folder, err := u.repo.FindByID(ctx, workspaceID, folderID)
	if err != nil {
		return err
	}

	folder.IsStarred = !folder.IsStarred
	return u.repo.Save(ctx, folder)
}
