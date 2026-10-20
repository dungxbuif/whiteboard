package postgres

import (
	"context"
	"testing"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

func TestFolderRepository(t *testing.T) {
	db := accessTestDB(t, 999)

	repo := NewFolderRepository(db)
	ctx := context.Background()

	// Setup: Create a real workspace and user for constraints
	uID := uuid.New()
	db.Exec("INSERT INTO users (id, email, name) VALUES (?, ?, ?)", uID, "folder-test@example.com", "Tester")
	defer db.Exec("DELETE FROM users WHERE id = ?", uID)

	wsID := uuid.New()
	db.Exec("INSERT INTO workspaces (id, name, owner_id) VALUES (?, ?, ?)", wsID, "Test Workspace", uID)
	defer db.Exec("DELETE FROM workspaces WHERE id = ?", wsID)

	t.Run("Save and FindByID", func(t *testing.T) {
		f := &domain.Folder{
			ID:          uuid.New(),
			WorkspaceID: wsID,
			Name:        "Repo Folder",
			Path:        "/some/path/",
		}

		err := repo.Save(ctx, f)
		if err != nil {
			t.Errorf("failed to save: %v", err)
		}

		found, err := repo.FindByID(ctx, wsID, f.ID)
		if err != nil {
			t.Errorf("failed to find: %v", err)
		}
		if found.Name != "Repo Folder" {
			t.Errorf("expected name Repo Folder, got %s", found.Name)
		}
	})

	t.Run("Recursive operations", func(t *testing.T) {
		root := &domain.Folder{ID: uuid.New(), WorkspaceID: wsID, Name: "Root", Path: "/root/"}
		child := &domain.Folder{ID: uuid.New(), WorkspaceID: wsID, Name: "Child", Path: "/root/child/", ParentID: &root.ID}

		repo.Save(ctx, root)
		repo.Save(ctx, child)
		scene := &domain.Scene{
			ID: uuid.New(), WorkspaceID: wsID, FolderID: &child.ID, Title: "Child Scene",
			Content: []byte(`{}`), ContentVersion: 1,
		}
		if err := db.Create(scene).Error; err != nil {
			t.Fatalf("failed to create child scene: %v", err)
		}
		pending := &domain.Scene{ID: uuid.New(), WorkspaceID: wsID, FolderID: &child.ID, Title: "Pending copy", Content: []byte(`{}`), PendingDuplicate: true}
		pending.DeletedAt.Time, pending.DeletedAt.Valid = time.Now(), true
		if err := db.Create(pending).Error; err != nil {
			t.Fatalf("failed to create pending copy: %v", err)
		}

		// Test UpdatePathRecursive
		err := repo.UpdatePathRecursive(ctx, wsID, "/root/", "/newroot/", 2)
		if err != nil {
			t.Errorf("update path failed: %v", err)
		}

		updatedChild, _ := repo.FindByID(ctx, wsID, child.ID)
		if updatedChild.Path != "/newroot/child/" {
			t.Errorf("expected /newroot/child/, got %s", updatedChild.Path)
		}
		if updatedChild.Depth != child.Depth+2 {
			t.Errorf("expected depth %d, got %d", child.Depth+2, updatedChild.Depth)
		}

		// Test DeleteRecursive
		err = repo.DeleteRecursive(ctx, wsID, root.ID)
		if err != nil {
			t.Errorf("delete recursive failed: %v", err)
		}

		_, err = repo.FindByID(ctx, wsID, child.ID)
		if err == nil {
			t.Error("expected child to be deleted")
		}
		var deletedScene domain.Scene
		if err := db.Unscoped().First(&deletedScene, "id = ? AND deleted_at IS NOT NULL", scene.ID).Error; err != nil {
			t.Errorf("expected child scene to be deleted: %v", err)
		}

		if err := repo.RestoreRecursive(ctx, wsID, root.ID); err != nil {
			t.Errorf("restore recursive failed: %v", err)
		}
		if _, err := repo.FindByID(ctx, wsID, child.ID); err != nil {
			t.Errorf("expected child to be restored: %v", err)
		}
		if err := db.First(&domain.Scene{}, "id = ?", scene.ID).Error; err != nil {
			t.Errorf("expected child scene to be restored: %v", err)
		}
		var hidden domain.Scene
		if err := db.Unscoped().First(&hidden, "id = ?", pending.ID).Error; err != nil || !hidden.PendingDuplicate || !hidden.DeletedAt.Valid {
			t.Errorf("folder restore exposed pending copy: %+v err=%v", hidden, err)
		}
	})
}
