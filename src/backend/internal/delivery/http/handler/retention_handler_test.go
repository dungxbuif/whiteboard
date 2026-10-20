package handler

import (
	"context"
	"devtool/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type retentionUsecaseStub struct {
	calls      int
	requestIDs []uuid.UUID
	workspace  uuid.UUID
	policy     domain.RetentionPolicyInput
	holdRef    domain.RetentionReference
}

func (s *retentionUsecaseStub) Policy(domain.Context, domain.Principal, uuid.UUID) (domain.RetentionPolicy, error) {
	s.calls++
	return domain.RetentionPolicy{WorkspaceID: s.workspace, Revision: 0, Policy: domain.DefaultRetentionPolicyInput()}, nil
}
func (s *retentionUsecaseStub) PreviewPolicy(_ domain.Context, _ domain.Principal, w uuid.UUID, input domain.RetentionPolicyInput) (domain.RetentionPreview, error) {
	s.calls++
	s.policy = input
	return domain.RetentionPreview{ID: uuid.New(), WorkspaceID: w, Policy: input, PolicyRevision: 0, EarliestDestructiveAt: time.Now().Add(7 * 24 * time.Hour)}, nil
}
func (s *retentionUsecaseStub) Schedule(_ domain.Context, _ domain.Principal, w, p uuid.UUID, r int64, id uuid.UUID) (domain.RetentionPolicy, error) {
	s.calls++
	s.requestIDs = append(s.requestIDs, id)
	return domain.RetentionPolicy{WorkspaceID: w, Revision: r + 1, Policy: domain.DefaultRetentionPolicyInput()}, nil
}

func TestRetentionHandlerForwardsManualPolicyAndSceneHold(t *testing.T) {
	w, u := uuid.New(), uuid.New()
	uc := &retentionUsecaseStub{workspace: w}
	router := retentionTestRouter(uc, w, u)
	request := httptest.NewRequest(http.MethodPost, "/workspaces/w/retention/previews", strings.NewReader(`{"activityDays":90,"notificationDays":30,"automaticVersionLimit":100,"manualVersionDays":45,"trashHardPurge":false}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || uc.policy.ManualVersionDays == nil || *uc.policy.ManualVersionDays != 45 {
		t.Fatalf("preview status=%d policy=%+v body=%s", response.Code, uc.policy, response.Body.String())
	}
	sceneID := uuid.New()
	request = httptest.NewRequest(http.MethodPut, "/workspaces/w/retention/holds", strings.NewReader(`{"kind":"scene","id":"`+sceneID.String()+`","held":true,"requestId":"`+uuid.NewString()+`"}`))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || uc.holdRef.Kind != "scene" || uc.holdRef.ID != sceneID.String() {
		t.Fatalf("hold status=%d ref=%+v body=%s", response.Code, uc.holdRef, response.Body.String())
	}
}
func (s *retentionUsecaseStub) SetHold(_ domain.Context, _ domain.Principal, ref domain.RetentionReference, v *uuid.UUID, held bool, id uuid.UUID) (domain.RetentionHold, error) {
	s.calls++
	s.requestIDs = append(s.requestIDs, id)
	s.holdRef = ref
	return domain.RetentionHold{ID: uuid.New(), Reference: ref, VersionID: v, Held: held}, nil
}
func (s *retentionUsecaseStub) PurgeBatch(domain.Context, time.Time, int) (domain.PurgeResult, error) {
	return domain.PurgeResult{}, nil
}

type retentionHandlerAccessStub struct{ workspace uuid.UUID }

func (s retentionHandlerAccessStub) ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error) {
	return s.workspace, nil
}
func retentionTestRouter(uc *retentionUsecaseStub, w, u uuid.UUID) *chi.Mux {
	router := chi.NewRouter()
	NewRetentionHandler(uc, retentionHandlerAccessStub{workspace: w}).Register(router, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(rw, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, u)))
		})
	})
	return router
}
func TestRetentionHandlerRejectsUnknownTrailingAndInvalidIDs(t *testing.T) {
	w, u := uuid.New(), uuid.New()
	tests := []struct{ method, path, body string }{{"POST", "/workspaces/w/retention/previews", `{"activityDays":90,"notificationDays":30,"automaticVersionLimit":100,"trashHardPurge":false,"extra":1}`}, {"POST", "/workspaces/w/retention/schedule", `{"previewId":"` + uuid.NewString() + `","expectedRevision":"0","requestId":"` + uuid.NewString() + `"}{}`}, {"PUT", "/workspaces/w/retention/holds", `{"kind":"version","id":"x","held":true,"requestId":"bad"}`}}
	for _, tt := range tests {
		uc := &retentionUsecaseStub{workspace: w}
		req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
		rec := httptest.NewRecorder()
		retentionTestRouter(uc, w, u).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest || uc.calls != 0 {
			t.Fatalf("%s status=%d calls=%d body=%s", tt.path, rec.Code, uc.calls, rec.Body.String())
		}
	}
}
func TestRetentionHandlerForwardsStableRequestID(t *testing.T) {
	w, u, requestID, previewID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	uc := &retentionUsecaseStub{workspace: w}
	router := retentionTestRouter(uc, w, u)
	body := `{"previewId":"` + previewID.String() + `","expectedRevision":"0","requestId":"` + requestID.String() + `"}`
	for range 2 {
		req := httptest.NewRequest(http.MethodPost, "/workspaces/w/retention/schedule", strings.NewReader(body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
	if len(uc.requestIDs) != 2 || uc.requestIDs[0] != requestID || uc.requestIDs[1] != requestID {
		t.Fatalf("ids=%v", uc.requestIDs)
	}
}
