package postgres

import (
	"context"
	delivery "devtool/internal/delivery/http"
	"devtool/internal/delivery/http/handler"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicNewLinkSnapshot(t *testing.T) {
	repo, links, rooms, access, owner, ref := publicLinkFixture(t)
	sceneUC := usecase.NewSceneUsecase(NewSceneRepository(repo.db), NewFolderRepository(repo.db))
	legacy := usecase.NewScenePermissionUsecase(NewScenePermissionRepository(repo.db), NewSceneRepository(repo.db))
	rh := handler.NewRoomHandler(rooms, access, sceneUC)
	router := delivery.NewRouterWithAccess(nil, nil, nil, handler.NewSceneHandler(sceneUC), handler.NewAssetHandler(nil), func(next http.Handler) http.Handler { return next }, access, nil, handler.NewSharingHandler(legacy, links), handler.NewPublicHandler(legacy, rh), nil, nil, nil)
	for _, mode := range []string{"view", "edit"} {
		link, err := links.Create(context.Background(), owner, ref, domain.CreateShareLink{Access: mode, Password: "secret-pass"})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", "/api/v1/public/scenes/"+link.Token, strings.NewReader(`{"password":"secret-pass","displayName":"Guest"}`))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"content"`) {
			t.Fatalf("%s snapshot: %d %s", mode, rr.Code, rr.Body.String())
		}
		if err := links.Revoke(context.Background(), owner, ref, link.ID); err != nil {
			t.Fatal(err)
		}
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest("POST", "/api/v1/public/scenes/"+link.Token, strings.NewReader(`{"password":"secret-pass","displayName":"Guest"}`)))
		if rr.Code == 200 {
			t.Fatal("revoked public snapshot admitted")
		}
	}
}
