package http

import (
	"devtool/internal/delivery/http/handler"
	deliverymiddleware "devtool/internal/delivery/http/middleware"
	"encoding/json"
	"net/http"
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	authHandler *handler.AuthHandler,
	workspaceHandler *handler.WorkspaceHandler,
	folderHandler *handler.FolderHandler,
	sceneHandler *handler.SceneHandler,
	assetHandler *handler.AssetHandler,
	authMiddleware func(http.Handler) http.Handler,
	searchHandlers ...*handler.SearchHandler,
) nethttp.Handler {
	var searchHandler *handler.SearchHandler
	if len(searchHandlers) > 0 {
		searchHandler = searchHandlers[0]
	}
	return newRouter(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, searchHandler, nil, nil)
}

func NewRouterWithSharing(
	authHandler *handler.AuthHandler,
	workspaceHandler *handler.WorkspaceHandler,
	folderHandler *handler.FolderHandler,
	sceneHandler *handler.SceneHandler,
	assetHandler *handler.AssetHandler,
	authMiddleware func(http.Handler) http.Handler,
	searchHandler *handler.SearchHandler,
	sharingHandler *handler.SharingHandler,
	publicHandler *handler.PublicHandler,
) nethttp.Handler {
	return newRouter(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, searchHandler, sharingHandler, publicHandler)
}

func newRouter(
	authHandler *handler.AuthHandler,
	workspaceHandler *handler.WorkspaceHandler,
	folderHandler *handler.FolderHandler,
	sceneHandler *handler.SceneHandler,
	assetHandler *handler.AssetHandler,
	authMiddleware func(http.Handler) http.Handler,
	searchHandler *handler.SearchHandler,
	sharingHandler *handler.SharingHandler,
	publicHandler *handler.PublicHandler,
) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	router.Get("/healthz", healthHandler)

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/auth/google/login", authHandler.GoogleLogin)
		r.Get("/auth/google/callback", authHandler.GoogleCallback)
		if publicHandler != nil {
			r.Get("/public/scenes/{token}", publicHandler.GetScene)
		}

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			if searchHandler != nil {
				r.Route("/search", func(r chi.Router) {
					r.Use(deliverymiddleware.TenantMiddleware)
					r.Get("/", searchHandler.Search)
				})
			}

			r.Get("/users/me", authHandler.Me)

			r.Route("/workspaces", func(r chi.Router) {
				r.Get("/", workspaceHandler.List)
				r.Post("/", workspaceHandler.Create)
			})

			r.Route("/folders", func(r chi.Router) {
				r.Use(deliverymiddleware.TenantMiddleware)
				r.Get("/", folderHandler.List)
				r.Post("/", folderHandler.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Patch("/move", folderHandler.Move)
					r.Post("/restore", folderHandler.Restore)
					r.Post("/star", folderHandler.ToggleStar)
					r.Delete("/", folderHandler.Delete)
				})
			})

			r.Route("/scenes", func(r chi.Router) {
				r.Use(deliverymiddleware.TenantMiddleware)
				r.Get("/", sceneHandler.List)
				r.Post("/", sceneHandler.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", sceneHandler.Get)
					r.Patch("/", sceneHandler.Update)
					r.Post("/restore", sceneHandler.Restore)
					r.Post("/star", sceneHandler.SetStar)
					if sharingHandler != nil {
						r.Put("/share", sharingHandler.Update)
					}
					r.Delete("/", sceneHandler.Delete)
				})
			})

			r.Route("/assets", func(r chi.Router) {
				r.Use(deliverymiddleware.TenantMiddleware)
				r.Post("/presign-upload", assetHandler.PresignUpload)
				r.Post("/presign-view", assetHandler.PresignView)
			})
		})
	})

	return router
}

func healthHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	writeJSON(w, nethttp.StatusOK, map[string]string{
		"status": "ok",
	})
}

func writeJSON(w nethttp.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
