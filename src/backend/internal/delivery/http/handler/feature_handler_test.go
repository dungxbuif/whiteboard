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

type featureServiceStub struct {
	policy domain.WorkspaceFeaturePolicy
	patch  domain.WorkspaceFeaturePolicyPatch
}

func (s *featureServiceStub) Require(context.Context, domain.Principal, domain.SceneRef, domain.FeatureAction) error {
	return nil
}
func (s *featureServiceStub) Policy(context.Context, domain.Principal, uuid.UUID) (domain.WorkspaceFeaturePolicy, error) {
	return s.policy, nil
}
func (s *featureServiceStub) UpdatePolicy(_ context.Context, _ domain.Principal, _ uuid.UUID, _ int64, _ uuid.UUID, patch domain.WorkspaceFeaturePolicyPatch) (domain.WorkspaceFeaturePolicy, error) {
	s.patch = patch
	s.policy.Revision++
	return s.policy, nil
}
func (s *featureServiceStub) Mutate(context.Context, domain.Principal, domain.SceneRef, domain.FeatureAction, uuid.UUID, []byte, func(context.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	return domain.StoredFeatureResponse{}, nil
}

func TestFeaturePolicyHandlerUsesDecimalRevision(t *testing.T) {
	workspace, user := uuid.New(), uuid.New()
	service := &featureServiceStub{policy: domain.DefaultWorkspaceFeaturePolicy(workspace)}
	handler := NewFeatureHandler(service)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+workspace.String()+"/feature-policy", strings.NewReader(`{"requestId":"`+uuid.NewString()+`","expectedRevision":"1","commentsEnabled":true}`))
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), domain.ContextKeyWorkspaceID, workspace), domain.ContextKeyUserID, user))
	response := httptest.NewRecorder()
	handler.Update(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"revision":"2"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if service.patch.CommentsEnabled == nil || !*service.patch.CommentsEnabled {
		t.Fatalf("patch not forwarded: %+v", service.patch)
	}
}

func TestFeaturePolicyHandlerRejectsUnknownFields(t *testing.T) {
	workspace, user := uuid.New(), uuid.New()
	handler := NewFeatureHandler(&featureServiceStub{policy: domain.DefaultWorkspaceFeaturePolicy(workspace)})
	request := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"requestId":"`+uuid.NewString()+`","expectedRevision":"1","surprise":true}`))
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), domain.ContextKeyWorkspaceID, workspace), domain.ContextKeyUserID, user))
	response := httptest.NewRecorder()
	handler.Update(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
