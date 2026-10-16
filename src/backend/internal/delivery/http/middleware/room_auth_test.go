package middleware

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRoomTokenCannotAuthenticateAsUser(t *testing.T) {
	secret := []byte("test-key-32-bytes-long-for-tests-only")
	for _, purpose := range []string{"room", "scene-session"} {
		raw, e := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": uuid.NewString(), "exp": time.Now().Add(time.Minute).Unix(), "purpose": purpose, "iss": "whiteboard-api", "aud": "whiteboard-collab"}).SignedString(secret)
		if e != nil {
			t.Fatal(e)
		}
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rr := httptest.NewRecorder()
		AuthMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })).ServeHTTP(rr, req)
		if rr.Code != 401 {
			t.Fatalf("%s token authenticated as user: %d", purpose, rr.Code)
		}
	}
}
