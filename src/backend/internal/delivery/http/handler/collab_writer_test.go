package handler

import (
	"context"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fenceSceneRepo struct {
	domain.SceneRepository
	scene *domain.Scene
}

func (r *fenceSceneRepo) FindByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Scene, error) {
	return r.scene, nil
}
func (r *fenceSceneRepo) Update(_ context.Context, s *domain.Scene) error { r.scene = s; return nil }
func TestV2SceneHTTPContentConflictAndMetadata(t *testing.T) {
	sid := uuid.New()
	repo := &fenceSceneRepo{scene: &domain.Scene{ID: sid, WriterMode: "v2", Title: "Existing", Content: json.RawMessage(`{"elements":[]}`)}}
	h := NewSceneHandler(usecase.NewSceneUsecase(repo, nil))
	for _, tc := range []struct {
		body   string
		status int
	}{{`{"content":{"elements":[]}}`, 409}, {`{"title":"Renamed"}`, 200}} {
		request := httptest.NewRequest(http.MethodPatch, "/scenes/"+sid.String(), strings.NewReader(tc.body))
		request = request.WithContext(context.WithValue(request.Context(), domain.ContextKeySceneID, sid))
		response := httptest.NewRecorder()
		h.Update(response, request)
		if response.Code != tc.status {
			t.Fatalf("expected %d got %d: %s", tc.status, response.Code, response.Body.String())
		}
		if tc.status == 409 && !strings.Contains(response.Body.String(), `"code":"COLLAB_REQUIRED"`) {
			t.Fatal("missing conflict code")
		}
	}
}
