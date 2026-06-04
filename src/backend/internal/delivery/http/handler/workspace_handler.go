package handler

import (
	"encoding/json"
	"net/http"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"

	"github.com/go-playground/validator/v10"
)

type WorkspaceHandler struct {
	usecase  domain.WorkspaceUsecase
	validate *validator.Validate
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
