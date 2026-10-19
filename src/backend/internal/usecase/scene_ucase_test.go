package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type mockSceneRepo struct {
	domain.SceneRepository
	scene *domain.Scene
}

func (m *mockSceneRepo) Create(_ context.Context, scene *domain.Scene) error {
	m.scene = scene
	return nil
}

func (m *mockSceneRepo) FindByID(_ context.Context, workspaceID uuid.UUID, id uuid.UUID) (*domain.Scene, error) {
	if m.scene == nil || m.scene.WorkspaceID != workspaceID || m.scene.ID != id {
		return nil, fmt.Errorf("not found")
	}
	return m.scene, nil
}

func (m *mockSceneRepo) Update(_ context.Context, scene *domain.Scene) error {
	m.scene = scene
	return nil
}

type mockSceneFolderRepo struct {
	domain.FolderRepository
	folders map[uuid.UUID]*domain.Folder
}

func (m *mockSceneFolderRepo) FindByID(_ context.Context, workspaceID uuid.UUID, id uuid.UUID) (*domain.Folder, error) {
	folder, ok := m.folders[id]
	if !ok || folder.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("not found")
	}
	return folder, nil
}

func TestSceneUsecaseCreateAndUpdate(t *testing.T) {
	workspaceID := uuid.New()
	userID := uuid.New()
	folderID := uuid.New()
	repo := &mockSceneRepo{}
	folders := &mockSceneFolderRepo{folders: map[uuid.UUID]*domain.Folder{
		folderID: {ID: folderID, WorkspaceID: workspaceID},
	}}
	usecase := NewSceneUsecase(repo, folders)

	scene, err := usecase.Create(context.Background(), workspaceID, userID, "Board", &folderID, json.RawMessage(`{"elements":[]}`))
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if scene.ContentVersion != 1 {
		t.Fatalf("expected content version 1, got %d", scene.ContentVersion)
	}

	title := "Updated board"
	content := json.RawMessage(`{"elements":[{"id":"1"}]}`)
	updated, err := usecase.Update(context.Background(), workspaceID, userID, scene.ID, &title, nil, &content)
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if updated.Title != title || updated.ContentVersion != 2 {
		t.Fatalf("unexpected updated scene: %#v", updated)
	}
}

func TestSceneUsecaseRejectsCrossWorkspaceFolder(t *testing.T) {
	workspaceID := uuid.New()
	folderID := uuid.New()
	folders := &mockSceneFolderRepo{folders: map[uuid.UUID]*domain.Folder{
		folderID: {ID: folderID, WorkspaceID: uuid.New()},
	}}
	usecase := NewSceneUsecase(&mockSceneRepo{}, folders)

	_, err := usecase.Create(context.Background(), workspaceID, uuid.New(), "Board", &folderID, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected cross-workspace folder error")
	}
}

func TestSceneUsecaseRejectsInvalidContent(t *testing.T) {
	usecase := NewSceneUsecase(&mockSceneRepo{}, &mockSceneFolderRepo{folders: map[uuid.UUID]*domain.Folder{}})
	_, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), "Board", nil, json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestV2SceneRejectsLegacyContentPatch(t *testing.T) {
	workspaceID, userID, id := uuid.New(), uuid.New(), uuid.New()
	repo := &mockSceneRepo{scene: &domain.Scene{ID: id, WorkspaceID: workspaceID, WriterMode: "v2", Title: "Original", Content: json.RawMessage(`{"elements":[]}`)}}
	service := NewSceneUsecase(repo, &mockSceneFolderRepo{})
	content := json.RawMessage(`{"elements":[{"id":"legacy"}]}`)
	_, err := service.Update(context.Background(), workspaceID, userID, id, nil, nil, &content)
	if err == nil || err.Error() != "COLLAB_REQUIRED" {
		t.Fatalf("expected COLLAB_REQUIRED, got %v", err)
	}
	title := "Metadata works"
	result, err := service.Update(context.Background(), workspaceID, userID, id, &title, nil, nil)
	if err != nil || result.Title != title {
		t.Fatalf("metadata update failed: %v", err)
	}
	if string(result.Content) != `{"elements":[]}` {
		t.Fatal("metadata overwrote content")
	}
}
