package middleware

import (
	"context"
	"encoding/json"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"net/http"
	"strings"

	"devtool/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthMiddleware validates the JWT token in the Authorization header.
func AuthMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get(domain.HeaderAuthorization)
			if authHeader == "" {
				writeAuthenticationError(w, r)
				return
			}

			if !strings.HasPrefix(authHeader, domain.BearerPrefix) {
				writeAuthenticationError(w, r)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, domain.BearerPrefix)
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return secret, nil
			}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())

			if err != nil || !token.Valid {
				writeAuthenticationError(w, r)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeAuthenticationError(w, r)
				return
			}

			// Login credentials intentionally have no scoped-token purpose or audience.
			for _, key := range []string{"purpose", "iss", "aud", "principalType", "workspaceId", "sceneId"} {
				if _, scoped := claims[key]; scoped {
					writeAuthenticationError(w, r)
					return
				}
			}

			sub, ok := claims["sub"].(string)
			if !ok {
				writeAuthenticationError(w, r)
				return
			}

			userID, err := uuid.Parse(sub)
			if err != nil {
				writeAuthenticationError(w, r)
				return
			}

			expiry, e := claims.GetExpirationTime()
			if e != nil || expiry == nil {
				writeAuthenticationError(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), domain.ContextKeyCredentialExpiry, expiry.Time)
			ctx = context.WithValue(ctx, domain.ContextKeyUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext retrieves the user ID from the context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(domain.ContextKeyUserID).(uuid.UUID)
	return id, ok
}

func writeAuthenticationError(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": "UNAUTHENTICATED", "message": "Authentication required", "requestId": chimiddleware.GetReqID(r.Context())})
}
