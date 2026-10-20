package http

import (
	"devtool/internal/delivery/http/handler"
	deliverymiddleware "devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
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
	return newRouter(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, nil, searchHandler, nil, nil, nil, nil, nil, nil)
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
	return newRouter(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, workspaceAccessMiddleware, searchHandler, sharingHandler, publicHandler, realtimeHandler, aiHandler, realtimeConfigHandler, nil)
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
	invitationHandler *handler.InvitationHandler,
	services ...routerServices,
) nethttp.Handler {
	var access domain.AccessService
	var accessHandler *handler.AccessHandler
	var agents *handler.AgentHandler
	var features *handler.FeatureHandler
	var comments *handler.CommentHandler
	var versions *handler.VersionHandler
	var notifications *handler.NotificationHandler
	var activity *handler.ActivityHandler
	if len(services) > 0 {
		access = services[0].access
		agents = services[0].agents
		features = services[0].features
		comments = services[0].comments
		versions = services[0].versions
		notifications = services[0].notifications
		activity = services[0].activity
		accessHandler = handler.NewAccessHandler(access)
	}
	if sceneHandler != nil {
		sceneHandler.Access = access
	}
	router := chi.NewRouter()
	router.Use(deliverymiddleware.RedactPublicCredentials)
	router.Use(middleware.RequestID)
	router.Use(deliverymiddleware.CapturePeerIP)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	router.Get("/healthz", healthHandler)
	// Realtime collaboration is served by the dedicated custom WebSocket
	// runtime at /collab/. The retired Socket.IO handler is intentionally not
	// mounted on the Go API.

	router.Route("/api/v1", func(r chi.Router) {
		if activity != nil {
			activity.Register(r, authMiddleware)
		}
		if notifications != nil {
			notifications.Register(r, authMiddleware)
		}
		if versions != nil {
			versions.Register(r, authMiddleware)
		}
		if comments != nil {
			comments.Register(r, authMiddleware)
		}
		if agents != nil {
			r.Mount("/agent", agents.Routes())
		}
		if assetHandler != nil && assetHandler.Capabilities != nil {
			cap := assetHandler.Capabilities
			r.With(deliverymiddleware.InvitationBodyMiddleware).Method(http.MethodPost, "/scenes/{id}/assets/upload-capability", cap.UploadHandler(authMiddleware))
			r.With(deliverymiddleware.InvitationBodyMiddleware).Post("/public/scenes/{id}/assets/presign-upload", cap.PublicUpload)
			r.With(deliverymiddleware.InvitationBodyMiddleware).Post("/public/scenes/{id}/assets/presign-view", cap.PublicView)
			r.Get("/realtime/config", cap.RealtimeConfig)
		}
		r.Post("/auth/login", authHandler.PasswordLogin)
		if invitationHandler != nil {
			r.With(deliverymiddleware.InvitationBodyMiddleware).Post("/invitations/register", invitationHandler.Register)
		}
		if publicHandler != nil {
			r.Get("/public/scenes/{token}", publicHandler.GetScene)
			r.Post("/public/scenes/{token}", publicHandler.GetScene)
			if publicHandler.Rooms != nil {
				r.Post("/public/scene-session", publicHandler.Rooms.Exchange)
				r.Method(http.MethodPost, "/scenes/{id}/room-token", publicHandler.Rooms.TokenHandler(authMiddleware))
			}
		}

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			if access != nil {
				r.Use(deliverymiddleware.CollaborationAccessMiddleware(access))
			}
			if searchHandler != nil {
				r.Route("/search", func(r chi.Router) {
					r.Use(deliverymiddleware.TenantMiddleware)
					if workspaceAccessMiddleware != nil {
						r.Use(workspaceAccessMiddleware)
					}
					r.Get("/", searchHandler.Search)
				})
			}

			if invitationHandler != nil {
				r.With(deliverymiddleware.InvitationBodyMiddleware).Post("/invitations/accept", invitationHandler.Accept)
			}
			r.Get("/users/me", authHandler.Me)
			if realtimeConfigHandler != nil && (assetHandler == nil || assetHandler.Capabilities == nil) {
				r.Get("/realtime/config", realtimeConfigHandler.ServeHTTP)
			}
			if aiHandler != nil {
				r.Mount("/ai", aiHandler)
			}

			r.Route("/workspaces", func(r chi.Router) {
				if features != nil {
					r.Get("/{id}/feature-policy", features.Get)
					r.Patch("/{id}/feature-policy", features.Update)
				}
				if agents != nil {
					r.Get("/{id}/agent-keys", agents.List)
					r.Post("/{id}/agent-keys", agents.Create)
					r.Delete("/{id}/agent-keys/{keyId}", agents.Revoke)
				}
				if invitationHandler != nil {
					r.Get("/{id}/invitations", invitationHandler.List)
					r.Post("/{id}/invitations", invitationHandler.Create)
					r.Delete("/{id}/invitations/{inviteId}", invitationHandler.Revoke)
				}
				r.Get("/", workspaceHandler.List)
				r.Post("/", workspaceHandler.Create)
				if accessHandler != nil {
					r.Get("/{id}/members", accessHandler.Members)
					r.Patch("/{id}/members/{userId}", accessHandler.Member)
					r.Delete("/{id}/members/{userId}", accessHandler.Member)
				}
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
					if accessHandler != nil {
						r.Put("/members/{userId}", accessHandler.Override)
						r.Delete("/members/{userId}", accessHandler.Override)
					}
					r.Patch("/", sceneHandler.Update)
					r.Post("/restore", sceneHandler.Restore)
					r.Post("/star", sceneHandler.SetStar)
					if sharingHandler != nil {
						r.Put("/share", sharingHandler.Update)
						if sharingHandler.Links != nil {
							r.Get("/share-links", sharingHandler.Links.List)
							r.Post("/share-links", sharingHandler.Links.Create)
							r.Delete("/share-links/{linkId}", sharingHandler.Links.Revoke)
						}
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

// NewRouterWithAccess is the production router; legacy constructors are retained
// for v1 tests while the new gateway enforces every scoped HTTP request.
func NewRouterWithAccess(auth *handler.AuthHandler, workspaces *handler.WorkspaceHandler, folders *handler.FolderHandler, scenes *handler.SceneHandler, assets *handler.AssetHandler, authMiddleware func(http.Handler) http.Handler, access domain.AccessService, search *handler.SearchHandler, sharing *handler.SharingHandler, public *handler.PublicHandler, realtime, ai, config http.Handler, invitations ...*handler.InvitationHandler) http.Handler {
	assets.SetAccessService(access)
	var invitationHandler *handler.InvitationHandler
	if len(invitations) > 0 {
		invitationHandler = invitations[0]
	}
	return newRouter(auth, workspaces, folders, scenes, assets, authMiddleware, nil, search, sharing, public, realtime, ai, config, invitationHandler, routerServices{access: access})
}

func NewRouterWithAgents(auth *handler.AuthHandler, workspaces *handler.WorkspaceHandler, folders *handler.FolderHandler, scenes *handler.SceneHandler, assets *handler.AssetHandler, authMiddleware func(http.Handler) http.Handler, access domain.AccessService, search *handler.SearchHandler, sharing *handler.SharingHandler, public *handler.PublicHandler, realtime, ai, config http.Handler, invitations *handler.InvitationHandler, agents *handler.AgentHandler, featureHandlers ...*handler.FeatureHandler) http.Handler {
	assets.SetAccessService(access)
	var features *handler.FeatureHandler
	if len(featureHandlers) > 0 {
		features = featureHandlers[0]
	}
	return newRouter(auth, workspaces, folders, scenes, assets, authMiddleware, nil, search, sharing, public, realtime, ai, config, invitations, routerServices{access: access, agents: agents, features: features})
}

func NewRouterWithEnterprise(auth *handler.AuthHandler, workspaces *handler.WorkspaceHandler, folders *handler.FolderHandler, scenes *handler.SceneHandler, assets *handler.AssetHandler, authMiddleware func(http.Handler) http.Handler, access domain.AccessService, search *handler.SearchHandler, sharing *handler.SharingHandler, public *handler.PublicHandler, realtime, ai, config http.Handler, invitations *handler.InvitationHandler, agents *handler.AgentHandler, features *handler.FeatureHandler, comments *handler.CommentHandler, versions *handler.VersionHandler, notifications *handler.NotificationHandler, activity *handler.ActivityHandler) http.Handler {
	assets.SetAccessService(access)
	return newRouter(auth, workspaces, folders, scenes, assets, authMiddleware, nil, search, sharing, public, realtime, ai, config, invitations, routerServices{access: access, agents: agents, features: features, comments: comments, versions: versions, notifications: notifications, activity: activity})
}

type routerServices struct {
	access        domain.AccessService
	agents        *handler.AgentHandler
	features      *handler.FeatureHandler
	comments      *handler.CommentHandler
	versions      *handler.VersionHandler
	notifications *handler.NotificationHandler
	activity      *handler.ActivityHandler
}
