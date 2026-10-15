package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"devtool/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockSharingUsecase struct {
	domain.ScenePermissionUsecase
	scene      *domain.Scene
	permission *domain.ScenePermission
}

func (m *mockSharingUsecase) Share(context.Context, uuid.UUID, uuid.UUID, bool, string) (*domain.ScenePermission, error) {
	return m.permission, nil
}

func (m *mockSharingUsecase) GetPublicScene(context.Context, string, string) (*domain.Scene, error) {
	return m.scene, nil
}

func TestPublicHandlerReturnsReadOnlyScene(t *testing.T) {
	mock := &mockSharingUsecase{scene: &domain.Scene{ID: uuid.New(), Title: "Shared"}}
	handler := NewPublicHandler(mock)
	router := chi.NewRouter()
	router.Get("/api/v1/public/scenes/{token}", handler.GetScene)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/scenes/token", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	var scene domain.Scene
	if err := json.NewDecoder(response.Body).Decode(&scene); err != nil {
		t.Fatalf("decode scene: %v", err)
	}
	if scene.Title != "Shared" {
		t.Fatalf("unexpected scene: %#v", scene)
	}
}
