package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SharingHandler struct{ usecase domain.ScenePermissionUsecase }

type ShareSceneRequest struct {
	Active   *bool  `json:"active"`
	Password string `json:"password"`
}

func NewSharingHandler(usecase domain.ScenePermissionUsecase) *SharingHandler {
	return &SharingHandler{usecase: usecase}
}

func (h *SharingHandler) Update(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	sceneID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid scene id", http.StatusBadRequest)
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

type PublicHandler struct{ usecase domain.ScenePermissionUsecase }

func NewPublicHandler(usecase domain.ScenePermissionUsecase) *PublicHandler {
	return &PublicHandler{usecase: usecase}
}

func (h *PublicHandler) GetScene(w http.ResponseWriter, r *http.Request) {
	password := r.URL.Query().Get("password")
	scene, err := h.usecase.GetPublicScene(r.Context(), chi.URLParam(r, "token"), password)
	if err != nil {
		if strings.Contains(err.Error(), "share password") {
			writeJSONResponse(w, http.StatusUnauthorized, map[string]any{"passwordRequired": true})
			return
		}
		http.Error(w, "Shared scene not found", http.StatusNotFound)
		return
	}
	writeJSONResponse(w, http.StatusOK, scene)
}
