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

type activeSessionStoreStub struct {
	session domain.AuthSession
	err     error
}

func (s activeSessionStoreStub) GetSession(context.Context, uuid.UUID) (domain.AuthSession, error) {
	return s.session, s.err
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

func TestAuthMiddlewareBindsLoginSessionClaims(t *testing.T) {
	secret := []byte("login-secret-0123456789abcdef0123456789")
	userID, sid := uuid.New(), uuid.New()
	authTime := time.Now().Add(-time.Minute).Truncate(time.Second)
	token, e := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": userID.String(), "sid": sid.String(), "auth_method": domain.AuthMethodOIDC, "auth_time": authTime.Unix(), "exp": time.Now().Add(time.Hour).Unix()}).SignedString(secret)
	if e != nil {
		t.Fatal(e)
	}
	handler := AuthMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := UserPrincipalFromContext(r.Context())
		if !ok || p.SessionID != sid || p.AuthMethod != domain.AuthMethodOIDC || !p.AuthenticatedAt.Equal(authTime) {
			t.Fatalf("principal=%+v ok=%v", p, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(domain.HeaderAuthorization, domain.BearerPrefix+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAuthMiddlewareRejectsPartialSessionClaims(t *testing.T) {
	secret := []byte("login-secret-0123456789abcdef0123456789")
	userID := uuid.New()
	for _, partial := range []jwt.MapClaims{
		{"auth_method": domain.AuthMethodOIDC},
		{"auth_time": time.Now().Unix()},
		{"sid": uuid.NewString(), "auth_method": domain.AuthMethodOIDC},
	} {
		partial["sub"] = userID.String()
		partial["exp"] = time.Now().Add(time.Hour).Unix()
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, partial).SignedString(secret)
		if err != nil {
			t.Fatal(err)
		}
		handler := AuthMiddleware(secret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("partial session token reached handler")
		}))
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set(domain.HeaderAuthorization, domain.BearerPrefix+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("claims=%v status=%d", partial, response.Code)
		}
	}
}

func TestAuthMiddlewareRejectsRevokedLoginSession(t *testing.T) {
	secret := []byte("login-secret-0123456789abcdef0123456789")
	userID, sid := uuid.New(), uuid.New()
	now := time.Now().UTC()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": userID.String(), "sid": sid.String(), "auth_method": domain.AuthMethodOIDC, "auth_time": now.Unix(), "exp": now.Add(time.Hour).Unix()}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	revoked := now
	handler := AuthMiddlewareWithSessions(secret, nil, activeSessionStoreStub{session: domain.AuthSession{ID: sid, UserID: userID, Method: domain.AuthMethodOIDC, AuthenticatedAt: now, ExpiresAt: now.Add(time.Hour), RevokedAt: &revoked}})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("revoked session reached handler")
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(domain.HeaderAuthorization, domain.BearerPrefix+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
