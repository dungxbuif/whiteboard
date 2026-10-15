package middleware

import (
	"net/http"

	"devtool/internal/domain"

	"github.com/casbin/casbin/v3"
)

// AuthorizeMiddleware checks if the authenticated user has permission to perform an action on an object.
func AuthorizeMiddleware(enforcer *casbin.Enforcer, objGetter func(r *http.Request) string, act string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "Unauthorized: User ID missing", http.StatusUnauthorized)
				return
			}

			obj := objGetter(r)
			sub := userID.String()

			ok, err := enforcer.Enforce(sub, obj, act)
			if err != nil {
				http.Error(w, "Internal Server Error during authorization", http.StatusInternalServerError)
				return
			}

			if !ok {
				http.Error(w, "Forbidden: You do not have permission to perform this action", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// WorkspaceIDGetter extracts the workspace ID from the URL or headers to use as the Casbin object.
func WorkspaceIDGetter(r *http.Request) string {
	return r.Header.Get(domain.HeaderWorkspaceID)
}
