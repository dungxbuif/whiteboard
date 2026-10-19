package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"devtool/internal/delivery/http/handler"
	deliverymiddleware "devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

type deletedRouterUserStore struct{ userID uuid.UUID }

func (s deletedRouterUserStore) FindByID(context.Context, uuid.UUID) (*domain.User, error) {
	return &domain.User{ID: s.userID, DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}}, nil
}

func TestDeletedJWTSubjectCannotReachWorkspaceCreationOrAI(t *testing.T) {
	secret := []byte("login-secret-0123456789abcdef0123456789")
	userID := uuid.New()
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(), "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	called := 0
	ai := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	})
	auth := deliverymiddleware.AuthMiddleware(secret, deletedRouterUserStore{userID: userID})
	router := newRouter(nil, handler.NewWorkspaceHandler(nil), nil, nil, nil, auth, nil, nil, nil, nil, nil, ai, nil, nil)

	for _, path := range []string{"/api/v1/workspaces/", "/api/v1/ai/generate"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		req.Header.Set(domain.HeaderAuthorization, domain.BearerPrefix+raw)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}
	if called != 0 {
		t.Fatalf("protected handler calls=%d, want 0", called)
	}
}
