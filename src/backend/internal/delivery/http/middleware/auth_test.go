package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"devtool/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type activeUserStoreStub struct {
	user  *domain.User
	err   error
	calls int
}

func (s *activeUserStoreStub) FindByID(context.Context, uuid.UUID) (*domain.User, error) {
	s.calls++
	return s.user, s.err
}

func TestAuthMiddleware(t *testing.T) {
	secret := []byte("test-secret")
	userID := uuid.New()

	// Generate a valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	validToken, _ := token.SignedString(secret)

	// Generate an expired token
	expiredTokenClaim := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	expiredToken, _ := expiredTokenClaim.SignedString(secret)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectUserID   bool
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectUserID:   true,
		},
		{
			name:           "missing header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name:           "invalid format",
			authHeader:     "InvalidToken",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name:           "expired token",
			authHeader:     "Bearer " + expiredToken,
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name: "wrong secret",
			authHeader: "Bearer " + func() string {
				t, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": userID.String()}).SignedString([]byte("wrong"))
				return t
			}(),
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, ok := UserIDFromContext(r.Context())
				if tt.expectUserID {
					if !ok {
						t.Errorf("expected user ID in context, but not found")
					}
					if id != userID {
						t.Errorf("expected user ID %v, got %v", userID, id)
					}
				} else if ok {
					t.Errorf("did not expect user ID in context, but found %v", id)
				}
				w.WriteHeader(http.StatusOK)
			})

			handler := AuthMiddleware(secret)(nextHandler)

			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestAuthMiddlewareFailsClosedUnlessJWTSubjectIsActive(t *testing.T) {
	secret := []byte("login-secret-0123456789abcdef0123456789")
	userID := uuid.New()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}

	deletedAt := time.Now()
	for _, tc := range []struct {
		name   string
		store  *activeUserStoreStub
		status int
	}{
		{"active", &activeUserStoreStub{user: &domain.User{ID: userID}}, http.StatusNoContent},
		{"missing", &activeUserStoreStub{err: gorm.ErrRecordNotFound}, http.StatusUnauthorized},
		{"nil result", &activeUserStoreStub{}, http.StatusUnauthorized},
		{"soft deleted", &activeUserStoreStub{user: &domain.User{ID: userID, DeletedAt: gorm.DeletedAt{Time: deletedAt, Valid: true}}}, http.StatusUnauthorized},
		{"store unavailable", &activeUserStoreStub{err: errors.New("database unavailable")}, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := AuthMiddleware(secret, tc.store)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/", nil)
			req.Header.Set(domain.HeaderAuthorization, domain.BearerPrefix+token)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tc.status {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, tc.status, rr.Body.String())
			}
			if called != (tc.status == http.StatusNoContent) {
				t.Fatalf("downstream called = %v", called)
			}
			if tc.store.calls != 1 {
				t.Fatalf("active-user lookups = %d, want 1", tc.store.calls)
			}
			if tc.status == http.StatusUnauthorized && !strings.Contains(rr.Body.String(), `"code":"UNAUTHENTICATED"`) {
				t.Fatalf("unexpected auth error: %s", rr.Body.String())
			}
		})
	}
}
