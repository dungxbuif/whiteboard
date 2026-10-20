package middleware

import (
	"context"
	"devtool/internal/domain"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type bodyPolicy struct {
	domain.AccessService
	workspace uuid.UUID
}

type featurePolicyAccess struct {
	bodyPolicy
	owner bool
}

func (p featurePolicyAccess) ResolveWorkspace(context.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error) {
	return domain.AccessDecision{Access: domain.AccessView, Owner: p.owner, ACLVersion: 1}, nil
}

func (p featurePolicyAccess) RequireOwner(context.Context, uuid.UUID, uuid.UUID) error {
	if !p.owner {
		return domain.ErrAccessDenied
	}
	return nil
}

func (p bodyPolicy) ResolveWorkspaceID(context.Context, string) (uuid.UUID, error) {
	return p.workspace, nil
}
func (p bodyPolicy) ResolveWorkspace(context.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error) {
	return domain.AccessDecision{Access: domain.AccessEdit, Owner: true, ACLVersion: 1}, nil
}
func (p bodyPolicy) WithWorkspaceLock(ctx context.Context, _ uuid.UUID, _ uuid.UUID, fn func(context.Context) error) error {
	return fn(ctx)
}

type memoryDeadlineResponse struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
}

func (r *memoryDeadlineResponse) SetReadDeadline(deadline time.Time) error {
	r.deadlines = append(r.deadlines, deadline)
	return nil
}

func TestAccessBodyLimitsPreserveDocumentSize(t *testing.T) {
	user, workspace := uuid.New(), uuid.New()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			t.Error(err)
		}
		w.Header().Set("Consumed-Bytes", strconv.FormatInt(n, 10))
		w.WriteHeader(http.StatusNoContent)
	})
	gateway := CollaborationAccessMiddleware(bodyPolicy{workspace: workspace})(next)
	for _, tc := range []struct {
		name, path string
		size       int
		chunked    bool
		want       int
	}{
		{"25 MiB document and envelope", "/api/v1/scenes/", 25*1024*1024 + 128, false, 204},
		{"oversized scene envelope", "/api/v1/scenes/", 26*1024*1024 + 1, false, 413},
		{"oversized chunked envelope", "/api/v1/scenes/", 26*1024*1024 + 1, true, 413},
		{"oversized asset metadata", "/api/v1/assets/presign-view", 1024*1024 + 1, true, 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(strings.Repeat("x", tc.size)))
			if tc.chunked {
				request.ContentLength = -1
			}
			request = request.WithContext(context.WithValue(request.Context(), domain.ContextKeyUserID, user))
			response := &memoryDeadlineResponse{ResponseRecorder: httptest.NewRecorder()}
			gateway.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("status %d want %d: %s", response.Code, tc.want, response.Body.String())
			}
			if tc.want == 204 {
				if response.Header().Get("Consumed-Bytes") != strconv.Itoa(tc.size) {
					t.Fatal("request body truncated")
				}
				if len(response.deadlines) != 2 || response.deadlines[0].IsZero() || !response.deadlines[1].IsZero() {
					t.Fatal("successful body read must set and clear its deadline")
				}
			} else if response.Header().Get("Consumed-Bytes") != "" {
				t.Fatal("oversized request reached mutation handler")
			}
		})
	}
}
func TestAccessBodyRequiresDeadlineSupport(t *testing.T) {
	user := uuid.New()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/scenes/", strings.NewReader("{}"))
	request = request.WithContext(context.WithValue(request.Context(), domain.ContextKeyUserID, user))
	response := httptest.NewRecorder()
	gateway := CollaborationAccessMiddleware(bodyPolicy{workspace: uuid.New()})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("unsupported body transport reached authority") }))
	gateway.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unsupported transport returned %d", response.Code)
	}
}

func TestFeaturePolicyGETAllowsMemberAndPATCHRequiresOwner(t *testing.T) {
	user, workspace := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name   string
		method string
		owner  bool
		want   int
	}{
		{name: "member reads", method: http.MethodGet, owner: false, want: http.StatusNoContent},
		{name: "member cannot update", method: http.MethodPatch, owner: false, want: http.StatusForbidden},
		{name: "owner updates", method: http.MethodPatch, owner: true, want: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				resolved, ok := WorkspaceIDFromContext(r.Context())
				if !ok || resolved != workspace {
					t.Fatalf("workspace context missing: %v %v", resolved, ok)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			gateway := CollaborationAccessMiddleware(featurePolicyAccess{bodyPolicy: bodyPolicy{workspace: workspace}, owner: tc.owner})(next)
			var body io.ReadCloser = http.NoBody
			if tc.method == http.MethodPatch {
				body = io.NopCloser(strings.NewReader(`{"requestId":"` + uuid.NewString() + `"}`))
			}
			request := httptest.NewRequest(tc.method, "/api/v1/workspaces/"+workspace.String()+"/feature-policy", body)
			request = request.WithContext(context.WithValue(request.Context(), domain.ContextKeyUserID, user))
			response := &memoryDeadlineResponse{ResponseRecorder: httptest.NewRecorder()}
			gateway.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("status %d want %d: %s", response.Code, tc.want, response.Body.String())
			}
			if called != (tc.want == http.StatusNoContent) {
				t.Fatalf("handler called=%v for status %d", called, tc.want)
			}
		})
	}
}
