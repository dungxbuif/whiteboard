package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
)

type AssetHandler struct {
	access       domain.AccessService
	assetUsecase domain.AssetUsecase
}

func NewAssetHandler(assetUsecase domain.AssetUsecase) *AssetHandler {
	return &AssetHandler{assetUsecase: assetUsecase}
}

type PresignUploadRequest struct {
	SceneID string                      `json:"sceneId"`
	Assets  []domain.AssetUploadRequest `json:"assets"`
}

type PresignViewRequest struct {
	SceneID string   `json:"sceneId"`
	FileIDs []string `json:"fileIds"`
}

func (h *AssetHandler) PresignUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workspaceIDStr := r.Header.Get("X-Workspace-ID")
	if workspaceIDStr == "" {
		http.Error(w, "missing X-Workspace-ID", http.StatusBadRequest)
		return
	}
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		http.Error(w, "invalid workspace ID", http.StatusBadRequest)
		return
	}

	userID, ok := ctx.Value(domain.ContextKeyUserID).(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req PresignUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	sceneID, err := uuid.Parse(req.SceneID)
	if err != nil {
		http.Error(w, "invalid scene ID", http.StatusBadRequest)
		return
	}

	if len(req.Assets) == 0 || len(req.Assets) > 100 {
		http.Error(w, "assets length must be between 1 and 100", http.StatusBadRequest)
		return
	}

	if !h.authorize(w, r, workspaceID, sceneID, true) {
		return
	}
	resp, err := h.assetUsecase.PresignUpload(ctx, workspaceID, userID, sceneID, req.Assets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"urls": resp})
}

func (h *AssetHandler) PresignView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workspaceIDStr := r.Header.Get("X-Workspace-ID")
	if workspaceIDStr == "" {
		http.Error(w, "missing X-Workspace-ID", http.StatusBadRequest)
		return
	}
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		http.Error(w, "invalid workspace ID", http.StatusBadRequest)
		return
	}

	var req PresignViewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	sceneID, err := uuid.Parse(req.SceneID)
	if err != nil {
		http.Error(w, "invalid scene ID", http.StatusBadRequest)
		return
	}

	if len(req.FileIDs) == 0 || len(req.FileIDs) > 100 {
		http.Error(w, "fileIDs length must be between 1 and 100", http.StatusBadRequest)
		return
	}

	if !h.authorize(w, r, workspaceID, sceneID, false) {
		return
	}
	resp, err := h.assetUsecase.PresignView(ctx, workspaceID, sceneID, req.FileIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"urls": resp})
}

func (h *AssetHandler) SetAccessService(access domain.AccessService) { h.access = access }
func (h *AssetHandler) authorize(w http.ResponseWriter, r *http.Request, workspace, scene uuid.UUID, edit bool) bool {
	if h.access == nil {
		return true
	}
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	d, e := h.access.Resolve(r.Context(), domain.UserPrincipal(user), domain.SceneRef{WorkspaceID: workspace, SceneID: scene})
	if e == nil && edit && d.Access != domain.AccessEdit {
		e = domain.ErrAccessDenied
	}
	if e != nil {
		middleware.WriteAccessError(w, r, e)
		return false
	}
	return true
}
