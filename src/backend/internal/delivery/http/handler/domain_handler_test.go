package handler

import (
	"context"
	"devtool/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type domainUsecaseStub struct {
	calls      int
	requestIDs []uuid.UUID
}

func (s *domainUsecaseStub) Begin(_ domain.Context, _ domain.Principal, workspaceID uuid.UUID, value string, requestID uuid.UUID) (domain.DomainChallenge, error) {
	s.calls++
	s.requestIDs = append(s.requestIDs, requestID)
	return domain.DomainChallenge{WorkspaceDomain: domain.WorkspaceDomain{ID: uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"), WorkspaceID: workspaceID, Domain: value, Status: domain.DomainPending}, DNSName: "_whiteboard-verification." + value, TXTValue: "whiteboard-verification=stable"}, nil
}
func (s *domainUsecaseStub) Verify(domain.Context, domain.Principal, uuid.UUID, uuid.UUID, uuid.UUID) (domain.WorkspaceDomain, error) {
	s.calls++
	return domain.WorkspaceDomain{}, nil
}
func (s *domainUsecaseStub) Capture(domain.Context, domain.Principal, uuid.UUID, uuid.UUID, uuid.UUID) (domain.MembershipResult, error) {
	s.calls++
	return domain.MembershipResult{}, nil
}
func (s *domainUsecaseStub) Revoke(domain.Context, domain.Principal, uuid.UUID, uuid.UUID, uuid.UUID) (domain.WorkspaceDomain, error) {
	s.calls++
	return domain.WorkspaceDomain{}, nil
}
func (s *domainUsecaseStub) List(domain.Context, domain.Principal, uuid.UUID) ([]domain.WorkspaceDomain, error) {
	s.calls++
	return nil, nil
}
func (s *domainUsecaseStub) Discover(domain.Context, string) (domain.DomainDiscovery, error) {
	s.calls++
	return domain.DomainDiscovery{}, nil
}

type domainHandlerAccessStub struct{ workspace uuid.UUID }

func (s domainHandlerAccessStub) ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error) {
	return s.workspace, nil
}

func domainTestRouter(uc *domainUsecaseStub, workspace, user uuid.UUID) *chi.Mux {
	router := chi.NewRouter()
	NewDomainHandler(uc, domainHandlerAccessStub{workspace: workspace}).Register(router, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, user)))
		})
	})
	return router
}

func TestDomainHandlerRejectsUnknownAndTrailingBodies(t *testing.T) {
	workspace, user, id := uuid.New(), uuid.New(), uuid.New()
	tests := []struct{ name, path, body string }{
		{name: "begin unknown", path: "/workspaces/w/domains", body: `{"domain":"example.com","requestId":"` + uuid.NewString() + `","admin":true}`},
		{name: "begin trailing", path: "/workspaces/w/domains", body: `{"domain":"example.com","requestId":"` + uuid.NewString() + `"}{}`},
		{name: "verify unknown", path: "/workspaces/w/domains/" + id.String() + "/verify", body: `{"requestId":"` + uuid.NewString() + `","extra":1}`},
		{name: "capture trailing", path: "/workspaces/w/domains/" + id.String() + "/capture", body: `{"requestId":"` + uuid.NewString() + `"} null`},
		{name: "revoke unknown", path: "/workspaces/w/domains/" + id.String() + "/revoke", body: `{"requestId":"` + uuid.NewString() + `","extra":true}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &domainUsecaseStub{}
			request := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			response := httptest.NewRecorder()
			domainTestRouter(uc, workspace, user).ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || uc.calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, uc.calls, response.Body.String())
			}
		})
	}
}

func TestDomainHandlerRejectsOversizedBody(t *testing.T) {
	uc := &domainUsecaseStub{}
	request := httptest.NewRequest(http.MethodPost, "/workspaces/w/domains", strings.NewReader(`{"domain":"`+strings.Repeat("a", int(domainRequestMaxBytes))+`","requestId":"`+uuid.NewString()+`"}`))
	response := httptest.NewRecorder()
	domainTestRouter(uc, uuid.New(), uuid.New()).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || uc.calls != 0 {
		t.Fatalf("status=%d calls=%d", response.Code, uc.calls)
	}
}

func TestDomainHandlerForwardsSameRequestIDOnLostResponseRetry(t *testing.T) {
	workspace, user, requestID := uuid.New(), uuid.New(), uuid.New()
	uc := &domainUsecaseStub{}
	router := domainTestRouter(uc, workspace, user)
	body := `{"domain":"example.com","requestId":"` + requestID.String() + `"}`
	responses := make([]string, 0, 2)
	for range 2 {
		request := httptest.NewRequest(http.MethodPost, "/workspaces/w/domains", strings.NewReader(body))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
		responses = append(responses, response.Body.String())
	}
	if uc.calls != 2 || len(uc.requestIDs) != 2 || uc.requestIDs[0] != requestID || uc.requestIDs[1] != requestID {
		t.Fatalf("calls=%d requestIDs=%v", uc.calls, uc.requestIDs)
	}
	if responses[0] != responses[1] || !strings.Contains(responses[0], "whiteboard-verification=stable") {
		t.Fatalf("responses=%q", responses)
	}
}
