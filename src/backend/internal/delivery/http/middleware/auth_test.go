package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

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
			name:           "wrong secret",
			authHeader:     "Bearer " + func() string {
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
