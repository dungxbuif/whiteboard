package middleware

import (
	"net/http"
	"strings"
)

// Route lookup retains the path, but request logging receives no legacy query
// or opaque token in RequestURI. Install this before any request logger.
func RedactPublicCredentials(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const legacyPrefix = "/api/v1/public/scenes/"
		if strings.HasPrefix(r.URL.Path, legacyPrefix) {
			r.URL.RawQuery = ""
			r.URL.ForceQuery = false
			r.RequestURI = legacyPrefix + "[REDACTED]"
		} else if strings.HasPrefix(r.URL.Path, "/api/v1/presentations/") && strings.HasSuffix(r.URL.Path, "/join") {
			r.URL.RawQuery = ""
			r.URL.ForceQuery = false
			r.RequestURI = "/api/v1/presentations/[REDACTED]/join"
		}
		next.ServeHTTP(w, r)
	})
}
