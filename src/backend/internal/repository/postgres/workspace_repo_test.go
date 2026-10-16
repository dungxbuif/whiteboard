package postgres

import (
	"context"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

func TestWorkspaceRepository(t *testing.T) {
	db := accessTestDB(t, 9)

	repo := NewWorkspaceRepository(db)
	ctx := context.Background()

	// Need a user for owner_id
	owner := &domain.User{
		ID:    uuid.New(),
		Email: "workspace-owner-" + uuid.New().String() + "@example.com",
		Name:  "Workspace Owner",
	}
	if err := db.Create(owner).Error; err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}

	t.Run("Save and FindByID", func(t *testing.T) {
		workspace := &domain.Workspace{
			Name:    "GORM Workspace",
			OwnerID: owner.ID,
		}

		err := repo.Save(ctx, workspace)
		if err != nil {
			t.Errorf("failed to save workspace: %v", err)
		}

		found, err := repo.FindByID(ctx, workspace.ID)
		if err != nil {
			t.Errorf("failed to find workspace: %v", err)
		}

		if found.ID != workspace.ID {
			t.Errorf("expected ID %v, got %v", workspace.ID, found.ID)
		}
	})
}
