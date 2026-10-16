package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

type RoomHandler struct {
	rooms  domain.RoomService
	access domain.AccessService
}

func NewRoomHandler(rooms domain.RoomService, access domain.AccessService) *RoomHandler {
	return &RoomHandler{rooms: rooms, access: access}
}

type publicSessionRequest struct {
	Token       string `json:"token" validate:"required,max=256"`
	Password    string `json:"password,omitempty" validate:"max=72"`
	DisplayName string `json:"displayName,omitempty" validate:"max=100"`
}

func (h *RoomHandler) Exchange(w http.ResponseWriter, r *http.Request) {
	var input publicSessionRequest
	if !decodeSharing(w, r, &input) {
		return
	}
	result, e := h.rooms.ExchangePublicSession(r.Context(), input.Token, input.Password, input.DisplayName)
	if e != nil {
		sharingError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, result)
}
func (h *RoomHandler) mint(w http.ResponseWriter, r *http.Request, p domain.Principal, ref domain.SceneRef) {
	result, e := h.rooms.MintRoomToken(r.Context(), p, ref)
	if e != nil {
		sharingError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, result)
}

// Only this route accepts a scene-session credential. It never populates UserID.
func (h *RoomHandler) TokenHandler(auth func(http.Handler) http.Handler) http.Handler {
	login := auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			sharingError(w, r, domain.ErrInvalidSceneSession)
			return
		}
		workspace, e := h.access.ResolveWorkspaceID(r.Context(), r.Header.Get(domain.HeaderWorkspaceID))
		if e != nil {
			sharingError(w, r, e)
			return
		}
		scene, e := h.access.ResolveSceneID(r.Context(), workspace, chi.URLParam(r, "id"))
		if e != nil {
			sharingError(w, r, e)
			return
		}
		h.mint(w, r, domain.UserPrincipal(user), domain.SceneRef{WorkspaceID: workspace, SceneID: scene})
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get(domain.HeaderAuthorization)
		if strings.HasPrefix(raw, domain.BearerPrefix) {
			identity, e := h.rooms.ParseSceneSession(strings.TrimPrefix(raw, domain.BearerPrefix))
			if e == nil {
				scene, e := h.access.ResolveSceneID(r.Context(), identity.SceneRef.WorkspaceID, chi.URLParam(r, "id"))
				if e != nil || scene != identity.SceneRef.SceneID {
					sharingError(w, r, domain.ErrInaccessible)
					return
				}
				if workspace := r.Header.Get(domain.HeaderWorkspaceID); workspace != "" {
					id, e := h.access.ResolveWorkspaceID(r.Context(), workspace)
					if e != nil || id != identity.SceneRef.WorkspaceID {
						sharingError(w, r, domain.ErrInaccessible)
						return
					}
				}
				h.mint(w, r, identity.Principal, identity.SceneRef)
				return
			}
		}
		login.ServeHTTP(w, r)
	})
}

func (h *RoomHandler) withPublicRead(ctx domain.Context, token, password string, read func(domain.Context) error) error {
	session, e := h.rooms.ExchangePublicSession(ctx, token, password, "")
	if e != nil {
		return e
	}
	identity, e := h.rooms.ParseSceneSession(session.Token)
	if e != nil {
		return e
	}
	return h.access.WithWorkspaceLock(ctx, identity.SceneRef.WorkspaceID, uuid.Nil, func(ctx domain.Context) error {
		if _, e := h.access.Resolve(ctx, identity.Principal, identity.SceneRef); e != nil {
			return e
		}
		return read(ctx)
	})
}
