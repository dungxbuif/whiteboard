package postgres

import (
	"context"
	"os"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestFolderRepository(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

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

		// Test UpdatePathRecursive
		err := repo.UpdatePathRecursive(ctx, wsID, "/root/", "/newroot/")
		if err != nil {
			t.Errorf("update path failed: %v", err)
		}

		updatedChild, _ := repo.FindByID(ctx, wsID, child.ID)
		if updatedChild.Path != "/newroot/child/" {
			t.Errorf("expected /newroot/child/, got %s", updatedChild.Path)
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
	})
}
