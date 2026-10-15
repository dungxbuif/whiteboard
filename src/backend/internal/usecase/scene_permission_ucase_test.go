package usecase

import (
	"context"
	"fmt"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockScenePermissionRepo struct {
	domain.ScenePermissionRepository
	permission *domain.ScenePermission
}

func (m *mockScenePermissionRepo) FindBySceneID(context.Context, uuid.UUID, uuid.UUID) (*domain.ScenePermission, error) {
	if m.permission == nil {
		return nil, fmt.Errorf("not found: %w", gorm.ErrRecordNotFound)
	}
	return m.permission, nil
}

func (m *mockScenePermissionRepo) FindActiveByToken(context.Context, string) (*domain.ScenePermission, error) {
	if m.permission == nil || !m.permission.Active {
		return nil, fmt.Errorf("not found")
	}
	return m.permission, nil
}

func (m *mockScenePermissionRepo) Save(_ context.Context, permission *domain.ScenePermission) error {
	m.permission = permission
	return nil
}

func TestScenePermissionUsecaseSharesAndProtectsScene(t *testing.T) {
	workspaceID := uuid.New()
	sceneID := uuid.New()
	sceneRepo := &mockSceneRepo{scene: &domain.Scene{ID: sceneID, WorkspaceID: workspaceID}}
	permissionRepo := &mockScenePermissionRepo{}
	usecase := NewScenePermissionUsecase(permissionRepo, sceneRepo)

	permission, err := usecase.Share(context.Background(), workspaceID, sceneID, true, "correct horse")
	if err != nil {
		t.Fatalf("share scene: %v", err)
	}
	if permission.Token == "" || permission.PasswordHash == "" {
		t.Fatalf("expected token and password hash: %#v", permission)
	}
	if _, err := usecase.GetPublicScene(context.Background(), permission.Token, "wrong password"); err == nil {
		t.Fatal("expected invalid password error")
	}
	if _, err := usecase.GetPublicScene(context.Background(), permission.Token, "correct horse"); err != nil {
		t.Fatalf("expected public scene: %v", err)
	}

	if _, err := usecase.Share(context.Background(), workspaceID, sceneID, false, ""); err != nil {
		t.Fatalf("revoke share: %v", err)
	}
	if _, err := usecase.GetPublicScene(context.Background(), permission.Token, ""); err == nil {
		t.Fatal("expected revoked link error")
	}
}
