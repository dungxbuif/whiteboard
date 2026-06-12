package handler

import (
	"bytes"
	"encoding/json"
	"net/http"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type SceneHandler struct {
	usecase  domain.SceneUsecase
	validate *validator.Validate
}

type CreateSceneRequest struct {
	Title    string          `json:"title" validate:"required,min=1,max=200"`
	FolderID *uuid.UUID      `json:"folderId"`
	Content  json.RawMessage `json:"content" validate:"required"`
}

type UpdateSceneRequest struct {
	Title    *string         `json:"title" validate:"omitempty,min=1,max=200"`
	FolderID json.RawMessage `json:"folderId"`
	Content  json.RawMessage `json:"content"`
}

type SetSceneStarRequest struct {
	IsStarred *bool `json:"isStarred" validate:"required"`
}

func NewSceneHandler(usecase domain.SceneUsecase) *SceneHandler {
	return &SceneHandler{usecase: usecase, validate: validator.New()}
}

func (h *SceneHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request CreateSceneRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.validate.Struct(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	userID, _ := middleware.UserIDFromContext(r.Context())
	scene, err := h.usecase.Create(r.Context(), workspaceID, userID, request.Title, request.FolderID, request.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSONResponse(w, http.StatusCreated, scene)
}

func (h *SceneHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	folderID, err := parseOptionalUUID(r.URL.Query().Get("folderId"))
	if err != nil {
		http.Error(w, "Invalid folderId", http.StatusBadRequest)
		return
	}
	scenes, err := h.usecase.List(r.Context(), workspaceID, folderID, r.URL.Query().Get("trash") == "true")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusOK, scenes)
}

func (h *SceneHandler) Get(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	sceneID, ok := parseURLUUID(w, r)
	if !ok {
		return
	}
	scene, err := h.usecase.Get(r.Context(), workspaceID, sceneID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSONResponse(w, http.StatusOK, scene)
}

func (h *SceneHandler) Update(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	userID, _ := middleware.UserIDFromContext(r.Context())
	sceneID, ok := parseURLUUID(w, r)
	if !ok {
		return
	}
	var request UpdateSceneRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.validate.Struct(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	folderID, err := parseFolderUpdate(request.FolderID)
	if err != nil {
		http.Error(w, "Invalid folderId", http.StatusBadRequest)
		return
	}
	var content *json.RawMessage
	if request.Content != nil {
		content = &request.Content
	}
	scene, err := h.usecase.Update(r.Context(), workspaceID, userID, sceneID, request.Title, folderID, content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSONResponse(w, http.StatusOK, scene)
}

func (h *SceneHandler) SetStar(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	sceneID, ok := parseURLUUID(w, r)
	if !ok {
		return
	}
	var request SetSceneStarRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || h.validate.Struct(request) != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.usecase.SetStar(r.Context(), workspaceID, sceneID, *request.IsStarred); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SceneHandler) Delete(w http.ResponseWriter, r *http.Request) {
	h.lifecycleAction(w, r, h.usecase.Delete)
}

func (h *SceneHandler) Restore(w http.ResponseWriter, r *http.Request) {
	h.lifecycleAction(w, r, h.usecase.Restore)
}

func (h *SceneHandler) lifecycleAction(w http.ResponseWriter, r *http.Request, action func(domain.Context, uuid.UUID, uuid.UUID) error) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	sceneID, ok := parseURLUUID(w, r)
	if !ok {
		return
	}
	if err := action(r.Context(), workspaceID, sceneID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseURLUUID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid scene ID", http.StatusBadRequest)
		return uuid.Nil, false
	}
	return id, true
}

func parseOptionalUUID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parseFolderUpdate(raw json.RawMessage) (**uuid.UUID, error) {
	if raw == nil {
		return nil, nil
	}
	var folderID *uuid.UUID
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return &folderID, nil
	}
	var id uuid.UUID
	if err := json.Unmarshal(raw, &id); err != nil {
		return nil, err
	}
	folderID = &id
	return &folderID, nil
}

func writeJSONResponse(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
