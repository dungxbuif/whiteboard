package http

import (
	"devtool/internal/delivery/http/handler"
	"encoding/json"
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(authHandler *handler.AuthHandler) nethttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	router.Get("/healthz", healthHandler)

	router.Route("/auth", func(r chi.Router) {
		r.Get("/google/login", authHandler.Login)
		r.Get("/google/callback", authHandler.Callback)
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
