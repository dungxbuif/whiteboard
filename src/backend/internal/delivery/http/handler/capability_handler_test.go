package handler

import (
	"bytes"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type authorityUnavailable struct{ domain.CapabilityService }

func (authorityUnavailable) AuthorizeCommand(domain.Context, domain.Principal, domain.SceneMutation) (domain.SignedGrant, error) {
	return domain.SignedGrant{}, errors.New("authority offline")
}

type invalidRoom struct{}

func (invalidRoom) VerifyRoomToken(string) (domain.SceneSessionIdentity, error) {
	return domain.SceneSessionIdentity{}, domain.ErrInvalidSceneSession
}
func (invalidRoom) ParseSceneSession(string) (domain.SceneSessionIdentity, error) {
	return domain.SceneSessionIdentity{}, domain.ErrInvalidSceneSession
}
func TestAuthorityServiceAuthentication(t *testing.T) {
	h, e := NewCapabilityHandler(authorityUnavailable{}, invalidRoom{}, nil, nil, []byte(strings.Repeat("s", 32)))
	if e != nil {
		t.Fatal(e)
	}
	for _, secret := range []string{"", "wrong"} {
		r := httptest.NewRequest("POST", "/internal/authority/decisions", strings.NewReader(`{}`))
		r.Header.Set("Authorization", "Bearer "+secret)
		w := httptest.NewRecorder()
		h.AuthorityHandler().ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("unauthenticated authority returned %d", w.Code)
		}
	}
}

type validRoom struct{ identity domain.SceneSessionIdentity }

func (v validRoom) VerifyRoomToken(string) (domain.SceneSessionIdentity, error) {
	return v.identity, nil
}
func (v validRoom) ParseSceneSession(string) (domain.SceneSessionIdentity, error) {
	return v.identity, nil
}
func TestAuthorityUnavailableHTTP(t *testing.T) {
	p := domain.UserPrincipal(uuid.New())
	p.CredentialExpiresAt = time.Now().Add(time.Minute)
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	h, _ := NewCapabilityHandler(authorityUnavailable{}, validRoom{domain.SceneSessionIdentity{Principal: p, SceneRef: ref}}, nil, nil, []byte(strings.Repeat("s", 32)))
	server := httptest.NewServer(h.AuthorityHandler())
	defer server.Close()
	body, _ := json.Marshal(map[string]any{"roomToken": "room", "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "action": "scene:update", "requestId": "request", "payloadHash": strings.Repeat("a", 64)})
	r, _ := http.NewRequest("POST", server.URL+"/internal/authority/decisions", bytes.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+strings.Repeat("s", 32))
	response, e := http.DefaultClient.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != 503 || bytes.Contains(raw, []byte(`"token"`)) {
		t.Fatalf("authority failure returned %d %s", response.StatusCode, raw)
	}
}
