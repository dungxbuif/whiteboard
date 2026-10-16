package postgres

import (
	"context"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

func TestUserRepository(t *testing.T) {
	db := accessTestDB(t, 9)

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("Save and FindByID", func(t *testing.T) {
		email := "test-gorm-" + uuid.New().String() + "@example.com"
		user := &domain.User{
			Email: email,
			Name:  "GORM Test User",
		}

		err := repo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to save user: %v", err)
		}

		found, err := repo.FindByID(ctx, user.ID)
		if err != nil {
			t.Errorf("failed to find user: %v", err)
		}

		if found.ID != user.ID {
			t.Errorf("expected ID %v, got %v", user.ID, found.ID)
		}
	})
}
