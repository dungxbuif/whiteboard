package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

const versionRequestMaxBytes int64 = 32768

type versionAccess interface {
	ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error)
	ResolveSceneID(domain.Context, uuid.UUID, string) (uuid.UUID, error)
}
type VersionHandler struct {
	versions domain.SceneVersionUsecase
	access   versionAccess
}

func NewVersionHandler(versions domain.SceneVersionUsecase, access versionAccess) *VersionHandler {
	return &VersionHandler{versions: versions, access: access}
}
func (h *VersionHandler) Routes(auth func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	h.Register(r, auth)
	return r
}
func (h *VersionHandler) Register(r chi.Router, auth func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				next.ServeHTTP(w, request)
			})
		})
		r.Use(auth)
		r.Get("/scenes/{id}/projection", h.CurrentProjection)
	})
	r.Group(func(r chi.Router) {
		r.Use(auth)
		r.Post("/scenes/{id}/duplicate", h.Duplicate)
		r.Get("/scenes/{id}/versions", h.List)
		r.Post("/scenes/{id}/versions", h.Create)
		r.Get("/scenes/{id}/versions/{versionId}", h.Get)
		r.Get("/scenes/{id}/versions/{versionId}/diff", h.Diff)
		r.Post("/scenes/{id}/versions/{versionId}/restore", h.Restore)
		r.Patch("/scenes/{id}/versions/{versionId}", h.SetMetadata)
	})
}

func (h *VersionHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	principal, ref, err := h.identity(r)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	var body struct {
		Title    string     `json:"title"`
		FolderID *uuid.UUID `json:"folderId"`
	}
	if !decodeVersion(w, r, &body) {
		return
	}
	scene, err := h.versions.Duplicate(r.Context(), principal, ref, body.Title, body.FolderID)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, scene)
}

func (h *VersionHandler) CurrentProjection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	principal, ref, err := h.identity(r)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	projection, err := h.versions.CurrentProjection(r.Context(), principal, ref)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, projection)
}
func (h *VersionHandler) identity(r *http.Request) (domain.Principal, domain.SceneRef, error) {
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return domain.Principal{}, domain.SceneRef{}, domain.ErrAccessDenied
	}
	workspace, err := h.access.ResolveWorkspaceID(r.Context(), r.Header.Get(domain.HeaderWorkspaceID))
	if err != nil {
		return domain.Principal{}, domain.SceneRef{}, err
	}
	scene, err := h.access.ResolveSceneID(r.Context(), workspace, chi.URLParam(r, "id"))
	if err != nil {
		return domain.Principal{}, domain.SceneRef{}, err
	}
	principal := domain.UserPrincipal(user)
	principal.GrantID = "membership:" + workspace.String() + ":" + user.String()
	principal.CredentialExpiresAt, _ = r.Context().Value(domain.ContextKeyCredentialExpiry).(time.Time)
	return principal, domain.SceneRef{WorkspaceID: workspace, SceneID: scene}, nil
}
func decodeVersion(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, versionRequestMaxBytes))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(value)
	if err == nil {
		var extra any
		if e := decoder.Decode(&extra); e != io.EOF {
			err = domain.ErrInvalidVersionInput
		}
	}
	if err != nil {
		writeVersionError(w, r, domain.ErrInvalidVersionInput)
		return false
	}
	return true
}
func writeVersionError(w http.ResponseWriter, r *http.Request, err error) {
	code, message, status := "INVALID_INPUT", "Invalid version request", http.StatusBadRequest
	var remote *domain.AgentCollabError
	switch {
	case errors.As(err, &remote):
		code, status = remote.Code, remote.Status
		message = "Version capture failed"
	case errors.Is(err, domain.ErrRequestIDReused):
		code, message, status = "REQUEST_ID_REUSED", "Request ID was already used", http.StatusConflict
	case errors.Is(err, domain.ErrRevisionConflict):
		code, message, status = "REVISION_CONFLICT", "Version metadata changed; reload and retry", http.StatusConflict
	case errors.Is(err, domain.ErrFeatureDisabled):
		code, message, status = "FEATURE_DISABLED", "Version history is disabled", http.StatusForbidden
	case errors.Is(err, domain.ErrInvalidVersionInput):
	default:
		middleware.WriteAccessError(w, r, err)
		return
	}
	writeJSONResponse(w, status, map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}

type createVersionRequest struct {
	RequestID        string  `json:"requestId"`
	ExpectedSequence string  `json:"expectedSequence"`
	Label            *string `json:"label,omitempty"`
	Pinned           bool    `json:"pinned"`
}
type metadataVersionRequest struct {
	RequestID        string  `json:"requestId"`
	ExpectedRevision string  `json:"expectedRevision"`
	Label            *string `json:"label,omitempty"`
	Pinned           *bool   `json:"pinned,omitempty"`
}
type restoreVersionRequest struct {
	RequestID        string `json:"requestId"`
	ExpectedSequence string `json:"expectedSequence"`
}

func (h *VersionHandler) Create(w http.ResponseWriter, r *http.Request) {
	p, ref, err := h.identity(r)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	var body createVersionRequest
	if !decodeVersion(w, r, &body) {
		return
	}
	requestID, e := uuid.Parse(body.RequestID)
	if e != nil {
		writeVersionError(w, r, domain.ErrInvalidVersionInput)
		return
	}
	out, err := h.versions.Create(r.Context(), p, ref, domain.CreateVersionInput{RequestID: requestID, ExpectedSequence: body.ExpectedSequence, Label: body.Label, Pinned: body.Pinned})
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusCreated, out)
}
func (h *VersionHandler) List(w http.ResponseWriter, r *http.Request) {
	p, ref, err := h.identity(r)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
	}
	if err != nil {
		writeVersionError(w, r, domain.ErrInvalidVersionInput)
		return
	}
	out, err := h.versions.List(r.Context(), p, ref, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, out)
}
func (h *VersionHandler) Get(w http.ResponseWriter, r *http.Request) {
	p, ref, err := h.identity(r)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	id, e := uuid.Parse(chi.URLParam(r, "versionId"))
	if e != nil {
		writeVersionError(w, r, domain.ErrInvalidVersionInput)
		return
	}
	out, err := h.versions.Get(r.Context(), p, ref, id)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, out)
}
func (h *VersionHandler) Diff(w http.ResponseWriter, r *http.Request) {
	p, ref, err := h.identity(r)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	id, parseErr := uuid.Parse(chi.URLParam(r, "versionId"))
	against := r.URL.Query().Get("againstSequence")
	if parseErr != nil || !validHTTPSequence(against) {
		writeVersionError(w, r, domain.ErrInvalidVersionInput)
		return
	}
	out, err := h.versions.Diff(r.Context(), p, ref, id, against)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, out)
}
func validHTTPSequence(value string) bool {
	n, err := strconv.ParseUint(value, 10, 63)
	return err == nil && strconv.FormatUint(n, 10) == value
}
func (h *VersionHandler) Restore(w http.ResponseWriter, r *http.Request) {
	p, ref, err := h.identity(r)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	versionID, versionErr := uuid.Parse(chi.URLParam(r, "versionId"))
	var body restoreVersionRequest
	if versionErr != nil || !decodeVersion(w, r, &body) {
		if versionErr != nil {
			writeVersionError(w, r, domain.ErrInvalidVersionInput)
		}
		return
	}
	requestID, requestErr := uuid.Parse(body.RequestID)
	if requestErr != nil || !validHTTPSequence(body.ExpectedSequence) {
		writeVersionError(w, r, domain.ErrInvalidVersionInput)
		return
	}
	out, err := h.versions.Restore(r.Context(), p, ref, domain.RestoreVersionInput{VersionID: versionID, RequestID: requestID, ExpectedSequence: body.ExpectedSequence})
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, out)
}
func (h *VersionHandler) SetMetadata(w http.ResponseWriter, r *http.Request) {
	p, ref, err := h.identity(r)
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	id, e := uuid.Parse(chi.URLParam(r, "versionId"))
	var body metadataVersionRequest
	if e != nil || !decodeVersion(w, r, &body) {
		if e != nil {
			writeVersionError(w, r, domain.ErrInvalidVersionInput)
		}
		return
	}
	requestID, e := uuid.Parse(body.RequestID)
	revision, re := strconv.ParseInt(body.ExpectedRevision, 10, 64)
	if e != nil || re != nil || revision < 1 || strings.TrimSpace(body.ExpectedRevision) != body.ExpectedRevision {
		writeVersionError(w, r, domain.ErrInvalidVersionInput)
		return
	}
	out, err := h.versions.SetMetadata(r.Context(), p, ref, id, domain.VersionMetadataInput{RequestID: requestID, ExpectedRevision: revision, Label: body.Label, Pinned: body.Pinned})
	if err != nil {
		writeVersionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, out)
}
