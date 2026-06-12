package usecase

import (
	"context"
	"fmt"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockUserRepo struct {
	domain.UserRepository
	user *domain.User
}

func (m *mockUserRepo) FindByEmail(context.Context, string) (*domain.User, error) {
	if m.user == nil {
		return nil, fmt.Errorf("not found: %w", gorm.ErrRecordNotFound)
	}
	return m.user, nil
}

func (m *mockUserRepo) FindByID(context.Context, uuid.UUID) (*domain.User, error) {
	return m.user, nil
}

func (m *mockUserRepo) Save(_ context.Context, user *domain.User) error {
	m.user = user
	return nil
}

func TestAuthUsecaseLogin(t *testing.T) {
	repo := &mockUserRepo{}
	usecase := NewAuthUsecase(repo, []byte("secret"))

	token, err := usecase.Login(context.Background(), domain.TemporaryAdminUsername, domain.TemporaryAdminPassword)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected a token")
	}
	if repo.user == nil || repo.user.Email != domain.TemporaryAdminEmail {
		t.Fatalf("expected temporary admin user, got %#v", repo.user)
	}
}

func TestAuthUsecaseLoginRejectsInvalidCredentials(t *testing.T) {
	usecase := NewAuthUsecase(&mockUserRepo{}, []byte("secret"))
	if _, err := usecase.Login(context.Background(), "wrong", "wrong"); err == nil {
		t.Fatal("expected invalid credentials error")
	}
}
