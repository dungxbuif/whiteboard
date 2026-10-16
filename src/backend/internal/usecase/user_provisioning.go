package usecase

import (
	"errors"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultPersonalWorkspaceName = "My Workspace"

// EnsureDefaultWorkspace uses the caller's transaction-scoped repository.
func EnsureDefaultWorkspace(ctx domain.Context, tx domain.UserProvisioningRepository, userID uuid.UUID) error {
	_, err := tx.FindDefaultWorkspace(ctx, userID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	alias, err := domain.NewPublicID()
	if err != nil {
		return err
	}
	return tx.CreateDefaultWorkspace(ctx, &domain.Workspace{ID: uuid.New(), PublicID: &alias, Name: defaultPersonalWorkspaceName, OwnerID: userID})
}
