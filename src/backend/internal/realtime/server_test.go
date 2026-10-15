package realtime

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestValidateRoom(t *testing.T) {
	workspaceID, sceneID := uuid.New(), uuid.New()
	room, err := validateRoom(joinPayload{WorkspaceID: workspaceID.String(), SceneID: sceneID.String()})
	if err != nil {
		t.Fatalf("validateRoom() error = %v", err)
	}
	if room != workspaceID.String()+":"+sceneID.String() {
		t.Fatalf("unexpected room: %s", room)
	}
}

func TestValidateRoomRejectsInvalidUUID(t *testing.T) {
	if _, err := validateRoom(joinPayload{WorkspaceID: "bad", SceneID: uuid.NewString()}); err == nil {
		t.Fatal("expected invalid workspace error")
	}
}

func TestUserIDFromRequest(t *testing.T) {
	secret := []byte("test-secret")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": uuid.NewString()})
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/socket.io/?token="+signed, nil)
	userID, err := userIDFromRequest(req, secret)
	if err != nil {
		t.Fatalf("userIDFromRequest() error = %v", err)
	}
	if userID == "" {
		t.Fatal("expected user id")
	}
}

func TestUserIDFromRequestRequiresTokenWhenSecretConfigured(t *testing.T) {
	if _, err := userIDFromRequest(httptest.NewRequest("GET", "/socket.io/", nil), []byte("test-secret")); err == nil {
		t.Fatal("expected missing token to be rejected")
	}
}

func TestTURNRESTCredentialUsesExpiringHMAC(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	username, credential := turnRESTCredential("whiteboard", []byte("shared-secret"), now, time.Hour)

	wantUsername := "1700003600:whiteboard"
	mac := hmac.New(sha1.New, []byte("shared-secret"))
	_, _ = mac.Write([]byte(wantUsername))
	wantCredential := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if username != wantUsername || credential != wantCredential {
		t.Fatalf("turnRESTCredential() = (%q, %q), want (%q, %q)", username, credential, wantUsername, wantCredential)
	}
}

func TestConfigHandlerMintsEphemeralTURNCredential(t *testing.T) {
	server, err := NewServer([]byte("jwt-secret"))
	if err != nil {
		t.Fatal(err)
	}
	server.SetTURNConfig("turn:turn.example.test:5349", "whiteboard", "static-password")
	server.SetTURNAuthSecret("shared-secret", "whiteboard")

	response := httptest.NewRecorder()
	server.ConfigHandler(response, httptest.NewRequest("GET", "/realtime/config", nil))
	var payload struct {
		ICEServers []struct {
			URLs       []string `json:"urls"`
			Username   string   `json:"username"`
			Credential string   `json:"credential"`
		} `json:"iceServers"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.ICEServers) != 1 || len(payload.ICEServers[0].URLs) != 2 || payload.ICEServers[0].URLs[1] != "turn:turn.example.test:5349?transport=tcp" || payload.ICEServers[0].Credential == "static-password" {
		t.Fatalf("expected an ephemeral TURN credential: %#v", payload)
	}
	parts := strings.SplitN(payload.ICEServers[0].Username, ":", 2)
	if len(parts) != 2 || parts[1] != "whiteboard" {
		t.Fatalf("unexpected TURN username: %q", payload.ICEServers[0].Username)
	}
	expiresAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || expiresAt <= time.Now().Unix() {
		t.Fatalf("TURN credential is not expiring in the future: %q", parts[0])
	}
}
