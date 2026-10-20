package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type activityAccess interface {
	ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error)
	ResolveSceneID(domain.Context, uuid.UUID, string) (uuid.UUID, error)
}

type ActivityHandler struct {
	activity domain.ActivityUsecase
	access   activityAccess
}

func NewActivityHandler(activity domain.ActivityUsecase, access activityAccess) *ActivityHandler {
	return &ActivityHandler{activity: activity, access: access}
}

func (h *ActivityHandler) Routes(auth func(http.Handler) http.Handler) http.Handler {
	router := chi.NewRouter()
	h.Register(router, auth)
	return router
}

func (h *ActivityHandler) Register(router chi.Router, auth func(http.Handler) http.Handler) {
	router.With(auth).Get("/workspaces/{id}/activity", h.Workspace)
	router.With(auth).Get("/scenes/{id}/activity", h.Scene)
}

func writeActivityError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, domain.ErrInvalidActivity) {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Invalid activity request", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	if errors.Is(err, domain.ErrFeatureDisabled) {
		writeJSONResponse(w, http.StatusForbidden, map[string]string{"code": "FEATURE_DISABLED", "message": "Activity is disabled", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	middleware.WriteAccessError(w, r, err)
}

func activityPrincipal(r *http.Request) (domain.Principal, error) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return domain.Principal{}, domain.ErrAccessDenied
	}
	principal := domain.UserPrincipal(userID)
	principal.CredentialExpiresAt, _ = r.Context().Value(domain.ContextKeyCredentialExpiry).(time.Time)
	return principal, nil
}

func activityLimit(r *http.Request) (int, error) {
	limit := 50
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
	}
	if err != nil || limit < 1 || limit > 50 {
		return 0, domain.ErrInvalidActivity
	}
	return limit, nil
}

func (h *ActivityHandler) list(w http.ResponseWriter, r *http.Request, workspaceID uuid.UUID, sceneID *uuid.UUID) {
	principal, err := activityPrincipal(r)
	if err != nil {
		writeActivityError(w, r, err)
		return
	}
	limit, err := activityLimit(r)
	if err != nil {
		writeActivityError(w, r, err)
		return
	}
	page, err := h.activity.List(r.Context(), principal, workspaceID, sceneID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeActivityError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, page)
}

func (h *ActivityHandler) Workspace(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := h.access.ResolveWorkspaceID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeActivityError(w, r, err)
		return
	}
	h.list(w, r, workspaceID, nil)
}

func (h *ActivityHandler) Scene(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := h.access.ResolveWorkspaceID(r.Context(), r.Header.Get(domain.HeaderWorkspaceID))
	if err != nil {
		writeActivityError(w, r, err)
		return
	}
	sceneID, err := h.access.ResolveSceneID(r.Context(), workspaceID, chi.URLParam(r, "id"))
	if err != nil {
		writeActivityError(w, r, err)
		return
	}
	h.list(w, r, workspaceID, &sceneID)
}
