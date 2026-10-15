package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type mockAuthUsecase struct {
	domain.AuthUsecase
}

func (mockAuthUsecase) OAuthLoginURL(state string) string {
	return "https://accounts.google.test/authorize?state=" + state
}

func (mockAuthUsecase) OAuthCallback(context.Context, string) (string, error) {
	return "test-token", nil
}

func (mockAuthUsecase) GetUserByID(context.Context, uuid.UUID) (*domain.User, error) {
	return &domain.User{}, nil
}

func TestAuthHandlerGoogleLoginSetsStateCookieAndRedirects(t *testing.T) {
	handler := NewAuthHandler(mockAuthUsecase{}, "http://localhost:5173")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
	response := httptest.NewRecorder()

	handler.GoogleLogin(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, response.Code)
	}
	if response.Header().Get("Location") == "" {
		t.Fatal("expected OAuth provider redirect")
	}
	stateCookie := response.Result().Cookies()
	if len(stateCookie) != 1 || stateCookie[0].Name != domain.OAuthStateCookie || stateCookie[0].Value == "" {
		t.Fatalf("expected OAuth state cookie, got %#v", stateCookie)
	}
}

func TestAuthHandlerGoogleCallbackRejectsInvalidState(t *testing.T) {
	handler := NewAuthHandler(mockAuthUsecase{}, "http://localhost:5173")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=wrong&code=code", nil)
	request.AddCookie(&http.Cookie{Name: domain.OAuthStateCookie, Value: "expected"})
	response := httptest.NewRecorder()

	handler.GoogleCallback(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestAuthHandlerGoogleCallbackRedirectsWithJWT(t *testing.T) {
	handler := NewAuthHandler(mockAuthUsecase{}, "http://localhost:5173")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=expected&code=code", nil)
	request.AddCookie(&http.Cookie{Name: domain.OAuthStateCookie, Value: "expected"})
	response := httptest.NewRecorder()

	handler.GoogleCallback(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, response.Code)
	}
	if response.Header().Get("Location") != "http://localhost:5173#token=test-token" {
		t.Fatalf("unexpected redirect %q", response.Header().Get("Location"))
	}
}
