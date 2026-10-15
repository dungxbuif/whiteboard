package realtime

import (
	"net/http/httptest"
	"testing"

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
