package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

const retentionRequestMaxBytes int64 = 16 * 1024

type retentionHandlerAccess interface {
	ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error)
}
type RetentionHandler struct {
	retention domain.RetentionUsecase
	access    retentionHandlerAccess
}

func NewRetentionHandler(retention domain.RetentionUsecase, access retentionHandlerAccess) *RetentionHandler {
	return &RetentionHandler{retention: retention, access: access}
}
func (h *RetentionHandler) Register(router chi.Router, auth func(http.Handler) http.Handler) {
	router.Group(func(r chi.Router) {
		r.Use(auth)
		r.Get("/workspaces/{id}/retention/policy", h.Policy)
		r.Post("/workspaces/{id}/retention/previews", h.Preview)
		r.Post("/workspaces/{id}/retention/schedule", h.Schedule)
		r.Put("/workspaces/{id}/retention/holds", h.Hold)
	})
}

func decodeRetentionRequest(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, retentionRequestMaxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return domain.ErrInvalidRetention
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return domain.ErrInvalidRetention
	}
	return nil
}
func retentionRequestID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidRetention
	}
	return id, nil
}
func writeRetentionError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := http.StatusBadRequest, "INVALID_INPUT", "Invalid retention request"
	switch {
	case errors.Is(err, domain.ErrInvalidRetention):
	case errors.Is(err, domain.ErrRequestIDReused):
		status, code, message = http.StatusConflict, "REQUEST_ID_REUSED", "Request ID was already used"
	case errors.Is(err, domain.ErrRevisionConflict):
		status, code, message = http.StatusConflict, "REVISION_CONFLICT", "Retention policy changed; create a new preview"
	case errors.Is(err, domain.ErrPreviewExpired):
		status, code, message = http.StatusConflict, "PREVIEW_EXPIRED", "Retention preview expired; create a new preview"
	case errors.Is(err, domain.ErrFeatureDisabled):
		status, code, message = http.StatusForbidden, "FEATURE_DISABLED", "Retention is disabled"
	default:
		middleware.WriteAccessError(w, r, err)
		return
	}
	writeJSONResponse(w, status, map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}
func (h *RetentionHandler) identity(r *http.Request) (domain.Principal, uuid.UUID, error) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return domain.Principal{}, uuid.Nil, domain.ErrAccessDenied
	}
	workspaceID, err := h.access.ResolveWorkspaceID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		return domain.Principal{}, uuid.Nil, err
	}
	p := domain.UserPrincipal(userID)
	p.CredentialExpiresAt, _ = r.Context().Value(domain.ContextKeyCredentialExpiry).(time.Time)
	return p, workspaceID, nil
}

func (h *RetentionHandler) Policy(w http.ResponseWriter, r *http.Request) {
	p, wid, err := h.identity(r)
	if err == nil {
		var value domain.RetentionPolicy
		value, err = h.retention.Policy(r.Context(), p, wid)
		if err == nil {
			w.Header().Set("Cache-Control", "no-store")
			writeJSONResponse(w, http.StatusOK, value)
			return
		}
	}
	writeRetentionError(w, r, err)
}
func (h *RetentionHandler) Preview(w http.ResponseWriter, r *http.Request) {
	p, wid, err := h.identity(r)
	var body domain.RetentionPolicyInput
	if err == nil {
		err = decodeRetentionRequest(w, r, &body)
	}
	if err == nil {
		var value domain.RetentionPreview
		value, err = h.retention.PreviewPolicy(r.Context(), p, wid, body)
		if err == nil {
			w.Header().Set("Cache-Control", "no-store")
			writeJSONResponse(w, http.StatusCreated, value)
			return
		}
	}
	writeRetentionError(w, r, err)
}
func (h *RetentionHandler) Schedule(w http.ResponseWriter, r *http.Request) {
	p, wid, err := h.identity(r)
	var body struct {
		PreviewID        string `json:"previewId"`
		ExpectedRevision int64  `json:"expectedRevision,string"`
		RequestID        string `json:"requestId"`
	}
	if err == nil {
		err = decodeRetentionRequest(w, r, &body)
	}
	previewID, previewErr := uuid.Parse(body.PreviewID)
	requestID, requestErr := retentionRequestID(body.RequestID)
	if err != nil || previewErr != nil || previewID == uuid.Nil || requestErr != nil {
		writeRetentionError(w, r, domain.ErrInvalidRetention)
		return
	}
	value, err := h.retention.Schedule(r.Context(), p, wid, previewID, body.ExpectedRevision, requestID)
	if err != nil {
		writeRetentionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, value)
}
func (h *RetentionHandler) Hold(w http.ResponseWriter, r *http.Request) {
	p, wid, err := h.identity(r)
	var body struct {
		Kind      string  `json:"kind"`
		ID        string  `json:"id"`
		VersionID *string `json:"versionId,omitempty"`
		Held      bool    `json:"held"`
		RequestID string  `json:"requestId"`
	}
	if err == nil {
		err = decodeRetentionRequest(w, r, &body)
	}
	requestID, requestErr := retentionRequestID(body.RequestID)
	var versionID *uuid.UUID
	if body.VersionID != nil {
		id, parseErr := uuid.Parse(*body.VersionID)
		if parseErr != nil || id == uuid.Nil {
			err = domain.ErrInvalidRetention
		} else {
			versionID = &id
		}
	}
	if err != nil || requestErr != nil {
		writeRetentionError(w, r, domain.ErrInvalidRetention)
		return
	}
	value, err := h.retention.SetHold(r.Context(), p, domain.RetentionReference{WorkspaceID: wid, Kind: body.Kind, ID: body.ID}, versionID, body.Held, requestID)
	if err != nil {
		writeRetentionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, value)
}
