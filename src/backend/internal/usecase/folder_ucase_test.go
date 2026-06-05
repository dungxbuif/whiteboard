package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type mockFolderRepo struct {
	domain.FolderRepository
	folders map[uuid.UUID]*domain.Folder
	saved   *domain.Folder
}

func (m *mockFolderRepo) FindByID(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID) (*domain.Folder, error) {
	if f, ok := m.folders[id]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockFolderRepo) Save(ctx context.Context, f *domain.Folder) error {
	m.saved = f
	m.folders[f.ID] = f
	return nil
}

func (m *mockFolderRepo) UpdatePathRecursive(ctx context.Context, workspaceID uuid.UUID, old, new string) error {
	return nil
}

func TestFolderUsecase_Create(t *testing.T) {
	repo := &mockFolderRepo{folders: make(map[uuid.UUID]*domain.Folder)}
	ucase := NewFolderUsecase(repo)
	wsID := uuid.New()
	uID := uuid.New()

	t.Run("root folder", func(t *testing.T) {
		f, err := ucase.Create(context.Background(), wsID, uID, "Root", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.Depth != 0 {
			t.Errorf("expected depth 0, got %d", f.Depth)
		}
		if len(f.Path) != 34 { // / + 32 chars + /
			t.Errorf("expected path length 34, got %d (%s)", len(f.Path), f.Path)
		}
	})

	t.Run("child folder", func(t *testing.T) {
		parentID := uuid.New()
		repo.folders[parentID] = &domain.Folder{
			ID:    parentID,
			Path:  "/parent/",
			Depth: 0,
		}

		f, err := ucase.Create(context.Background(), wsID, uID, "Child", &parentID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.Depth != 1 {
			t.Errorf("expected depth 1, got %d", f.Depth)
		}
		if !strings.HasPrefix(f.Path, "/parent/") {
			t.Errorf("expected path to start with /parent/, got %s", f.Path)
		}
	})

	t.Run("max depth limit", func(t *testing.T) {
		parentID := uuid.New()
		repo.folders[parentID] = &domain.Folder{
			ID:    parentID,
			Path:  "/some/deep/path/",
			Depth: 29,
		}

		_, err := ucase.Create(context.Background(), wsID, uID, "Too Deep", &parentID)
		if err == nil {
			t.Error("expected error for max depth, got nil")
		}
	})
}
