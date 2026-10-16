package middleware

import (
	"context"
	"net/http"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

// TenantMiddleware injects the canonical workspace UUID. Production first
// resolves aliases and authorizes membership in CollaborationAccessMiddleware.
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspaceIDStr := r.Header.Get(domain.HeaderWorkspaceID)
		if workspaceIDStr == "" {
			http.Error(w, domain.HeaderWorkspaceID+" header is required", http.StatusBadRequest)
			return
		}

		workspaceID, err := uuid.Parse(workspaceIDStr)
		if err != nil {
			http.Error(w, "invalid "+domain.HeaderWorkspaceID+" format", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), domain.ContextKeyWorkspaceID, workspaceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WorkspaceAccessMiddleware is the legacy owner-only guard retained for v1
// callers. Production uses CollaborationAccessMiddleware for current roles.
func WorkspaceAccessMiddleware(workspaces domain.WorkspaceRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			workspaceID, ok := WorkspaceIDFromContext(r.Context())
			if !ok {
				http.Error(w, "Workspace context is required", http.StatusBadRequest)
				return
			}
			workspace, err := workspaces.FindByID(r.Context(), workspaceID)
			if err != nil || workspace.OwnerID != userID {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WorkspaceIDFromContext retrieves the workspace ID from the context.
func WorkspaceIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(domain.ContextKeyWorkspaceID).(uuid.UUID)
	return id, ok
}
