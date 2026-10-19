package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type mockAuthUsecase struct {
	domain.AuthUsecase
}

func TestPasswordLoginRejectsUnboundedOrAmbiguousBodiesBeforeUsecase(t *testing.T) {
	uc := &rateLimitedAuthUsecase{}
	h := NewAuthHandler(uc)
	for _, tc := range []struct {
		name   string
		body   string
		status int
	}{
		{"oversized", `{"email":"` + strings.Repeat("a", loginRequestMaxBytes) + `","password":"password"}`, http.StatusRequestEntityTooLarge},
		{"unknown field", `{"email":"user@example.test","password":"password","extra":true}`, http.StatusBadRequest},
		{"trailing JSON", `{"email":"user@example.test","password":"password"}{}`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.PasswordLogin(rr, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tc.body)))
			if rr.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, tc.status, rr.Body.String())
			}
		})
	}
	if uc.calls != 0 {
		t.Fatalf("invalid bodies reached login usecase %d times", uc.calls)
	}
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
