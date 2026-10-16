package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"net/http"
)

type AccessHandler struct {
	access   domain.AccessService
	validate *validator.Validate
}

func NewAccessHandler(access domain.AccessService) *AccessHandler {
	return &AccessHandler{access: access, validate: validator.New()}
}

type memberRoleRequest struct {
	Role domain.WorkspaceRole `json:"role" validate:"required,oneof=editor viewer"`
}

func (h *AccessHandler) Members(w http.ResponseWriter, r *http.Request) {
	actor, _ := middleware.UserIDFromContext(r.Context())
	workspace, _ := middleware.WorkspaceIDFromContext(r.Context())
	members, e := h.access.ListMembers(r.Context(), actor, workspace)
	if e != nil {
		middleware.WriteAccessError(w, r, e)
		return
	}
	writeJSONResponse(w, http.StatusOK, members)
}
func (h *AccessHandler) mutate(w http.ResponseWriter, r *http.Request, scene bool) {
	actor, _ := middleware.UserIDFromContext(r.Context())
	workspace, _ := middleware.WorkspaceIDFromContext(r.Context())
	user, e := uuid.Parse(chi.URLParam(r, "userId"))
	if e != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	var role *domain.WorkspaceRole
	if r.Method != http.MethodDelete {
		var request memberRoleRequest
		if e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); e != nil || h.validate.Struct(request) != nil {
			http.Error(w, "Invalid role", http.StatusBadRequest)
			return
		}
		role = &request.Role
	}
	if scene {
		id, ok := parseURLUUID(w, r)
		if !ok {
			return
		}
		e = h.access.SetOverride(r.Context(), actor, domain.SceneRef{WorkspaceID: workspace, SceneID: id}, user, role)
		if e != nil {
			middleware.WriteAccessError(w, r, e)
			return
		}
	} else {
		e = h.access.SetMember(r.Context(), actor, workspace, user, role)
		if e != nil {
			middleware.WriteAccessError(w, r, e)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *AccessHandler) Member(w http.ResponseWriter, r *http.Request)   { h.mutate(w, r, false) }
func (h *AccessHandler) Override(w http.ResponseWriter, r *http.Request) { h.mutate(w, r, true) }
