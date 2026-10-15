package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtool/internal/delivery/http/handler"
	"devtool/internal/domain"

	"github.com/google/uuid"
)

type routerFolderUsecase struct {
	domain.FolderUsecase
}

type routerSceneUsecase struct {
	domain.SceneUsecase
}

func (routerSceneUsecase) List(_ context.Context, workspaceID uuid.UUID, _ *uuid.UUID, _ bool) ([]domain.Scene, error) {
	return []domain.Scene{{ID: uuid.New(), WorkspaceID: workspaceID, Title: "Board"}}, nil
}

func (routerFolderUsecase) ListAll(_ context.Context, workspaceID uuid.UUID) ([]domain.Folder, error) {
	return []domain.Folder{{ID: uuid.New(), WorkspaceID: workspaceID, Name: "Folder"}}, nil
}

func (routerFolderUsecase) List(context.Context, uuid.UUID, *uuid.UUID) ([]domain.Folder, error) {
	return []domain.Folder{}, nil
}

func TestFolderRoutesListAllWithCamelCaseJSON(t *testing.T) {
	folderHandler := handler.NewFolderHandler(routerFolderUsecase{})
	passthroughAuth := func(next http.Handler) http.Handler { return next }
	router := NewRouter(nil, nil, folderHandler, nil, nil, passthroughAuth)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/folders/?all=true", nil)
	request.Header.Set(domain.HeaderWorkspaceID, uuid.NewString())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"workspaceId"`) {
		t.Fatalf("expected camelCase workspaceId, got %s", response.Body.String())
	}
}

func TestFolderRoutesRequireWorkspaceHeader(t *testing.T) {
	folderHandler := handler.NewFolderHandler(routerFolderUsecase{})
	passthroughAuth := func(next http.Handler) http.Handler { return next }
	router := NewRouter(nil, nil, folderHandler, nil, nil, passthroughAuth)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/folders/", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestFolderRoutesInjectWorkspaceID(t *testing.T) {
	folderHandler := handler.NewFolderHandler(routerFolderUsecase{})
	passthroughAuth := func(next http.Handler) http.Handler { return next }
	router := NewRouter(nil, nil, folderHandler, nil, nil, passthroughAuth)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/folders/", nil)
	request.Header.Set(domain.HeaderWorkspaceID, uuid.NewString())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestSceneRoutesRequireWorkspaceHeader(t *testing.T) {
	sceneHandler := handler.NewSceneHandler(routerSceneUsecase{})
	passthroughAuth := func(next http.Handler) http.Handler { return next }
	router := NewRouter(nil, nil, nil, sceneHandler, nil, passthroughAuth)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestSceneRoutesReturnCamelCaseJSON(t *testing.T) {
	sceneHandler := handler.NewSceneHandler(routerSceneUsecase{})
	passthroughAuth := func(next http.Handler) http.Handler { return next }
	router := NewRouter(nil, nil, nil, sceneHandler, nil, passthroughAuth)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/", nil)
	request.Header.Set(domain.HeaderWorkspaceID, uuid.NewString())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"workspaceId"`) || !strings.Contains(response.Body.String(), `"contentVersion"`) {
		t.Fatalf("expected camelCase scene JSON, got %s", response.Body.String())
	}
}
