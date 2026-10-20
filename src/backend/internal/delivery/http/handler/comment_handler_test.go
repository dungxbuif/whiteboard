package handler

import (
	"context"
	"devtool/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type commentServiceStub struct {
	principal domain.Principal
	called    string
}

func (s *commentServiceStub) remember(name string, principal domain.Principal) {
	s.called, s.principal = name, principal
}
func (s *commentServiceStub) CreateThread(_ context.Context, p domain.Principal, ref domain.SceneRef, in domain.CreateThreadInput) (domain.CommentThread, error) {
	s.remember("create", p)
	return domain.CommentThread{ID: uuid.New(), SceneID: ref.SceneID, Revision: 1, Status: domain.CommentStatusOpen}, nil
}
func (s *commentServiceStub) Reply(_ context.Context, p domain.Principal, _ domain.SceneRef, thread uuid.UUID, in domain.ReplyInput) (domain.Comment, error) {
	s.remember("reply", p)
	return domain.Comment{ID: uuid.New(), ThreadID: thread, Revision: 1}, nil
}
func (s *commentServiceStub) UpdateMessage(_ context.Context, p domain.Principal, _ domain.SceneRef, thread, message uuid.UUID, in domain.CommentEditInput) (domain.Comment, error) {
	s.remember("update-message", p)
	return domain.Comment{ID: message, ThreadID: thread, Revision: 2}, nil
}
func (s *commentServiceStub) DeleteMessage(_ context.Context, p domain.Principal, _ domain.SceneRef, _, _ uuid.UUID, _ domain.RevisionInput) error {
	s.remember("delete-message", p)
	return nil
}
func (s *commentServiceStub) UpdateThread(_ context.Context, p domain.Principal, ref domain.SceneRef, thread uuid.UUID, in domain.ThreadEditInput) (domain.CommentThread, error) {
	s.remember("update-thread", p)
	return domain.CommentThread{ID: thread, SceneID: ref.SceneID, Revision: 2}, nil
}
func (s *commentServiceStub) List(_ context.Context, p domain.Principal, _ domain.SceneRef, _ string, _ int) (domain.CommentPage, error) {
	s.remember("list", p)
	return domain.CommentPage{Items: []domain.CommentThread{}}, nil
}
func (s *commentServiceStub) GetThread(_ context.Context, p domain.Principal, ref domain.SceneRef, thread uuid.UUID) (domain.CommentThread, error) {
	s.remember("get", p)
	return domain.CommentThread{ID: thread, SceneID: ref.SceneID, Revision: 1}, nil
}

type commentAccessStub struct{ workspace, scene uuid.UUID }

func (s commentAccessStub) ResolveWorkspaceID(context.Context, string) (uuid.UUID, error) {
	return s.workspace, nil
}
func (s commentAccessStub) ResolveSceneID(context.Context, uuid.UUID, string) (uuid.UUID, error) {
	return s.scene, nil
}

type commentRoomStub struct{ identity domain.SceneSessionIdentity }

func (s commentRoomStub) ExchangePublicSession(context.Context, string, string, string) (domain.SceneSession, error) {
	return domain.SceneSession{}, domain.ErrAccessDenied
}
func (s commentRoomStub) MintRoomToken(context.Context, domain.Principal, domain.SceneRef) (domain.RoomToken, error) {
	return domain.RoomToken{}, domain.ErrAccessDenied
}
func (s commentRoomStub) ParseSceneSession(string) (domain.SceneSessionIdentity, error) {
	return s.identity, nil
}

func commentAuth(user uuid.UUID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, user)))
		})
	}
}

func TestCommentHandlerAllRoutesAndDualPrincipals(t *testing.T) {
	workspace, scene, user := uuid.New(), uuid.New(), uuid.New()
	guest := domain.Principal{Kind: domain.PrincipalPublic, ID: "guest:test", GrantID: uuid.NewString(), GrantVersion: "1", DisplayName: "Guest", CredentialExpiresAt: time.Now().Add(time.Minute)}
	service := &commentServiceStub{}
	handler := NewCommentHandler(service, commentRoomStub{identity: domain.SceneSessionIdentity{Principal: guest, SceneRef: domain.SceneRef{WorkspaceID: workspace, SceneID: scene}, ExpiresAt: guest.CredentialExpiresAt}}, commentAccessStub{workspace: workspace, scene: scene})
	routes := handler.Routes(commentAuth(user))
	thread, message := uuid.New(), uuid.New()
	tests := []struct {
		name, method, path, body, called string
		public                           bool
		status                           int
	}{
		{"list", http.MethodGet, "/scenes/scene/comments?limit=20", "", "list", false, 200},
		{"create public", http.MethodPost, "/scenes/scene/comments", `{"requestId":"` + uuid.NewString() + `","anchor":{"x":1,"y":2},"body":"hello"}`, "create", true, 201},
		{"get", http.MethodGet, "/scenes/scene/comments/" + thread.String(), "", "get", false, 200},
		{"reply", http.MethodPost, "/scenes/scene/comments/" + thread.String() + "/replies", `{"requestId":"` + uuid.NewString() + `","body":"reply"}`, "reply", false, 201},
		{"edit", http.MethodPatch, "/scenes/scene/comments/" + thread.String() + "/messages/" + message.String(), `{"requestId":"` + uuid.NewString() + `","expectedRevision":"1","body":"edit"}`, "update-message", false, 200},
		{"delete", http.MethodDelete, "/scenes/scene/comments/" + thread.String() + "/messages/" + message.String(), `{"requestId":"` + uuid.NewString() + `","expectedRevision":"1"}`, "delete-message", false, 204},
		{"thread", http.MethodPatch, "/scenes/scene/comments/" + thread.String(), `{"requestId":"` + uuid.NewString() + `","expectedRevision":"1","resolved":true}`, "update-thread", false, 200},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set(domain.HeaderWorkspaceID, workspace.String())
			if test.public {
				request.Header.Set(domain.HeaderAuthorization, domain.BearerPrefix+"scene-session")
			}
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, request)
			if response.Code != test.status || service.called != test.called {
				t.Fatalf("status=%d called=%s body=%s", response.Code, service.called, response.Body.String())
			}
			wantKind := domain.PrincipalUser
			if test.public {
				wantKind = domain.PrincipalPublic
			}
			if service.principal.Kind != wantKind {
				t.Fatalf("principal=%+v", service.principal)
			}
		})
	}
}

func TestCommentHandlerDeleteRequiresStrictBody(t *testing.T) {
	workspace, scene, user := uuid.New(), uuid.New(), uuid.New()
	service := &commentServiceStub{}
	handler := NewCommentHandler(service, commentRoomStub{}, commentAccessStub{workspace: workspace, scene: scene}).Routes(commentAuth(user))
	request := httptest.NewRequest(http.MethodDelete, "/scenes/scene/comments/"+uuid.NewString()+"/messages/"+uuid.NewString(), strings.NewReader(`{"requestId":"`+uuid.NewString()+`","expectedRevision":"1","body":"forbidden"}`))
	request.Header.Set(domain.HeaderWorkspaceID, workspace.String())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || service.called != "" {
		t.Fatalf("status=%d called=%s", response.Code, service.called)
	}
}
