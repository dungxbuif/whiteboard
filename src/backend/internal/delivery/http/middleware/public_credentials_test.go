package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicLegacyQueryRedaction(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/public/scenes/private-token?password=private-password&other=value", nil)
	RedactPublicCredentials(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "private-") || r.URL.RawQuery != "" {
			t.Fatalf("legacy credentials exposed to request logging: %s %s", r.RequestURI, r.URL.RawQuery)
		}
		if r.URL.Path != "/api/v1/public/scenes/private-token" {
			t.Fatal("route lookup corrupted")
		}
	})).ServeHTTP(httptest.NewRecorder(), req)
}
