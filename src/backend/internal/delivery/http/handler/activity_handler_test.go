package handler

import (
	"context"
	"devtool/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type activityUsecaseStub struct {
	workspaceID uuid.UUID
	sceneID     *uuid.UUID
	page        domain.ActivityPage
}

func (s *activityUsecaseStub) List(_ domain.Context, _ domain.Principal, workspaceID uuid.UUID, sceneID *uuid.UUID, _ string, _ int) (domain.ActivityPage, error) {
	s.workspaceID, s.sceneID = workspaceID, sceneID
	return s.page, nil
}

type activityHandlerAccessStub struct {
	workspaceID uuid.UUID
	sceneID     uuid.UUID
}

func (s activityHandlerAccessStub) ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error) {
	return s.workspaceID, nil
}
func (s activityHandlerAccessStub) ResolveSceneID(domain.Context, uuid.UUID, string) (uuid.UUID, error) {
	return s.sceneID, nil
}

func TestActivityHandlerBindsWorkspaceAndSceneRoutes(t *testing.T) {
	workspaceID, sceneID, userID := uuid.New(), uuid.New(), uuid.New()
	uc := &activityUsecaseStub{page: domain.ActivityPage{Items: []domain.ActivityItem{}, RetainedSince: time.Now().UTC()}}
	h := NewActivityHandler(uc, activityHandlerAccessStub{workspaceID: workspaceID, sceneID: sceneID})
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, userID)))
		})
	}
	router := chi.NewRouter()
	h.Register(router, auth)

	request := httptest.NewRequest(http.MethodGet, "/scenes/scene-alias/activity?limit=25", nil)
	request.Header.Set(domain.HeaderWorkspaceID, "workspace-alias")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || uc.workspaceID != workspaceID || uc.sceneID == nil || *uc.sceneID != sceneID {
		t.Fatalf("scene activity status=%d workspace=%s scene=%v body=%s", response.Code, uc.workspaceID, uc.sceneID, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/workspaces/workspace-alias/activity", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || uc.sceneID != nil {
		t.Fatalf("workspace activity status=%d scene=%v body=%s", response.Code, uc.sceneID, response.Body.String())
	}
}

func TestActivityHandlerRejectsInvalidLimit(t *testing.T) {
	h := NewActivityHandler(&activityUsecaseStub{}, activityHandlerAccessStub{workspaceID: uuid.New(), sceneID: uuid.New()})
	auth := func(next http.Handler) http.Handler { return next }
	router := chi.NewRouter()
	h.Register(router, auth)
	request := httptest.NewRequest(http.MethodGet, "/workspaces/x/activity?limit=51", nil)
	request = request.WithContext(context.WithValue(request.Context(), domain.ContextKeyUserID, uuid.New()))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
