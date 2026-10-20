package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"strings"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type SCIMCredentialResolver interface {
	ResolveCredential(context.Context, []byte) (uuid.UUID, error)
}

func SCIMAuth(resolver SCIMCredentialResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get(domain.HeaderAuthorization)
			if !strings.HasPrefix(raw, domain.BearerPrefix) {
				writeSCIMUnauthorized(w)
				return
			}
			token := strings.TrimPrefix(raw, domain.BearerPrefix)
			if !strings.HasPrefix(token, "scim_") || len(token) < 40 || len(token) > 128 || strings.TrimSpace(token) != token {
				writeSCIMUnauthorized(w)
				return
			}
			sum := sha256.Sum256([]byte(token))
			workspace, e := resolver.ResolveCredential(r.Context(), sum[:])
			if e != nil || workspace == uuid.Nil {
				writeSCIMUnauthorized(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeySCIMWorkspace, workspace)))
		})
	}
}
func SCIMWorkspaceFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(domain.ContextKeySCIMWorkspace).(uuid.UUID)
	return id, ok && id != uuid.Nil
}
func writeSCIMUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(domain.SCIMError{Schemas: []string{domain.SCIMErrorSchema}, Status: "401", Detail: "Invalid SCIM credential"})
}
