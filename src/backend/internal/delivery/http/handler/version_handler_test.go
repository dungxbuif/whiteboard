package handler

import (
	"context"
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type versionServiceHandlerStub struct {
	called    string
	principal domain.Principal
	err       error
}

func (s *versionServiceHandlerStub) Duplicate(_ domain.Context, p domain.Principal, _ domain.SceneRef, title string, folderID *uuid.UUID) (*domain.Scene, error) {
	s.called, s.principal = "duplicate", p
	return &domain.Scene{ID: uuid.New(), Title: title, FolderID: folderID}, s.err
}

func TestVersionHandlerDuplicateUsesAuthenticatedRoute(t *testing.T) {
	workspace, scene, user := uuid.New(), uuid.New(), uuid.New()
	service := &versionServiceHandlerStub{}
	routes := NewVersionHandler(service, versionAccessHandlerStub{workspace, scene}).Routes(commentAuth(user))
	request := httptest.NewRequest(http.MethodPost, "/scenes/scene/duplicate", strings.NewReader(`{"title":"Copy","folderId":null}`))
	request.Header.Set(domain.HeaderWorkspaceID, workspace.String())
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || service.called != "duplicate" || !strings.Contains(response.Body.String(), `"title":"Copy"`) {
		t.Fatalf("status=%d called=%s body=%s", response.Code, service.called, response.Body.String())
	}
}

func TestVersionHandlerPropagatesLoginCredentialExpiry(t *testing.T) {
	workspace, scene, user := uuid.New(), uuid.New(), uuid.New()
	expires := time.Now().Add(15 * time.Minute).Truncate(time.Second)
	secret := []byte(strings.Repeat("v", 32))
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: user.String(), ExpiresAt: jwt.NewNumericDate(expires)}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	service := &versionServiceHandlerStub{}
	routes := NewVersionHandler(service, versionAccessHandlerStub{workspace, scene}).Routes(middleware.AuthMiddleware(secret))
	request := httptest.NewRequest(http.MethodPost, "/scenes/scene/versions", strings.NewReader(`{"requestId":"`+uuid.NewString()+`","expectedSequence":"1","pinned":false}`))
	request.Header.Set(domain.HeaderAuthorization, domain.BearerPrefix+token)
	request.Header.Set(domain.HeaderWorkspaceID, workspace.String())
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !service.principal.CredentialExpiresAt.Equal(expires) || service.principal.GrantID != "membership:"+workspace.String()+":"+user.String() {
		t.Fatalf("status=%d expiry=%v want=%v body=%s", response.Code, service.principal.CredentialExpiresAt, expires, response.Body.String())
	}
}

func (s *versionServiceHandlerStub) Create(_ domain.Context, p domain.Principal, _ domain.SceneRef, _ domain.CreateVersionInput) (domain.SceneVersion, error) {
	s.called, s.principal = "create", p
	if s.err != nil {
		return domain.SceneVersion{}, s.err
	}
	return domain.SceneVersion{ID: uuid.New(), DocumentID: uuid.New(), Sequence: "1"}, nil
}

func (s *versionServiceHandlerStub) CurrentProjection(_ domain.Context, p domain.Principal, _ domain.SceneRef) (json.RawMessage, error) {
	s.called, s.principal = "projection", p
	return json.RawMessage(`{"content":{"elements":[{"id":"recent"}],"appState":{},"files":{}},"sequence":"2"}`), s.err
}

func TestVersionHandlerServesCurrentDurableProjection(t *testing.T) {
	workspace, scene, user := uuid.New(), uuid.New(), uuid.New()
	service := &versionServiceHandlerStub{}
	routes := NewVersionHandler(service, versionAccessHandlerStub{workspace, scene}).Routes(commentAuth(user))
	request := httptest.NewRequest(http.MethodGet, "/scenes/scene/projection", nil)
	request.Header.Set(domain.HeaderWorkspaceID, workspace.String())
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" || service.called != "projection" || !strings.Contains(response.Body.String(), `"id":"recent"`) {
		t.Fatalf("status=%d called=%s body=%s", response.Code, service.called, response.Body.String())
	}
}

func TestVersionProjectionNoStoreOnAuthenticationAndServiceErrors(t *testing.T) {
	workspace, scene := uuid.New(), uuid.New()
	secret := []byte(strings.Repeat("v", 32))
	service := &versionServiceHandlerStub{}
	routes := NewVersionHandler(service, versionAccessHandlerStub{workspace, scene}).Routes(middleware.AuthMiddleware(secret))
	request := httptest.NewRequest(http.MethodGet, "/scenes/scene/projection", nil)
	request.Header.Set(domain.HeaderWorkspaceID, workspace.String())
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unauthenticated status=%d cache=%q", response.Code, response.Header().Get("Cache-Control"))
	}

	service.err = &domain.AgentCollabError{Status: http.StatusServiceUnavailable, Code: "UNAVAILABLE"}
	routes = NewVersionHandler(service, versionAccessHandlerStub{workspace, scene}).Routes(commentAuth(uuid.New()))
	response = httptest.NewRecorder()
	routes.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("service status=%d cache=%q", response.Code, response.Header().Get("Cache-Control"))
	}
}

func TestVersionHandlerReturnsUnavailableWhenWriterBridgeIsMissing(t *testing.T) {
	workspace, scene, user := uuid.New(), uuid.New(), uuid.New()
	service := &versionServiceHandlerStub{err: &domain.AgentCollabError{Status: 503, Code: "UNAVAILABLE"}}
	routes := NewVersionHandler(service, versionAccessHandlerStub{workspace, scene}).Routes(commentAuth(user))
	request := httptest.NewRequest(http.MethodPost, "/scenes/scene/versions", strings.NewReader(`{"requestId":"`+uuid.NewString()+`","expectedSequence":"1","pinned":false}`))
	request.Header.Set(domain.HeaderWorkspaceID, workspace.String())
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"code":"UNAVAILABLE"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
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
func (s *versionServiceHandlerStub) Diff(_ domain.Context, p domain.Principal, _ domain.SceneRef, _ uuid.UUID, _ string) (domain.SceneDiff, error) {
	s.called, s.principal = "diff", p
	return domain.SceneDiff{AddedIDs: []string{"new"}, RemovedIDs: []string{}, ChangedIDs: []string{}, ChangedAppStateFields: []string{}}, s.err
}
func (s *versionServiceHandlerStub) Restore(_ domain.Context, p domain.Principal, _ domain.SceneRef, input domain.RestoreVersionInput) (domain.RestoreVersionResult, error) {
	s.called, s.principal = "restore", p
	return domain.RestoreVersionResult{RequestID: input.RequestID, Sequence: "2", StateVector: "AA==", PreRestoreVersionID: uuid.New()}, s.err
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
		{"diff", http.MethodGet, "/scenes/scene/versions/" + version.String() + "/diff?againstSequence=1", "", "diff", 200},
		{"restore", http.MethodPost, "/scenes/scene/versions/" + version.String() + "/restore", `{"requestId":"` + uuid.NewString() + `","expectedSequence":"1"}`, "restore", 200},
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
