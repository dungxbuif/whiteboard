package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

const ssoRequestMaxBytes int64 = 16 * 1024
const ssoBrowserCookie = "devtool_sso_browser"

type SSOHandler struct {
	service domain.SSOUsecase
	access  interface {
		ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error)
	}
	frontendURL string
}

func NewSSOHandler(service domain.SSOUsecase, access interface {
	ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error)
}, frontendURL string) *SSOHandler {
	return &SSOHandler{service: service, access: access, frontendURL: strings.TrimRight(frontendURL, "/")}
}
func (h *SSOHandler) Register(router chi.Router, auth func(http.Handler) http.Handler) {
	router.Post("/sso/begin", h.Begin)
	router.Get("/sso/callback", h.Callback)
	router.Post("/sso/exchange", h.Exchange)
	router.Group(func(r chi.Router) {
		r.Use(auth)
		r.Post("/sso/link/begin", h.BeginLink)
		r.Get("/workspaces/{id}/sso", h.Get)
		r.Put("/workspaces/{id}/sso", h.Configure)
	})
}

func (h *SSOHandler) BeginLink(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.UserPrincipalFromContext(r.Context())
	if !ok {
		writeSSOError(w, r, domain.ErrAccessDenied)
		return
	}
	var body struct {
		ConnectionID string `json:"connectionId"`
		ReturnPath   string `json:"returnPath"`
		RequestID    string `json:"requestId"`
	}
	if err := decodeSSO(w, r, &body); err != nil {
		writeSSOError(w, r, err)
		return
	}
	connectionID, connectionErr := uuid.Parse(body.ConnectionID)
	requestID, requestErr := uuid.Parse(body.RequestID)
	if connectionErr != nil || requestErr != nil || connectionID == uuid.Nil || requestID == uuid.Nil {
		writeSSOError(w, r, domain.ErrSSOInvalid)
		return
	}
	binding, err := browserBinding(w, r, true)
	if err != nil {
		writeSSOError(w, r, err)
		return
	}
	value, err := h.service.BeginLink(withBrowser(r.Context(), binding), principal, connectionID, body.ReturnPath, requestID)
	if err != nil {
		writeSSOError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, value)
}

func decodeSSO(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, ssoRequestMaxBytes))
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(value); e != nil {
		return domain.ErrSSOInvalid
	}
	var extra any
	if e := decoder.Decode(&extra); e != io.EOF {
		return domain.ErrSSOInvalid
	}
	return nil
}
func writeSSOError(w http.ResponseWriter, r *http.Request, e error) {
	status, code, message := http.StatusBadRequest, "SSO_INVALID", "Invalid SSO request"
	switch {
	case errors.Is(e, domain.ErrSSOReplay):
		status, code, message = http.StatusConflict, "SSO_REPLAY", "SSO request was already used"
	case errors.Is(e, domain.ErrSSOExpired):
		status, code, message = http.StatusConflict, "SSO_EXPIRED", "SSO request expired"
	case errors.Is(e, domain.ErrSSONotConfigured):
		status, code, message = http.StatusNotFound, "SSO_NOT_CONFIGURED", "SSO is not configured"
	case errors.Is(e, domain.ErrSSOEmailLinkDenied):
		status, code, message = http.StatusConflict, "SSO_LINK_REQUIRED", "Sign in with your existing account before linking this identity"
	case errors.Is(e, domain.ErrRequestIDReused):
		status, code, message = http.StatusConflict, "REQUEST_ID_REUSED", "Request ID was already used"
	case errors.Is(e, domain.ErrRevisionConflict):
		status, code, message = http.StatusConflict, "REVISION_CONFLICT", "SSO settings changed; reload and retry"
	case errors.Is(e, domain.ErrRecentAuthRequired):
		status, code, message = http.StatusForbidden, "RECENT_AUTH_REQUIRED", "Reauthenticate before enforcing SSO"
	case errors.Is(e, domain.ErrBreakGlassRequired):
		status, code, message = http.StatusConflict, "BREAK_GLASS_REQUIRED", "Test operator recovery before enforcing SSO"
	case errors.Is(e, domain.ErrSessionRevoked):
		status, code, message = http.StatusUnauthorized, "SESSION_REVOKED", "Sign in again"
	case errors.Is(e, domain.ErrWorkspaceAssurance):
		status, code, message = http.StatusForbidden, "WORKSPACE_SSO_REQUIRED", "Sign in with this workspace's SSO connection"
	case errors.Is(e, domain.ErrAccessDenied):
		middleware.WriteAccessError(w, r, e)
		return
	}
	writeJSONResponse(w, status, map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}

func browserBinding(w http.ResponseWriter, r *http.Request, create bool) (string, error) {
	if cookie, e := r.Cookie(ssoBrowserCookie); e == nil && len(cookie.Value) >= 32 && len(cookie.Value) <= 256 {
		return cookie.Value, nil
	}
	if !create {
		return "", domain.ErrSSOInvalid
	}
	var b [32]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	value := base64.RawURLEncoding.EncodeToString(b[:])
	http.SetCookie(w, &http.Cookie{Name: ssoBrowserCookie, Value: value, Path: "/api/v1/sso", MaxAge: 600, HttpOnly: true, Secure: requestIsSecure(r), SameSite: http.SameSiteLaxMode})
	return value, nil
}
func withBrowser(ctx domain.Context, binding string) domain.Context {
	return context.WithValue(ctx, domain.ContextKeyBrowserBinding, binding)
}

func (h *SSOHandler) Begin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ConnectionID string `json:"connectionId"`
		ReturnPath   string `json:"returnPath"`
	}
	if e := decodeSSO(w, r, &body); e != nil {
		writeSSOError(w, r, e)
		return
	}
	id, e := uuid.Parse(body.ConnectionID)
	if e != nil || id == uuid.Nil {
		writeSSOError(w, r, domain.ErrSSOInvalid)
		return
	}
	binding, e := browserBinding(w, r, true)
	if e != nil {
		writeSSOError(w, r, e)
		return
	}
	value, e := h.service.Begin(withBrowser(r.Context(), binding), id, body.ReturnPath)
	if e != nil {
		writeSSOError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, value)
}
func (h *SSOHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		writeSSOError(w, r, domain.ErrSSOInvalid)
		return
	}
	binding, e := browserBinding(w, r, false)
	if e != nil {
		writeSSOError(w, r, e)
		return
	}
	value, e := h.service.Callback(r.Context(), r.URL.Query().Get("state"), r.URL.Query().Get("code"), binding)
	if e != nil {
		writeSSOError(w, r, e)
		return
	}
	target, parseErr := url.Parse(h.frontendURL + "/auth/sso/exchange")
	if parseErr != nil {
		writeSSOError(w, r, domain.ErrSSOInvalid)
		return
	}
	query := target.Query()
	query.Set("code", value.Code)
	query.Set("return", value.ReturnPath)
	target.RawQuery = query.Encode()
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, target.String(), http.StatusFound)
}
func (h *SSOHandler) Exchange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if e := decodeSSO(w, r, &body); e != nil {
		writeSSOError(w, r, e)
		return
	}
	binding, e := browserBinding(w, r, false)
	if e != nil {
		writeSSOError(w, r, e)
		return
	}
	value, e := h.service.Exchange(r.Context(), body.Code, binding)
	if e != nil {
		writeSSOError(w, r, e)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: ssoBrowserCookie, Value: "", Path: "/api/v1/sso", MaxAge: -1, HttpOnly: true, Secure: requestIsSecure(r), SameSite: http.SameSiteLaxMode})
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, value)
}

func (h *SSOHandler) identity(r *http.Request) (domain.Principal, uuid.UUID, error) {
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return domain.Principal{}, uuid.Nil, domain.ErrAccessDenied
	}
	workspace, e := h.access.ResolveWorkspaceID(r.Context(), chi.URLParam(r, "id"))
	if e != nil {
		return domain.Principal{}, uuid.Nil, e
	}
	p := domain.UserPrincipal(user)
	if sid, ok := r.Context().Value(domain.ContextKeySessionID).(uuid.UUID); ok {
		p.SessionID = sid
	}
	p.AuthMethod, _ = r.Context().Value(domain.ContextKeyAuthMethod).(string)
	p.AuthenticatedAt, _ = r.Context().Value(domain.ContextKeyAuthTime).(time.Time)
	return p, workspace, nil
}
func (h *SSOHandler) Get(w http.ResponseWriter, r *http.Request) {
	p, workspace, e := h.identity(r)
	if e == nil {
		var value domain.SSOConnection
		value, e = h.service.GetConnection(r.Context(), p, workspace)
		if e == nil {
			w.Header().Set("Cache-Control", "no-store")
			writeJSONResponse(w, http.StatusOK, value)
			return
		}
	}
	writeSSOError(w, r, e)
}
func (h *SSOHandler) Configure(w http.ResponseWriter, r *http.Request) {
	p, workspace, e := h.identity(r)
	if e != nil {
		writeSSOError(w, r, e)
		return
	}
	var body struct {
		Issuer             string                   `json:"issuer"`
		ClientID           string                   `json:"clientId"`
		ClientSecret       string                   `json:"clientSecret"`
		Mode               domain.SSOConnectionMode `json:"mode"`
		PublicLinksEnabled *bool                    `json:"publicLinksEnabled"`
		RequestID          string                   `json:"requestId"`
		ExpectedRevision   int64                    `json:"expectedRevision"`
	}
	if e = decodeSSO(w, r, &body); e != nil {
		writeSSOError(w, r, e)
		return
	}
	requestID, parseErr := uuid.Parse(body.RequestID)
	if parseErr != nil || requestID == uuid.Nil {
		writeSSOError(w, r, domain.ErrSSOInvalid)
		return
	}
	value, e := h.service.Configure(r.Context(), p, workspace, domain.SSOConnectionInput{Issuer: body.Issuer, ClientID: body.ClientID, ClientSecret: body.ClientSecret, Mode: body.Mode, PublicLinksEnabled: body.PublicLinksEnabled, RequestID: requestID, ExpectedRevision: body.ExpectedRevision})
	if e != nil {
		writeSSOError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, value)
}
