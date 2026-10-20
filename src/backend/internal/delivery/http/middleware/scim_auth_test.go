package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type scimResolverStub struct {
	workspace uuid.UUID
	hash      []byte
}

func (s *scimResolverStub) ResolveCredential(_ context.Context, hash []byte) (uuid.UUID, error) {
	s.hash = append([]byte(nil), hash...)
	return s.workspace, nil
}
func TestSCIMAuthOnlyAcceptsDedicatedToken(t *testing.T) {
	resolver := &scimResolverStub{workspace: uuid.New()}
	handler := SCIMAuth(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id, ok := SCIMWorkspaceFromContext(r.Context()); !ok || id != resolver.workspace {
			t.Fatal("workspace scope missing")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, raw := range []string{"", "Bearer login.jwt", "Bearer agent_key"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", raw)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Header().Get("Content-Type"), "application/scim+json") {
			t.Fatalf("%q status=%d", raw, rec.Code)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer scim_abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || len(resolver.hash) != 32 {
		t.Fatalf("status=%d hash=%d", rec.Code, len(resolver.hash))
	}
}
