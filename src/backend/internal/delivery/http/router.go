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
	authMiddleware func(http.Handler) http.Handler,
) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	router.Get("/healthz", healthHandler)

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)

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
					r.Delete("/", sceneHandler.Delete)
				})
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
