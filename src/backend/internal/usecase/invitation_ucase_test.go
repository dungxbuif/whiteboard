package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type invitationHasherSpy struct {
	calls           int
	hash            string
	locked          *bool
	hashedWhileLock bool
}

func (s *invitationHasherSpy) HashPassword(string) (string, error) {
	s.calls++
	if s.locked != nil && *s.locked {
		s.hashedWhileLock = true
	}
	return s.hash, nil
}

type invitationRepositoryStub struct {
	domain.InvitationRepository
	responses []*domain.Invitation
	errors    []error
	calls     int
}

func (s *invitationRepositoryStub) FindByHash(context.Context, string, bool) (*domain.Invitation, error) {
	index := s.calls
	s.calls++
	if index < len(s.errors) && s.errors[index] != nil {
		return nil, s.errors[index]
	}
	if index >= len(s.responses) {
		return nil, domain.ErrInvitationInvalid
	}
	copy := *s.responses[index]
	return &copy, nil
}

type invitationAccessStub struct {
	domain.AccessService
	locked *bool
}

func (s invitationAccessStub) WithWorkspaceLock(ctx context.Context, _ uuid.UUID, _ uuid.UUID, fn func(context.Context) error) error {
	if s.locked != nil {
		*s.locked = true
		defer func() { *s.locked = false }()
	}
	return fn(ctx)
}

func (invitationAccessStub) RequireOwner(context.Context, uuid.UUID, uuid.UUID) error { return nil }

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

func TestInvitationRegistrationValidatesTokenBeforePasswordHash(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	validToken := base64.RawURLEncoding.EncodeToString(make([]byte, invitationTokenBytes))
	validInvite := &domain.Invitation{ID: uuid.New(), WorkspaceID: uuid.New(), InviterID: uuid.New(), Email: "new@example.test", ExpiresAt: now.Add(time.Hour)}
	revokedAt := now.Add(-time.Minute)
	revoked := *validInvite
	revoked.RevokedAt = &revokedAt
	expired := *validInvite
	expired.ExpiresAt = now

	for _, tc := range []struct {
		name  string
		token string
		repo  *invitationRepositoryStub
	}{
		{"malformed", "not-a-token", &invitationRepositoryStub{}},
		{"unknown", validToken, &invitationRepositoryStub{errors: []error{domain.ErrInvitationInvalid}}},
		{"revoked", validToken, &invitationRepositoryStub{responses: []*domain.Invitation{&revoked}}},
		{"expired", validToken, &invitationRepositoryStub{responses: []*domain.Invitation{&expired}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hasher := &invitationHasherSpy{hash: "hash"}
			uc := NewInvitationUsecaseWithPasswordHasher(tc.repo, invitationAccessStub{}, "https://example.test", hasher, func() time.Time { return now })
			_, err := uc.RegisterWithInvitation(context.Background(), tc.token, domain.RegistrationInput{Email: "new@example.test", Name: "New User", Password: "valid-password"})
			if err == nil {
				t.Fatal("invalid invitation registered")
			}
			if hasher.calls != 0 {
				t.Fatalf("password hash calls = %d, want 0", hasher.calls)
			}
		})
	}
}

func TestInvitationRegistrationRevalidatesUnderLockAfterHash(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	token := base64.RawURLEncoding.EncodeToString(make([]byte, invitationTokenBytes))
	valid := &domain.Invitation{ID: uuid.New(), WorkspaceID: uuid.New(), InviterID: uuid.New(), Email: "new@example.test", ExpiresAt: now.Add(time.Hour)}
	revoked := *valid
	revokedAt := now.Add(time.Second)
	revoked.RevokedAt = &revokedAt
	repo := &invitationRepositoryStub{responses: []*domain.Invitation{valid, &revoked}}
	locked := false
	hasher := &invitationHasherSpy{hash: "hash", locked: &locked}
	uc := NewInvitationUsecaseWithPasswordHasher(repo, invitationAccessStub{locked: &locked}, "https://example.test", hasher, func() time.Time { return now })

	if _, err := uc.RegisterWithInvitation(context.Background(), token, domain.RegistrationInput{Email: valid.Email, Name: "New User", Password: "valid-password"}); !errors.Is(err, domain.ErrRegistrationRejected) {
		t.Fatalf("error = %v, want registration rejected", err)
	}
	if hasher.calls != 1 {
		t.Fatalf("password hash calls = %d, want 1", hasher.calls)
	}
	if hasher.hashedWhileLock {
		t.Fatal("password hashing ran while the workspace authority lock was held")
	}
	if repo.calls != 2 {
		t.Fatalf("invitation lookups = %d, want pre-hash and locked revalidation", repo.calls)
	}
}
