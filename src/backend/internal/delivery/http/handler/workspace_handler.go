package handler

import (
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

type WorkspaceHandler struct {
	usecase  domain.WorkspaceUsecase
	validate *validator.Validate
}

func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		middleware.WriteAccessError(w, r, domain.ErrAccessDenied)
		return
	}
	workspaceID, ok := middleware.WorkspaceIDFromContext(r.Context())
	if !ok {
		var err error
		workspaceID, err = uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Invalid workspace ID", "requestId": chimiddleware.GetReqID(r.Context())})
			return
		}
	}
	var req CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Invalid request body", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := h.validate.Struct(req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Workspace name must be 3–100 characters", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	workspace, err := h.usecase.UpdateName(r.Context(), userID, workspaceID, req.Name)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidWorkspaceName) {
			writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Workspace name must be 3–100 characters", "requestId": chimiddleware.GetReqID(r.Context())})
			return
		}
		if errors.Is(err, domain.ErrAccessDenied) || errors.Is(err, domain.ErrInaccessible) {
			middleware.WriteAccessError(w, r, err)
			return
		}
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": "Unable to rename workspace", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	writeJSONResponse(w, http.StatusOK, workspace)
}

func NewWorkspaceHandler(usecase domain.WorkspaceUsecase) *WorkspaceHandler {
	return &WorkspaceHandler{
		usecase:  usecase,
		validate: validator.New(),
	}
}

type CreateWorkspaceRequest struct {
	Name string `json:"name" validate:"required,min=3,max=100"`
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	workspace, err := h.usecase.Create(r.Context(), userID, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(workspace)
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	workspaces, err := h.usecase.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(workspaces)
}
