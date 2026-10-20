package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type deniedSceneAccess struct{ domain.AccessService }

func (deniedSceneAccess) ResolveWorkspace(context.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error) {
	return domain.AccessDecision{}, domain.ErrSessionRevoked
}
func (deniedSceneAccess) Resolve(context.Context, domain.Principal, domain.SceneRef) (domain.AccessDecision, error) {
	return domain.AccessDecision{}, domain.ErrSessionRevoked
}

type createProbeSceneUsecase struct {
	domain.SceneUsecase
	called bool
}

func (s *createProbeSceneUsecase) Create(context.Context, uuid.UUID, uuid.UUID, string, *uuid.UUID, json.RawMessage) (*domain.Scene, error) {
	s.called = true
	return &domain.Scene{}, nil
}

func TestSceneCreateChecksWorkspaceAuthorityBeforeWrite(t *testing.T) {
	workspace, user := uuid.New(), uuid.New()
	probe := &createProbeSceneUsecase{}
	handler := NewSceneHandler(probe)
	handler.Access = deniedSceneAccess{}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/scenes", strings.NewReader(`{"title":"Blocked","content":{}}`))
	ctx := context.WithValue(request.Context(), domain.ContextKeyWorkspaceID, workspace)
	ctx = context.WithValue(ctx, domain.ContextKeyUserID, user)
	response := httptest.NewRecorder()
	handler.Create(response, request.WithContext(ctx))
	if probe.called {
		t.Fatal("scene write happened before workspace authority check")
	}
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
