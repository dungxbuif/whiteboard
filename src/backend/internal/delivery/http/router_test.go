package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"devtool/internal/delivery/http/handler"
	"devtool/internal/domain"

	"github.com/google/uuid"
)

type routerFolderUsecase struct {
	domain.FolderUsecase
}

func (routerFolderUsecase) List(context.Context, uuid.UUID, *uuid.UUID) ([]domain.Folder, error) {
	return []domain.Folder{}, nil
}

func TestFolderRoutesRequireWorkspaceHeader(t *testing.T) {
	folderHandler := handler.NewFolderHandler(routerFolderUsecase{})
	passthroughAuth := func(next http.Handler) http.Handler { return next }
	router := NewRouter(nil, nil, folderHandler, passthroughAuth)

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
	router := NewRouter(nil, nil, folderHandler, passthroughAuth)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/folders/", nil)
	request.Header.Set(domain.HeaderWorkspaceID, uuid.NewString())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}
