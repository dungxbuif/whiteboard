package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"

	"github.com/go-chi/chi/v5"
)

type SharingHandler struct {
	usecase domain.ScenePermissionUsecase
	Links   *ShareLinkHandler
}

type ShareSceneRequest struct {
	Active   *bool  `json:"active"`
	Password string `json:"password"`
}

func NewSharingHandler(usecase domain.ScenePermissionUsecase, links ...domain.ShareLinkService) *SharingHandler {
	h := &SharingHandler{usecase: usecase}
	if len(links) > 0 {
		h.Links = NewShareLinkHandler(links[0])
	}
	return h
}

func (h *SharingHandler) Update(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	sceneID, ok := parseURLUUID(w, r)
	if !ok {
		return
	}
	var request ShareSceneRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Active == nil {
		http.Error(w, "active is required", http.StatusBadRequest)
		return
	}
	permission, err := h.usecase.Share(r.Context(), workspaceID, sceneID, *request.Active, request.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"active": permission.Active,
		"token":  permission.Token,
	})
}

type PublicHandler struct {
	usecase domain.ScenePermissionUsecase
	Rooms   *RoomHandler
}

func NewPublicHandler(usecase domain.ScenePermissionUsecase, rooms ...*RoomHandler) *PublicHandler {
	h := &PublicHandler{usecase: usecase}
	if len(rooms) > 0 {
		h.Rooms = rooms[0]
	}
	return h
}

func (h *PublicHandler) GetScene(w http.ResponseWriter, r *http.Request) {
	password := ""
	displayName := ""
	if r.Method == http.MethodPost {
		var input struct {
			Password    string `json:"password" validate:"max=72"`
			DisplayName string `json:"displayName,omitempty" validate:"max=100"`
		}
		if !decodeSharing(w, r, &input) {
			return
		}
		password = input.Password
		displayName = input.DisplayName
	}
	var scene *domain.Scene
	var err error
	read := func(ctx domain.Context) error {
		var e error
		scene, e = h.usecase.GetPublicScene(ctx, chi.URLParam(r, "token"), password)
		return e
	}
	if h.Rooms != nil && h.Rooms.scenes != nil {
		scene, err = h.Rooms.readPublicSnapshot(r.Context(), chi.URLParam(r, "token"), password, displayName)
	} else if h.Rooms != nil {
		err = h.Rooms.withPublicRead(r.Context(), chi.URLParam(r, "token"), password, read)
	} else {
		err = read(r.Context())
	}
	if err != nil {
		if strings.Contains(err.Error(), "share password") {
			sharingError(w, r, domain.ErrSharePassword)
			return
		}
		sharingError(w, r, domain.ErrInaccessible)
		return
	}
	writeJSONResponse(w, http.StatusOK, scene)
}
