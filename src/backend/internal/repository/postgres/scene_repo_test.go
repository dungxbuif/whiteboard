package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

func TestSceneRepository(t *testing.T) {
	db := accessTestDB(t, 999)

	ctx := context.Background()
	userID := uuid.New()
	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()
	if err := db.Exec("INSERT INTO users (id, email, name) VALUES (?, ?, ?)", userID, "scene-"+uuid.NewString()+"@example.com", "Scene Tester").Error; err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DELETE FROM users WHERE id = ?", userID)
	for _, id := range []uuid.UUID{workspaceID, otherWorkspaceID} {
		if err := db.Exec("INSERT INTO workspaces (id, name, owner_id) VALUES (?, ?, ?)", id, "Scene Workspace "+id.String(), userID).Error; err != nil {
			t.Fatal(err)
		}
		defer db.Exec("DELETE FROM workspaces WHERE id = ?", id)
	}

	repo := NewSceneRepository(db)
	scene := &domain.Scene{
		ID:             uuid.New(),
		WorkspaceID:    workspaceID,
		Title:          "Board",
		Content:        json.RawMessage(`{"elements":[]}`),
		ContentVersion: 1,
		CreatedBy:      &userID,
		UpdatedBy:      &userID,
	}
	if err := repo.Create(ctx, scene); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	found, err := repo.FindByID(ctx, workspaceID, scene.ID)
	if err != nil || string(found.Content) != `{"elements": []}` && string(found.Content) != `{"elements":[]}` {
		t.Fatalf("unexpected JSONB result: %s, %v", found.Content, err)
	}
	if _, err := repo.FindByID(ctx, otherWorkspaceID, scene.ID); err == nil {
		t.Fatal("expected tenant isolation error")
	}

	found.Title = "Updated"
	found.ContentPatch = true
	found.Content = json.RawMessage(`{"elements":[{"id":"1"}]}`)
	if err := repo.Update(ctx, found); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	found, err = repo.FindByID(ctx, workspaceID, scene.ID)
	if err != nil || found.ContentVersion != 2 {
		t.Fatalf("expected content version 2, got %d, %v", found.ContentVersion, err)
	}

	if err := repo.SetStar(ctx, workspaceID, scene.ID, true); err != nil {
		t.Fatalf("star failed: %v", err)
	}
	if err := repo.SoftDelete(ctx, workspaceID, scene.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	trash, err := repo.List(ctx, workspaceID, nil, true)
	if err != nil || len(trash) != 1 || !trash[0].IsStarred {
		t.Fatalf("unexpected trash result: %#v, %v", trash, err)
	}
	if err := repo.Restore(ctx, workspaceID, scene.ID); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	hidden := &domain.Scene{ID: uuid.New(), WorkspaceID: workspaceID, Title: "Copy", Content: json.RawMessage(`{"elements":[],"appState":{},"files":{}}`), CreatedBy: &userID, UpdatedBy: &userID, PendingDuplicate: true}
	hidden.DeletedAt.Time, hidden.DeletedAt.Valid = time.Now(), true
	if err := repo.Create(ctx, hidden); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindByID(ctx, workspaceID, hidden.ID); err == nil {
		t.Fatal("unfinished duplicate became visible")
	}
	trash, err = repo.List(ctx, workspaceID, nil, true)
	if err != nil || len(trash) != 0 {
		t.Fatalf("pending duplicate leaked to trash: %+v err=%v", trash, err)
	}
	search, err := NewSearchRepository(db).Search(ctx, workspaceID, "Copy", domain.SearchViewTrash, 10)
	if err != nil || len(search) != 0 {
		t.Fatalf("pending duplicate leaked to search: %+v err=%v", search, err)
	}
	if err := repo.Restore(ctx, workspaceID, hidden.ID); err != domain.ErrInaccessible {
		t.Fatalf("pending duplicate restore err=%v", err)
	}
	var stillPending domain.Scene
	if err := db.Unscoped().First(&stillPending, "id=?", hidden.ID).Error; err != nil || !stillPending.PendingDuplicate || !stillPending.DeletedAt.Valid {
		t.Fatalf("pending scene changed: %+v err=%v", stillPending, err)
	}
	hidden.Content = json.RawMessage(`{"elements":[{"id":"copy"}],"appState":{},"files":{}}`)
	if err := repo.PublishDuplicate(ctx, hidden); err != nil {
		t.Fatal(err)
	}
	published, err := repo.FindByID(ctx, workspaceID, hidden.ID)
	if err != nil || published.PendingDuplicate || published.ContentVersion != 2 || !json.Valid(published.Content) {
		t.Fatalf("published=%+v err=%v", published, err)
	}
}
