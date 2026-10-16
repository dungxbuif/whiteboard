package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"io"
	"net/http"
)

const shareRequestMaxBytes int64 = 4096

func writeSharingError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSONResponse(w, status, map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}
func sharingError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrSharePassword):
		writeSharingError(w, r, http.StatusUnauthorized, "SHARE_PASSWORD_REQUIRED", "Share password is required or invalid")
	case errors.Is(err, domain.ErrInvalidShareInput):
		writeSharingError(w, r, http.StatusBadRequest, "INVALID_INPUT", "Invalid sharing request")
	case errors.Is(err, domain.ErrInvalidSceneSession):
		writeSharingError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Invalid or expired scene session")
	default:
		middleware.WriteAccessError(w, r, err)
	}
}
func decodeSharing(w http.ResponseWriter, r *http.Request, input any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, shareRequestMaxBytes))
	decoder.DisallowUnknownFields()
	e := decoder.Decode(input)
	if e == nil {
		var extra any
		e = decoder.Decode(&extra)
		if e == io.EOF {
			e = nil
		} else if e == nil {
			e = domain.ErrInvalidShareInput
		}
	}
	var tooLarge *http.MaxBytesError
	if errors.As(e, &tooLarge) {
		writeSharingError(w, r, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body exceeds limit")
		return false
	}
	if e != nil || validator.New().Struct(input) != nil {
		sharingError(w, r, domain.ErrInvalidShareInput)
		return false
	}
	return true
}

type ShareLinkHandler struct{ links domain.ShareLinkService }

func NewShareLinkHandler(links domain.ShareLinkService) *ShareLinkHandler {
	return &ShareLinkHandler{links: links}
}
func shareRef(r *http.Request) (domain.SceneRef, error) {
	w, ok := middleware.WorkspaceIDFromContext(r.Context())
	if !ok {
		return domain.SceneRef{}, domain.ErrInaccessible
	}
	id, ok := r.Context().Value(domain.ContextKeySceneID).(uuid.UUID)
	if !ok {
		var e error
		id, e = uuid.Parse(chi.URLParam(r, "id"))
		if e != nil {
			return domain.SceneRef{}, domain.ErrInvalidShareInput
		}
	}
	return domain.SceneRef{WorkspaceID: w, SceneID: id}, nil
}
func (h *ShareLinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateShareLink
	if !decodeSharing(w, r, &input) {
		return
	}
	ref, e := shareRef(r)
	if e != nil {
		sharingError(w, r, e)
		return
	}
	actor, _ := middleware.UserIDFromContext(r.Context())
	result, e := h.links.Create(r.Context(), actor, ref, input)
	if e != nil {
		sharingError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusCreated, result)
}
func (h *ShareLinkHandler) List(w http.ResponseWriter, r *http.Request) {
	ref, e := shareRef(r)
	if e != nil {
		sharingError(w, r, e)
		return
	}
	actor, _ := middleware.UserIDFromContext(r.Context())
	result, e := h.links.List(r.Context(), actor, ref)
	if e != nil {
		sharingError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, result)
}
func (h *ShareLinkHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(chi.URLParam(r, "linkId"))
	if e != nil {
		sharingError(w, r, domain.ErrInvalidShareInput)
		return
	}
	ref, e := shareRef(r)
	if e != nil {
		sharingError(w, r, e)
		return
	}
	actor, _ := middleware.UserIDFromContext(r.Context())
	if e = h.links.Revoke(r.Context(), actor, ref, id); e != nil {
		sharingError(w, r, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
