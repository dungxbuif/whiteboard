package handler

import (
	"devtool/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type presentationHandlerStub struct{}

func (presentationHandlerStub) Create(domain.Context, domain.Principal, domain.SceneRef, uuid.UUID) (domain.SessionGrant, error) {
	return domain.SessionGrant{}, nil
}
func (presentationHandlerStub) CreateLink(domain.Context, domain.Principal, domain.SceneRef, uuid.UUID, uuid.UUID) (domain.OneTimeShownLink, error) {
	return domain.OneTimeShownLink{}, nil
}
func (presentationHandlerStub) Join(_ domain.Context, _ string, name string) (domain.PendingPresentationGrant, error) {
	return domain.PendingPresentationGrant{SessionID: uuid.New(), DisplayName: name, Token: "pending"}, nil
}
func (presentationHandlerStub) Admit(domain.Context, domain.Principal, domain.SceneRef, uuid.UUID, string, int64) (domain.SessionGrant, error) {
	return domain.SessionGrant{}, nil
}
func (presentationHandlerStub) ResolveAudience(domain.Context, domain.Principal, domain.SceneRef, uuid.UUID) error {
	return nil
}
func (presentationHandlerStub) RevokeLink(domain.Context, domain.Principal, domain.SceneRef, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}

type presentationAccessStub struct{}

func (presentationAccessStub) ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (presentationAccessStub) ResolveSceneID(domain.Context, uuid.UUID, string) (uuid.UUID, error) {
	return uuid.New(), nil
}

func TestPresentationJoinRateLimitAndNoStore(t *testing.T) {
	h := NewPresentationHandler(presentationHandlerStub{}, presentationAccessStub{})
	router := h.Routes(func(next http.Handler) http.Handler { return next })
	for index := 0; index < 61; index++ {
		request := httptest.NewRequest(http.MethodPost, "/presentations/invite/join", strings.NewReader(`{"displayName":"Guest"}`))
		request.RemoteAddr = "203.0.113.10:1234"
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		want := http.StatusOK
		if index == 60 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("attempt=%d status=%d body=%s", index+1, response.Code, response.Body.String())
		}
		if index == 0 && response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("pending credential response was cacheable")
		}
	}
}
