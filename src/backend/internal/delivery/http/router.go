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
	return newRouter(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, nil, searchHandler, nil, nil, nil, nil, nil)
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
	realtimeHandlers ...nethttp.Handler,
) nethttp.Handler {
	return newRouterWithSharing(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, nil, searchHandler, sharingHandler, publicHandler, realtimeHandlers...)
}

// NewRouterWithSharingAndWorkspaceAccess wires ownership checks into every
// workspace-scoped route. The legacy constructor remains available for unit
// tests and callers that provide their own authorization layer.
func NewRouterWithSharingAndWorkspaceAccess(
	authHandler *handler.AuthHandler,
	workspaceHandler *handler.WorkspaceHandler,
	folderHandler *handler.FolderHandler,
	sceneHandler *handler.SceneHandler,
	assetHandler *handler.AssetHandler,
	authMiddleware func(http.Handler) http.Handler,
	workspaceAccessMiddleware func(http.Handler) http.Handler,
	searchHandler *handler.SearchHandler,
	sharingHandler *handler.SharingHandler,
	publicHandler *handler.PublicHandler,
	realtimeHandlers ...nethttp.Handler,
) nethttp.Handler {
	return newRouterWithSharing(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, workspaceAccessMiddleware, searchHandler, sharingHandler, publicHandler, realtimeHandlers...)
}

func newRouterWithSharing(
	authHandler *handler.AuthHandler,
	workspaceHandler *handler.WorkspaceHandler,
	folderHandler *handler.FolderHandler,
	sceneHandler *handler.SceneHandler,
	assetHandler *handler.AssetHandler,
	authMiddleware func(http.Handler) http.Handler,
	workspaceAccessMiddleware func(http.Handler) http.Handler,
	searchHandler *handler.SearchHandler,
	sharingHandler *handler.SharingHandler,
	publicHandler *handler.PublicHandler,
	realtimeHandlers ...nethttp.Handler,
) nethttp.Handler {
	var realtimeHandler nethttp.Handler
	var aiHandler nethttp.Handler
	var realtimeConfigHandler nethttp.Handler
	if len(realtimeHandlers) > 0 {
		realtimeHandler = realtimeHandlers[0]
	}
	if len(realtimeHandlers) > 1 {
		aiHandler = realtimeHandlers[1]
	}
	if len(realtimeHandlers) > 2 {
		realtimeConfigHandler = realtimeHandlers[2]
	}
	return newRouter(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, workspaceAccessMiddleware, searchHandler, sharingHandler, publicHandler, realtimeHandler, aiHandler, realtimeConfigHandler)
}

func newRouter(
	authHandler *handler.AuthHandler,
	workspaceHandler *handler.WorkspaceHandler,
	folderHandler *handler.FolderHandler,
	sceneHandler *handler.SceneHandler,
	assetHandler *handler.AssetHandler,
	authMiddleware func(http.Handler) http.Handler,
	workspaceAccessMiddleware func(http.Handler) http.Handler,
	searchHandler *handler.SearchHandler,
	sharingHandler *handler.SharingHandler,
	publicHandler *handler.PublicHandler,
	realtimeHandler nethttp.Handler,
	aiHandler nethttp.Handler,
	realtimeConfigHandler nethttp.Handler,
) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	router.Get("/healthz", healthHandler)
	if realtimeHandler != nil {
		router.Mount("/socket.io", realtimeHandler)
	}

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authHandler.PasswordLogin)
		if publicHandler != nil {
			r.Get("/public/scenes/{token}", publicHandler.GetScene)
		}

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			if searchHandler != nil {
				r.Route("/search", func(r chi.Router) {
					r.Use(deliverymiddleware.TenantMiddleware)
					if workspaceAccessMiddleware != nil {
						r.Use(workspaceAccessMiddleware)
					}
					r.Get("/", searchHandler.Search)
				})
			}

			r.Get("/users/me", authHandler.Me)
			if realtimeConfigHandler != nil {
				r.Get("/realtime/config", realtimeConfigHandler.ServeHTTP)
			}
			if aiHandler != nil {
				r.Mount("/ai", aiHandler)
			}

			r.Route("/workspaces", func(r chi.Router) {
				r.Get("/", workspaceHandler.List)
				r.Post("/", workspaceHandler.Create)
			})

			r.Route("/folders", func(r chi.Router) {
				r.Use(deliverymiddleware.TenantMiddleware)
				if workspaceAccessMiddleware != nil {
					r.Use(workspaceAccessMiddleware)
				}
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
				if workspaceAccessMiddleware != nil {
					r.Use(workspaceAccessMiddleware)
				}
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
				if workspaceAccessMiddleware != nil {
					r.Use(workspaceAccessMiddleware)
				}
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
