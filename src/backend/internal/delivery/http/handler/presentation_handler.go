package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type presentationAccess interface {
	ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error)
	ResolveSceneID(domain.Context, uuid.UUID, string) (uuid.UUID, error)
}
type presentationAttempt struct {
	count int
	reset time.Time
}
type PresentationHandler struct {
	presentations domain.PresentationUsecase
	access        presentationAccess
	mu            sync.Mutex
	attempts      map[string]presentationAttempt
	now           func() time.Time
}

func NewPresentationHandler(p domain.PresentationUsecase, a presentationAccess) *PresentationHandler {
	return &PresentationHandler{presentations: p, access: a, attempts: map[string]presentationAttempt{}, now: time.Now}
}
func (h *PresentationHandler) Register(r chi.Router, auth func(http.Handler) http.Handler) {
	r.Post("/presentations/{token}/join", h.Join)
	r.Group(func(r chi.Router) {
		r.Use(auth)
		r.Post("/scenes/{id}/presentations", h.Create)
		r.Post("/scenes/{id}/presentations/{sessionId}/links", h.CreateLink)
		r.Delete("/scenes/{id}/presentations/{sessionId}/links/{linkId}", h.RevokeLink)
		r.Post("/scenes/{id}/presentations/{sessionId}/admit", h.Admit)
	})
}
func (h *PresentationHandler) Routes(auth func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	h.Register(r, auth)
	return r
}
func decodePresentation(w http.ResponseWriter, r *http.Request, value any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	d.DisallowUnknownFields()
	if e := d.Decode(value); e != nil {
		writePresentationError(w, r, domain.ErrInvalidPresentation)
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		writePresentationError(w, r, domain.ErrInvalidPresentation)
		return false
	}
	return true
}
func writePresentationError(w http.ResponseWriter, r *http.Request, e error) {
	code, message, status := "INVALID_INPUT", "Invalid presentation request", http.StatusBadRequest
	var remote *domain.AgentCollabError
	switch {
	case errors.As(e, &remote):
		code, message, status = remote.Code, "Presentation runtime rejected the request", remote.Status
	case errors.Is(e, domain.ErrFeatureDisabled):
		code, message, status = "FEATURE_DISABLED", "Presentation is disabled", http.StatusForbidden
	case errors.Is(e, domain.ErrPresentationEnded):
		code, message, status = "PRESENTATION_ENDED", "Presentation has ended", http.StatusConflict
	case errors.Is(e, domain.ErrPresentationFull):
		code, message, status = "WAITING_ROOM_FULL", "Waiting room is full", http.StatusConflict
	case errors.Is(e, domain.ErrRequestIDReused):
		code, message, status = "REQUEST_ID_REUSED", "Request ID was already used; create a new presentation link request", http.StatusConflict
	case errors.Is(e, domain.ErrInvalidPresentation):
	default:
		middleware.WriteAccessError(w, r, e)
		return
	}
	writeJSONResponse(w, status, map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}
func (h *PresentationHandler) identity(r *http.Request) (domain.Principal, domain.SceneRef, error) {
	u, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return domain.Principal{}, domain.SceneRef{}, domain.ErrAccessDenied
	}
	w, e := h.access.ResolveWorkspaceID(r.Context(), r.Header.Get(domain.HeaderWorkspaceID))
	if e != nil {
		return domain.Principal{}, domain.SceneRef{}, e
	}
	scene, e := h.access.ResolveSceneID(r.Context(), w, chi.URLParam(r, "id"))
	return domain.UserPrincipal(u), domain.SceneRef{WorkspaceID: w, SceneID: scene}, e
}
func requestUUID(raw string) (uuid.UUID, error) {
	id, e := uuid.Parse(raw)
	if e != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidPresentation
	}
	return id, nil
}
func (h *PresentationHandler) Create(w http.ResponseWriter, r *http.Request) {
	p, ref, e := h.identity(r)
	var in struct {
		RequestID string `json:"requestId"`
	}
	if e != nil || !decodePresentation(w, r, &in) {
		if e != nil {
			writePresentationError(w, r, e)
		}
		return
	}
	request, e := requestUUID(in.RequestID)
	if e != nil {
		writePresentationError(w, r, e)
		return
	}
	out, e := h.presentations.Create(r.Context(), p, ref, request)
	if e != nil {
		writePresentationError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusCreated, out)
}
func (h *PresentationHandler) CreateLink(w http.ResponseWriter, r *http.Request) {
	p, ref, e := h.identity(r)
	var in struct {
		RequestID string `json:"requestId"`
	}
	if e != nil || !decodePresentation(w, r, &in) {
		if e != nil {
			writePresentationError(w, r, e)
		}
		return
	}
	session, se := requestUUID(chi.URLParam(r, "sessionId"))
	request, re := requestUUID(in.RequestID)
	if se != nil || re != nil {
		writePresentationError(w, r, domain.ErrInvalidPresentation)
		return
	}
	out, e := h.presentations.CreateLink(r.Context(), p, ref, session, request)
	if e != nil {
		writePresentationError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusCreated, out)
}
func (h *PresentationHandler) allowJoin(r *http.Request) bool {
	host := middleware.TrustedClientIP(r)
	now := h.now()
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.attempts) >= 20_000 {
		for key, value := range h.attempts {
			if !now.Before(value.reset) {
				delete(h.attempts, key)
			}
		}
		if len(h.attempts) >= 20_000 {
			return false
		}
	}
	entry := h.attempts[host]
	if !now.Before(entry.reset) {
		entry = presentationAttempt{reset: now.Add(time.Minute)}
	}
	if entry.count >= 60 {
		return false
	}
	entry.count++
	h.attempts[host] = entry
	return true
}
func (h *PresentationHandler) Join(w http.ResponseWriter, r *http.Request) {
	if !h.allowJoin(r) {
		writeJSONResponse(w, http.StatusTooManyRequests, map[string]string{"code": "RATE_LIMITED", "message": "Try again later", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	var in struct {
		DisplayName string `json:"displayName"`
	}
	if !decodePresentation(w, r, &in) {
		return
	}
	out, e := h.presentations.Join(r.Context(), chi.URLParam(r, "token"), in.DisplayName)
	if e != nil {
		writePresentationError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, out)
}
func (h *PresentationHandler) Admit(w http.ResponseWriter, r *http.Request) {
	p, ref, e := h.identity(r)
	var in struct {
		ConnectionID     string `json:"connectionId"`
		ExpectedRevision string `json:"expectedRevision"`
	}
	if e != nil || !decodePresentation(w, r, &in) {
		if e != nil {
			writePresentationError(w, r, e)
		}
		return
	}
	session, se := requestUUID(chi.URLParam(r, "sessionId"))
	revision, re := strconv.ParseInt(in.ExpectedRevision, 10, 64)
	if se != nil || re != nil || revision < 1 || strconv.FormatInt(revision, 10) != in.ExpectedRevision {
		writePresentationError(w, r, domain.ErrInvalidPresentation)
		return
	}
	out, e := h.presentations.Admit(r.Context(), p, ref, session, in.ConnectionID, revision)
	if e != nil {
		writePresentationError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, out)
}
func (h *PresentationHandler) RevokeLink(w http.ResponseWriter, r *http.Request) {
	p, ref, e := h.identity(r)
	var in struct {
		RequestID string `json:"requestId"`
	}
	if e != nil || !decodePresentation(w, r, &in) {
		if e != nil {
			writePresentationError(w, r, e)
		}
		return
	}
	session, se := requestUUID(chi.URLParam(r, "sessionId"))
	link, le := requestUUID(chi.URLParam(r, "linkId"))
	request, re := requestUUID(in.RequestID)
	if se != nil || le != nil || re != nil {
		writePresentationError(w, r, domain.ErrInvalidPresentation)
		return
	}
	if e = h.presentations.RevokeLink(r.Context(), p, ref, session, link, request); e != nil {
		writePresentationError(w, r, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
