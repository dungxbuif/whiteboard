package usecase

import (
	"context"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type mockWorkspaceRepo struct {
	domain.WorkspaceRepository
	savedWorkspace *domain.Workspace
}

func (m *mockWorkspaceRepo) Save(ctx context.Context, w *domain.Workspace) error {
	m.savedWorkspace = w
	return nil
}

func (m *mockWorkspaceRepo) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	return []domain.Workspace{{ID: uuid.New(), Name: "Test Workspace", OwnerID: userID}}, nil
}

func TestWorkspaceUsecase_Create(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	ucase := NewWorkspaceUsecase(repo)
	userID := uuid.New()

	w, err := ucase.Create(context.Background(), userID, "New Workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Name != "New Workspace" {
		t.Errorf("expected name New Workspace, got %s", w.Name)
	}
	if w.OwnerID != userID {
		t.Errorf("expected owner %v, got %v", userID, w.OwnerID)
	}
}

func TestWorkspaceUsecase_ListByUser(t *testing.T) {
	repo := &mockWorkspaceRepo{}
	ucase := NewWorkspaceUsecase(repo)
	userID := uuid.New()

	workspaces, err := ucase.ListByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workspaces) != 1 {
		t.Errorf("expected 1 workspace, got %d", len(workspaces))
	}
}
