package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockFolderRepo struct {
	domain.FolderRepository
	folders          map[uuid.UUID]*domain.Folder
	saved            *domain.Folder
	depthDelta       int
	maxDepth         int
	restoredFolderID uuid.UUID
}

func (m *mockFolderRepo) FindByID(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID) (*domain.Folder, error) {
	if f, ok := m.folders[id]; ok {
		return f, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockFolderRepo) Save(ctx context.Context, f *domain.Folder) error {
	m.saved = f
	m.folders[f.ID] = f
	return nil
}

func (m *mockFolderRepo) MaxDepthByPath(context.Context, uuid.UUID, string) (int, error) {
	return m.maxDepth, nil
}

func (m *mockFolderRepo) RestoreRecursive(_ context.Context, _ uuid.UUID, folderID uuid.UUID) error {
	m.restoredFolderID = folderID
	return nil
}

func (m *mockFolderRepo) UpdatePathRecursive(ctx context.Context, workspaceID uuid.UUID, old, new string, depthDelta int) error {
	m.depthDelta = depthDelta
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

func TestFolderUsecase_MoveUpdatesDescendantDepth(t *testing.T) {
	workspaceID := uuid.New()
	folderID := uuid.New()
	parentID := uuid.New()
	repo := &mockFolderRepo{folders: map[uuid.UUID]*domain.Folder{
		folderID: {ID: folderID, WorkspaceID: workspaceID, Path: "/folder/", Depth: 0},
		parentID: {ID: parentID, WorkspaceID: workspaceID, Path: "/parent/", Depth: 2},
	}, maxDepth: 0}

	err := NewFolderUsecase(repo).Move(context.Background(), workspaceID, folderID, &parentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.depthDelta != 3 {
		t.Fatalf("expected descendant depth delta 3, got %d", repo.depthDelta)
	}
}

func TestFolderUsecase_MoveRejectsSubtreePastDepthLimit(t *testing.T) {
	workspaceID := uuid.New()
	folderID := uuid.New()
	parentID := uuid.New()
	repo := &mockFolderRepo{folders: map[uuid.UUID]*domain.Folder{
		folderID: {ID: folderID, WorkspaceID: workspaceID, Path: "/folder/", Depth: 5},
		parentID: {ID: parentID, WorkspaceID: workspaceID, Path: "/parent/", Depth: 20},
	}, maxDepth: 14}

	err := NewFolderUsecase(repo).Move(context.Background(), workspaceID, folderID, &parentID)
	if err == nil || !strings.Contains(err.Error(), "maximum depth") {
		t.Fatalf("expected maximum depth error, got %v", err)
	}
}

func TestFolderUsecase_Restore(t *testing.T) {
	workspaceID := uuid.New()
	folderID := uuid.New()
	repo := &mockFolderRepo{folders: make(map[uuid.UUID]*domain.Folder)}

	if err := NewFolderUsecase(repo).Restore(context.Background(), workspaceID, folderID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.restoredFolderID != folderID {
		t.Fatalf("expected restored folder %s, got %s", folderID, repo.restoredFolderID)
	}
}

func TestFolderUsecase_UpdateNameAndParent(t *testing.T) {
	workspaceID, folderID, parentID := uuid.New(), uuid.New(), uuid.New()
	repo := &mockFolderRepo{folders: map[uuid.UUID]*domain.Folder{
		folderID: {ID: folderID, WorkspaceID: workspaceID, Name: "Old", Path: "/folder/"},
		parentID: {ID: parentID, WorkspaceID: workspaceID, Path: "/parent/"},
	}}
	name := "  New  "
	parent := &parentID
	if err := NewFolderUsecase(repo).Update(context.Background(), workspaceID, folderID, &name, &parent); err != nil {
		t.Fatal(err)
	}
	if repo.saved.Name != "New" || repo.saved.ParentID == nil || *repo.saved.ParentID != parentID {
		t.Fatalf("folder=%+v", repo.saved)
	}
	self := &folderID
	if err := NewFolderUsecase(repo).Update(context.Background(), workspaceID, folderID, nil, &self); err == nil {
		t.Fatal("self move succeeded")
	}
	if repo.saved.Name != "New" {
		t.Fatal("invalid move changed folder name")
	}
	childID := uuid.New()
	repo.folders[childID] = &domain.Folder{ID: childID, WorkspaceID: workspaceID, Path: repo.saved.Path + "child/"}
	child := &childID
	if err := NewFolderUsecase(repo).Update(context.Background(), workspaceID, folderID, nil, &child); err == nil {
		t.Fatal("descendant move succeeded")
	}
}

func TestFolderUsecase_UpdateMissingParent(t *testing.T) {
	workspaceID, folderID, parentID := uuid.New(), uuid.New(), uuid.New()
	repo := &mockFolderRepo{folders: map[uuid.UUID]*domain.Folder{
		folderID: {ID: folderID, WorkspaceID: workspaceID, Path: "/folder/"},
	}}
	parent := &parentID
	err := NewFolderUsecase(repo).Update(context.Background(), workspaceID, folderID, nil, &parent)
	if !errors.Is(err, domain.ErrInaccessible) {
		t.Fatalf("want inaccessible, got %v", err)
	}
	if repo.saved != nil {
		t.Fatal("missing parent changed folder")
	}
}
