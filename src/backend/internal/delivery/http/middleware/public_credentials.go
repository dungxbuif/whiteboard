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
		}
		next.ServeHTTP(w, r)
	})
}
