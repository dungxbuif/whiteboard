package handler

import (
	"context"
	"devtool/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type versionServiceHandlerStub struct {
	called    string
	principal domain.Principal
}

func (s *versionServiceHandlerStub) Create(_ domain.Context, p domain.Principal, _ domain.SceneRef, _ domain.CreateVersionInput) (domain.SceneVersion, error) {
	s.called, s.principal = "create", p
	return domain.SceneVersion{ID: uuid.New(), DocumentID: uuid.New(), Sequence: "1"}, nil
}
func (s *versionServiceHandlerStub) List(_ domain.Context, p domain.Principal, _ domain.SceneRef, _ string, _ int) (domain.VersionPage, error) {
	s.called, s.principal = "list", p
	return domain.VersionPage{Items: []domain.SceneVersion{}}, nil
}
func (s *versionServiceHandlerStub) Get(_ domain.Context, p domain.Principal, _ domain.SceneRef, id uuid.UUID) (domain.SceneVersion, error) {
	s.called, s.principal = "get", p
	return domain.SceneVersion{ID: id, DocumentID: uuid.New(), Sequence: "1"}, nil
}
func (s *versionServiceHandlerStub) SetMetadata(_ domain.Context, p domain.Principal, _ domain.SceneRef, id uuid.UUID, _ domain.VersionMetadataInput) (domain.SceneVersion, error) {
	s.called, s.principal = "patch", p
	return domain.SceneVersion{ID: id, DocumentID: uuid.New(), Sequence: "1", MetadataRevision: 2}, nil
}

type versionAccessHandlerStub struct{ workspace, scene uuid.UUID }

func (s versionAccessHandlerStub) ResolveWorkspaceID(context.Context, string) (uuid.UUID, error) {
	return s.workspace, nil
}
func (s versionAccessHandlerStub) ResolveSceneID(context.Context, uuid.UUID, string) (uuid.UUID, error) {
	return s.scene, nil
}

func TestVersionHandlerRoutesRequireLoginPrincipal(t *testing.T) {
	workspace, scene, user := uuid.New(), uuid.New(), uuid.New()
	service := &versionServiceHandlerStub{}
	routes := NewVersionHandler(service, versionAccessHandlerStub{workspace, scene}).Routes(commentAuth(user))
	version := uuid.New()
	tests := []struct {
		name, method, path, body, called string
		status                           int
	}{
		{"list", http.MethodGet, "/scenes/scene/versions?limit=20", "", "list", 200},
		{"create", http.MethodPost, "/scenes/scene/versions", `{"requestId":"` + uuid.NewString() + `","expectedSequence":"1","pinned":false}`, "create", 201},
		{"get", http.MethodGet, "/scenes/scene/versions/" + version.String(), "", "get", 200},
		{"patch", http.MethodPatch, "/scenes/scene/versions/" + version.String(), `{"requestId":"` + uuid.NewString() + `","expectedRevision":"1","pinned":true}`, "patch", 200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			request.Header.Set(domain.HeaderWorkspaceID, workspace.String())
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, request)
			if response.Code != tc.status || service.called != tc.called || service.principal.Kind != domain.PrincipalUser {
				t.Fatalf("status=%d called=%s principal=%+v body=%s", response.Code, service.called, service.principal, response.Body.String())
			}
		})
	}
}
