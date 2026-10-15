package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type scenePermissionUsecase struct {
	permissions domain.ScenePermissionRepository
	scenes      domain.SceneRepository
}

func NewScenePermissionUsecase(permissions domain.ScenePermissionRepository, scenes domain.SceneRepository) domain.ScenePermissionUsecase {
	return &scenePermissionUsecase{permissions: permissions, scenes: scenes}
}

func (u *scenePermissionUsecase) Share(ctx context.Context, workspaceID, sceneID uuid.UUID, active bool, password string) (*domain.ScenePermission, error) {
	if _, err := u.scenes.FindByID(ctx, workspaceID, sceneID); err != nil {
		return nil, err
	}
	permission, err := u.permissions.FindBySceneID(ctx, workspaceID, sceneID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		permission = &domain.ScenePermission{ID: uuid.New(), WorkspaceID: workspaceID, SceneID: sceneID, Token: uuid.NewString()}
	} else if err != nil {
		return nil, err
	}
	permission.Active = active
	password = strings.TrimSpace(password)
	if password == "" {
		permission.PasswordHash = ""
	} else if len(password) < 6 {
		return nil, fmt.Errorf("share password must be at least 6 characters")
	} else {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, fmt.Errorf("failed to hash share password: %w", hashErr)
		}
		permission.PasswordHash = string(hash)
	}
	if err := u.permissions.Save(ctx, permission); err != nil {
		return nil, err
	}
	return permission, nil
}

func (u *scenePermissionUsecase) GetPublicScene(ctx context.Context, token, password string) (*domain.Scene, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("share token is required")
	}
	permission, err := u.permissions.FindActiveByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if permission.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(permission.PasswordHash), []byte(password)); err != nil {
			return nil, fmt.Errorf("share password is invalid: %w", err)
		}
	}
	return u.scenes.FindByID(ctx, permission.WorkspaceID, permission.SceneID)
}
