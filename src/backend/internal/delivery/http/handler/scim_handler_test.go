package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtool/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type scimHandlerService struct {
	filter       string
	start, count int
	err          error
	user         domain.SCIMUser
	creates      int
	patches      int
	deletes      int
}

func (s *scimHandlerService) RotateToken(context.Context, domain.Principal, uuid.UUID) (domain.SCIMTokenResult, error) {
	return domain.SCIMTokenResult{}, s.err
}
func (s *scimHandlerService) TokenStatus(context.Context, domain.Principal, uuid.UUID) (domain.SCIMTokenStatus, error) {
	return domain.SCIMTokenStatus{}, s.err
}
func (s *scimHandlerService) RevokeToken(context.Context, domain.Principal, uuid.UUID) error {
	return s.err
}
func (s *scimHandlerService) CreateUser(context.Context, uuid.UUID, domain.SCIMUser) (domain.SCIMUser, error) {
	s.creates++
	if s.user.ID == "" {
		s.user = domain.SCIMUser{Schemas: []string{domain.SCIMUserSchema}, ID: uuid.NewString(), UserName: "fixture@example.test", Active: true, Role: "viewer", Meta: domain.SCIMMeta{ResourceType: "User", Version: `W/"1"`}}
	}
	return s.user, s.err
}
func (s *scimHandlerService) GetUser(context.Context, uuid.UUID, uuid.UUID) (domain.SCIMUser, error) {
	return s.user, s.err
}
func (s *scimHandlerService) ListUsers(_ context.Context, _ uuid.UUID, f string, start, count int) (domain.SCIMPage[domain.SCIMUser], error) {
	s.filter, s.start, s.count = f, start, count
	return domain.SCIMPage[domain.SCIMUser]{Schemas: []string{domain.SCIMListSchema}, StartIndex: start}, s.err
}
func (s *scimHandlerService) ReplaceUser(context.Context, uuid.UUID, uuid.UUID, string, domain.SCIMUser) (domain.SCIMUser, error) {
	return domain.SCIMUser{}, s.err
}
func (s *scimHandlerService) PatchUser(context.Context, uuid.UUID, uuid.UUID, string, domain.SCIMPatchRequest) (domain.SCIMUser, error) {
	s.patches++
	s.user.Active = false
	s.user.Meta.Version = `W/"2"`
	return s.user, s.err
}
func (s *scimHandlerService) DeleteUser(context.Context, uuid.UUID, uuid.UUID, string) (domain.SCIMUser, error) {
	s.deletes++
	return s.user, s.err
}
func (s *scimHandlerService) AdoptUser(context.Context, domain.Principal, uuid.UUID, uuid.UUID) (domain.SCIMUser, error) {
	return domain.SCIMUser{}, s.err
}
func (s *scimHandlerService) CreateGroup(context.Context, uuid.UUID, domain.SCIMGroup) (domain.SCIMGroup, error) {
	return domain.SCIMGroup{}, s.err
}
func (s *scimHandlerService) GetGroup(context.Context, uuid.UUID, uuid.UUID) (domain.SCIMGroup, error) {
	return domain.SCIMGroup{}, s.err
}
func (s *scimHandlerService) ListGroups(context.Context, uuid.UUID, string, int, int) (domain.SCIMPage[domain.SCIMGroup], error) {
	return domain.SCIMPage[domain.SCIMGroup]{}, s.err
}
func (s *scimHandlerService) ReplaceGroup(context.Context, uuid.UUID, uuid.UUID, string, domain.SCIMGroup) (domain.SCIMGroup, error) {
	return domain.SCIMGroup{}, s.err
}
func (s *scimHandlerService) PatchGroup(context.Context, uuid.UUID, uuid.UUID, string, domain.SCIMPatchRequest) (domain.SCIMGroup, error) {
	return domain.SCIMGroup{}, s.err
}
func (s *scimHandlerService) DeleteGroup(context.Context, uuid.UUID, uuid.UUID, string) error {
	return s.err
}

type scimHandlerAccessStub struct{ workspace uuid.UUID }

func (s scimHandlerAccessStub) ResolveWorkspaceID(context.Context, string) (uuid.UUID, error) {
	return s.workspace, nil
}
func scimTestRouter(service *scimHandlerService, workspace uuid.UUID) *chi.Mux {
	r := chi.NewRouter()
	scoped := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), domain.ContextKeySCIMWorkspace, workspace)))
		})
	}
	NewSCIMHandler(service, scimHandlerAccessStub{workspace}, scoped).Register(r, func(next http.Handler) http.Handler { return next })
	return r
}
func TestSCIMDiscoveryPagingAndBulk(t *testing.T) {
	service := &scimHandlerService{}
	router := scimTestRouter(service, uuid.New())
	for _, path := range []string{"/scim/v2/ServiceProviderConfig", "/scim/v2/ResourceTypes", "/scim/v2/Schemas"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "application/scim+json") {
			t.Fatalf("%s status=%d", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, `/scim/v2/Users?startIndex=1&count=999&filter=userName%20eq%20%22a@example.test%22`, nil))
	if rec.Code != http.StatusOK || service.start != 1 || service.count != 999 {
		t.Fatalf("list status=%d start=%d count=%d", rec.Code, service.start, service.count)
	}
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/scim/v2/Bulk", strings.NewReader(`{}`)))
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("bulk=%d", rec.Code)
	}
}
func TestSCIMErrorsUseProtocolStatus(t *testing.T) {
	service := &scimHandlerService{err: domain.ErrSCIMPrecondition}
	router := scimTestRouter(service, uuid.New())
	id := uuid.NewString()
	req := httptest.NewRequest(http.MethodPut, "/scim/v2/Users/"+id, strings.NewReader(`{"schemas":["`+domain.SCIMUserSchema+`"],"userName":"a@example.test","active":true,"role":"viewer"}`))
	req.Header.Set("If-Match", `W/"1"`)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusPreconditionFailed || !strings.Contains(rec.Body.String(), `"status":"412"`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// TestSCIMFixtureHTTPIdPStyleSequence is an in-process provider fixture. It
// proves the HTTP/media-type/ETag sequence; it is not evidence of a real IdP.
func TestSCIMFixtureHTTPIdPStyleSequence(t *testing.T) {
	service := &scimHandlerService{}
	router := scimTestRouter(service, uuid.New())
	request := func(method, path, body, match string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/scim+json")
		if match != "" {
			req.Header.Set("If-Match", match)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	created := request(http.MethodPost, "/scim/v2/Users", `{"schemas":["`+domain.SCIMUserSchema+`"],"userName":"fixture@example.test","active":true,"role":"viewer"}`, "")
	if created.Code != http.StatusCreated || created.Header().Get("ETag") != `W/"1"` || service.creates != 1 {
		t.Fatalf("create status=%d etag=%s body=%s", created.Code, created.Header().Get("ETag"), created.Body.String())
	}
	id := service.user.ID
	got := request(http.MethodGet, "/scim/v2/Users/"+id, "", "")
	if got.Code != http.StatusOK || !strings.Contains(got.Header().Get("Content-Type"), "application/scim+json") {
		t.Fatalf("get status=%d type=%s", got.Code, got.Header().Get("Content-Type"))
	}
	patched := request(http.MethodPatch, "/scim/v2/Users/"+id, `{"schemas":["`+domain.SCIMPatchSchema+`"],"Operations":[{"op":"replace","path":"active","value":false}]}`, `W/"1"`)
	if patched.Code != http.StatusOK || patched.Header().Get("ETag") != `W/"2"` || service.patches != 1 {
		t.Fatalf("patch status=%d etag=%s", patched.Code, patched.Header().Get("ETag"))
	}
	deleted := request(http.MethodDelete, "/scim/v2/Users/"+id, "", `W/"2"`)
	if deleted.Code != http.StatusNoContent || service.deletes != 1 {
		t.Fatalf("delete status=%d calls=%d", deleted.Code, service.deletes)
	}
}
