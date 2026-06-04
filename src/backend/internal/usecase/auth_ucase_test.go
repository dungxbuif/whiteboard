package usecase

import (
	"context"
	"testing"

	"devtool/internal/domain"

	"golang.org/x/oauth2"
)

type mockUserRepo struct {
	domain.UserRepository
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil // Simulate not found
}

func (m *mockUserRepo) Save(ctx context.Context, user *domain.User) error {
	return nil
}

func TestAuthUsecase_GetGoogleLoginURL(t *testing.T) {
	config := &oauth2.Config{
		ClientID: "client-id",
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://accounts.google.com/o/oauth2/auth",
		},
	}
	ucase := NewAuthUsecase(config, &mockUserRepo{}, []byte("secret"))
	
	url := ucase.GetGoogleLoginURL("state")
	if url == "" {
		t.Error("expected non-empty login URL")
	}
}
