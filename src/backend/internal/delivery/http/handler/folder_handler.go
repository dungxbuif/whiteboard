package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type FolderHandler struct {
	usecase  domain.FolderUsecase
	validate *validator.Validate
}

type UpdateFolderRequest struct {
	Name        *string         `json:"name"`
	NewParentID json.RawMessage `json:"newParentId"`
}

func (h *FolderHandler) Update(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Invalid folder ID", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	var req UpdateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Invalid request body", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	var parent **uuid.UUID
	if req.NewParentID != nil {
		var parsed *uuid.UUID
		if !bytes.Equal(bytes.TrimSpace(req.NewParentID), []byte("null")) {
			var id uuid.UUID
			if err := json.Unmarshal(req.NewParentID, &id); err != nil || id == uuid.Nil {
				writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Invalid newParentId", "requestId": chimiddleware.GetReqID(r.Context())})
				return
			}
			parsed = &id
		}
		parent = &parsed
	}
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if err := h.validate.Struct(struct {
			Name string `validate:"required,min=1,max=100"`
		}{trimmed}); err != nil {
			writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Folder name must be 1–100 characters", "requestId": chimiddleware.GetReqID(r.Context())})
			return
		}
		req.Name = &trimmed
	}
	if err := h.usecase.Update(r.Context(), workspaceID, folderID, req.Name, parent); err != nil {
		if errors.Is(err, domain.ErrInvalidFolderUpdate) {
			writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": err.Error(), "requestId": chimiddleware.GetReqID(r.Context())})
			return
		}
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": "Unable to update folder", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func NewFolderHandler(usecase domain.FolderUsecase) *FolderHandler {
	return &FolderHandler{
		usecase:  usecase,
		validate: validator.New(),
	}
}

type CreateFolderRequest struct {
	Name     string     `json:"name" validate:"required,min=1,max=100"`
	ParentID *uuid.UUID `json:"parentId"`
}

type MoveFolderRequest struct {
	NewParentID *uuid.UUID `json:"newParentId"`
}

func (h *FolderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())

	var req CreateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	folder, err := h.usecase.Create(r.Context(), workspaceID, userID, req.Name, req.ParentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(folder)
}

func (h *FolderHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	if r.URL.Query().Get("all") == "true" {
		folders, err := h.usecase.ListAll(r.Context(), workspaceID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(folders)
		return
	}

	parentIDStr := r.URL.Query().Get("parentId")

	var parentID *uuid.UUID
	if parentIDStr != "" {
		id, err := uuid.Parse(parentIDStr)
		if err != nil {
			http.Error(w, "Invalid parentId", http.StatusBadRequest)
			return
		}
		parentID = &id
	}

	folders, err := h.usecase.List(r.Context(), workspaceID, parentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(folders)
}

func (h *FolderHandler) Move(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	var req MoveFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.usecase.Move(r.Context(), workspaceID, folderID, req.NewParentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *FolderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	if err := h.usecase.Delete(r.Context(), workspaceID, folderID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *FolderHandler) Restore(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	if err := h.usecase.Restore(r.Context(), workspaceID, folderID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *FolderHandler) ToggleStar(w http.ResponseWriter, r *http.Request) {
	workspaceID, _ := middleware.WorkspaceIDFromContext(r.Context())
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	if err := h.usecase.ToggleStar(r.Context(), workspaceID, folderID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
