package middleware

import (
	"context"
	"encoding/json"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"net/http"
	"strings"
	"time"

	"devtool/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ActiveUserStore interface {
	FindByID(context.Context, uuid.UUID) (*domain.User, error)
}

type ActiveSessionStore interface {
	GetSession(context.Context, uuid.UUID) (domain.AuthSession, error)
}

// AuthMiddleware validates the JWT token in the Authorization header.
func AuthMiddleware(secret []byte, activeUserStores ...ActiveUserStore) func(http.Handler) http.Handler {
	var activeUsers ActiveUserStore
	if len(activeUserStores) > 0 {
		activeUsers = activeUserStores[0]
	}
	return authMiddleware(secret, activeUsers, nil)
}

func AuthMiddlewareWithSessions(secret []byte, activeUsers ActiveUserStore, sessions ActiveSessionStore) func(http.Handler) http.Handler {
	return authMiddleware(secret, activeUsers, sessions)
}

func authMiddleware(secret []byte, activeUsers ActiveUserStore, sessions ActiveSessionStore) func(http.Handler) http.Handler {
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
			if activeUsers != nil {
				user, lookupErr := activeUsers.FindByID(r.Context(), userID)
				if lookupErr != nil || user == nil || user.ID != userID || user.DeletedAt.Valid {
					writeAuthenticationError(w, r)
					return
				}
			}

			expiry, e := claims.GetExpirationTime()
			if e != nil || expiry == nil {
				writeAuthenticationError(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), domain.ContextKeyCredentialExpiry, expiry.Time)
			ctx = context.WithValue(ctx, domain.ContextKeyUserID, userID)
			if rawSID, present := claims["sid"]; present {
				sidValue, sidOK := rawSID.(string)
				sid, sidErr := uuid.Parse(sidValue)
				method, methodOK := claims["auth_method"].(string)
				authNumber, timeOK := claims["auth_time"].(float64)
				if !sidOK || sidErr != nil || sid == uuid.Nil || !methodOK || (method != domain.AuthMethodPassword && method != domain.AuthMethodOIDC && method != domain.AuthMethodBreakGlass) || !timeOK || authNumber <= 0 {
					writeAuthenticationError(w, r)
					return
				}
				ctx = context.WithValue(ctx, domain.ContextKeySessionID, sid)
				ctx = context.WithValue(ctx, domain.ContextKeyAuthMethod, method)
				ctx = context.WithValue(ctx, domain.ContextKeyAuthTime, time.Unix(int64(authNumber), 0))
				var recoveryWorkspace uuid.UUID
				recoveryRaw, recoveryPresent := claims["recovery_workspace_id"]
				if method == domain.AuthMethodBreakGlass {
					recoveryValue, recoveryOK := recoveryRaw.(string)
					recoveryWorkspace, err = uuid.Parse(recoveryValue)
					if !recoveryPresent || !recoveryOK || err != nil || recoveryWorkspace == uuid.Nil {
						writeAuthenticationError(w, r)
						return
					}
					ctx = context.WithValue(ctx, domain.ContextKeyRecoveryWorkspace, recoveryWorkspace)
				} else if recoveryPresent {
					writeAuthenticationError(w, r)
					return
				}
				if sessions != nil {
					session, lookupErr := sessions.GetSession(r.Context(), sid)
					if lookupErr != nil || session.ID != sid || session.UserID != userID || session.Method != method || session.AuthenticatedAt.Unix() != int64(authNumber) || session.RevokedAt != nil || !time.Now().Before(session.ExpiresAt) {
						writeAuthenticationError(w, r)
						return
					}
					if method == domain.AuthMethodBreakGlass && (session.WorkspaceID == nil || *session.WorkspaceID != recoveryWorkspace) {
						writeAuthenticationError(w, r)
						return
					}
				}
			} else {
				_, methodPresent := claims["auth_method"]
				_, timePresent := claims["auth_time"]
				if methodPresent || timePresent {
					writeAuthenticationError(w, r)
					return
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext retrieves the user ID from the context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(domain.ContextKeyUserID).(uuid.UUID)
	return id, ok
}

func UserPrincipalFromContext(ctx context.Context) (domain.Principal, bool) {
	id, ok := UserIDFromContext(ctx)
	if !ok {
		return domain.Principal{}, false
	}
	p := domain.UserPrincipal(id)
	p.CredentialExpiresAt, _ = ctx.Value(domain.ContextKeyCredentialExpiry).(time.Time)
	p.SessionID, _ = ctx.Value(domain.ContextKeySessionID).(uuid.UUID)
	p.AuthMethod, _ = ctx.Value(domain.ContextKeyAuthMethod).(string)
	p.AuthenticatedAt, _ = ctx.Value(domain.ContextKeyAuthTime).(time.Time)
	p.RecoveryWorkspaceID, _ = ctx.Value(domain.ContextKeyRecoveryWorkspace).(uuid.UUID)
	return p, true
}

func writeAuthenticationError(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": "UNAUTHENTICATED", "message": "Authentication required", "requestId": chimiddleware.GetReqID(r.Context())})
}
