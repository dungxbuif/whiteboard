package usecase

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
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

func (m *mockUserRepo) FindByGoogleSubject(context.Context, string) (*domain.User, error) {
	if m.user == nil || m.user.GoogleSubject == nil {
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

func TestAuthUsecaseOAuthLoginURL(t *testing.T) {
	usecase := NewAuthUsecase(&mockUserRepo{}, []byte("secret"), OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost/callback",
	})

	loginURL, err := url.Parse(usecase.OAuthLoginURL("state-value"))
	if err != nil {
		t.Fatalf("parse login URL: %v", err)
	}
	if loginURL.Query().Get("state") != "state-value" {
		t.Fatalf("expected state, got %q", loginURL.Query().Get("state"))
	}
	if loginURL.Query().Get("client_id") != "client-id" {
		t.Fatalf("expected client id, got %q", loginURL.Query().Get("client_id"))
	}
}

func TestAuthUsecaseOAuthCallbackCreatesUserAndIssuesJWT(t *testing.T) {
	var receivedAuthorization string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			if r.Form.Get("code") != "authorization-code" {
				t.Errorf("expected authorization code, got %q", r.Form.Get("code"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"access-token","token_type":"Bearer"}`))
		case "/userinfo":
			receivedAuthorization = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sub":"google-subject","email":"user@example.com","name":"Example User","picture":"https://example.com/avatar.png"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()

	repo := &mockUserRepo{}
	usecase := NewAuthUsecase(repo, []byte("secret"), OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost/callback",
		UserInfoURL:  provider.URL + "/userinfo",
		Endpoint: oauth2.Endpoint{
			AuthURL:  provider.URL + "/authorize",
			TokenURL: provider.URL + "/token",
		},
	})

	token, err := usecase.OAuthCallback(context.Background(), "authorization-code")
	if err != nil {
		t.Fatalf("OAuth callback: %v", err)
	}
	if token == "" || !strings.Contains(token, ".") {
		t.Fatalf("expected signed JWT, got %q", token)
	}
	if receivedAuthorization != "Bearer access-token" {
		t.Fatalf("expected bearer token, got %q", receivedAuthorization)
	}
	if repo.user == nil || repo.user.Email != "user@example.com" || repo.user.GoogleSubject == nil || *repo.user.GoogleSubject != "google-subject" {
		t.Fatalf("expected persisted Google user, got %#v", repo.user)
	}
}

func TestAuthUsecaseOAuthCallbackRequiresCode(t *testing.T) {
	usecase := NewAuthUsecase(&mockUserRepo{}, []byte("secret"))
	if _, err := usecase.OAuthCallback(context.Background(), ""); err == nil {
		t.Fatal("expected missing code error")
	}
}
