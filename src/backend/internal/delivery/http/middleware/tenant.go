package middleware

import (
	"context"
	"net/http"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

// TenantMiddleware extracts X-Workspace-ID from header, validates it as UUID, and injects it into context.
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

// WorkspaceIDFromContext retrieves the workspace ID from the context.
func WorkspaceIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(domain.ContextKeyWorkspaceID).(uuid.UUID)
	return id, ok
}
