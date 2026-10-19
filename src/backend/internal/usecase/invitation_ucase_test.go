package usecase

import (
	"context"
	"errors"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

func TestInviteValidationBeforeAuthority(t *testing.T) {
	uc := NewInvitationUsecase(nil, nil, "https://example.test")
	for _, tc := range []struct {
		email string
		role  domain.WorkspaceRole
	}{{"invalid", domain.WorkspaceRoleEditor}, {"valid@example.test", domain.WorkspaceRoleOwner}} {
		if _, err := uc.CreateInvitation(context.Background(), uuid.New(), uuid.New(), tc.email, tc.role); !errors.Is(err, domain.ErrInvitationInput) {
			t.Fatalf("invalid create: %v", err)
		}
	}
	for _, token := range []string{"", "not-a-token"} {
		if _, err := uc.AcceptInvitation(context.Background(), uuid.New(), token); !errors.Is(err, domain.ErrInvitationInvalid) {
			t.Fatalf("invalid token: %v", err)
		}
	}
	if _, err := uc.RegisterWithInvitation(context.Background(), "", domain.RegistrationInput{}); !errors.Is(err, domain.ErrRegistrationRejected) {
		t.Fatalf("invalid register: %v", err)
	}
}
