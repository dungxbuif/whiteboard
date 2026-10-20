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

const domainRequestMaxBytes int64 = 8 * 1024

type domainHandlerAccess interface {
	ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error)
}
type DomainHandler struct {
	domains domain.DomainUsecase
	access  domainHandlerAccess
}

func NewDomainHandler(domains domain.DomainUsecase, access domainHandlerAccess) *DomainHandler {
	return &DomainHandler{domains: domains, access: access}
}

func (h *DomainHandler) Register(router chi.Router, auth func(http.Handler) http.Handler) {
	router.Get("/identity/domains/discover", h.Discover)
	router.Group(func(router chi.Router) {
		router.Use(auth)
		router.Get("/workspaces/{id}/domains", h.List)
		router.Post("/workspaces/{id}/domains", h.Begin)
		router.Post("/workspaces/{id}/domains/{domainId}/verify", h.Verify)
		router.Post("/workspaces/{id}/domains/{domainId}/capture", h.Capture)
		router.Post("/workspaces/{id}/domains/{domainId}/revoke", h.Revoke)
	})
}

func (h *DomainHandler) Discover(w http.ResponseWriter, r *http.Request) {
	value, err := h.domains.Discover(r.Context(), r.URL.Query().Get("domain"))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, value)
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	code, message, status := "INVALID_DOMAIN", "Invalid domain request", http.StatusBadRequest
	switch {
	case errors.Is(err, domain.ErrInvalidDomain):
	case errors.Is(err, domain.ErrDomainConflict):
		code, message, status = "DOMAIN_CONFLICT", "This exact domain is already claimed", http.StatusConflict
	case errors.Is(err, domain.ErrRequestIDReused):
		code, message, status = "REQUEST_ID_REUSED", "Request ID was already used", http.StatusConflict
	case errors.Is(err, domain.ErrDomainChallengeExpired):
		code, message, status = "CHALLENGE_EXPIRED", "The verification challenge has expired or was consumed", http.StatusConflict
	case errors.Is(err, domain.ErrDomainChallengeFailed):
		code, message, status = "DNS_VERIFICATION_FAILED", "The exact verification TXT record was not found", http.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrDomainNotVerified):
		code, message, status = "DOMAIN_NOT_VERIFIED", "Verify the domain before capture", http.StatusConflict
	case errors.Is(err, domain.ErrRevisionConflict):
		code, message, status = "REVISION_CONFLICT", "Domain state changed; reload and retry", http.StatusConflict
	default:
		middleware.WriteAccessError(w, r, err)
		return
	}
	writeJSONResponse(w, status, map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}

func decodeDomainRequest(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, domainRequestMaxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return domain.ErrInvalidDomain
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return domain.ErrInvalidDomain
	}
	return nil
}
func (h *DomainHandler) identity(r *http.Request) (domain.Principal, uuid.UUID, error) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return domain.Principal{}, uuid.Nil, domain.ErrAccessDenied
	}
	workspaceID, err := h.access.ResolveWorkspaceID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		return domain.Principal{}, uuid.Nil, err
	}
	principal := domain.UserPrincipal(userID)
	principal.CredentialExpiresAt, _ = r.Context().Value(domain.ContextKeyCredentialExpiry).(time.Time)
	return principal, workspaceID, nil
}
func domainID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "domainId"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidDomain
	}
	return id, nil
}
func parseDomainRequestID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidDomain
	}
	return id, nil
}

func (h *DomainHandler) List(w http.ResponseWriter, r *http.Request) {
	p, wid, err := h.identity(r)
	if err == nil {
		var values []domain.WorkspaceDomain
		values, err = h.domains.List(r.Context(), p, wid)
		if err == nil {
			w.Header().Set("Cache-Control", "no-store")
			writeJSONResponse(w, http.StatusOK, map[string]any{"domains": values})
			return
		}
	}
	writeDomainError(w, r, err)
}
func (h *DomainHandler) Begin(w http.ResponseWriter, r *http.Request) {
	p, wid, err := h.identity(r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	var body struct {
		Domain    string `json:"domain"`
		RequestID string `json:"requestId"`
	}
	err = decodeDomainRequest(w, r, &body)
	requestID, parseErr := parseDomainRequestID(body.RequestID)
	if err != nil || parseErr != nil {
		writeDomainError(w, r, domain.ErrInvalidDomain)
		return
	}
	value, err := h.domains.Begin(r.Context(), p, wid, body.Domain, requestID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusCreated, value)
}
func (h *DomainHandler) mutation(w http.ResponseWriter, r *http.Request, action string) {
	p, wid, err := h.identity(r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	id, parseErr := domainID(r)
	var body struct {
		RequestID string `json:"requestId"`
	}
	if parseErr == nil {
		err = decodeDomainRequest(w, r, &body)
	}
	requestID, idErr := parseDomainRequestID(body.RequestID)
	if err != nil || parseErr != nil || idErr != nil {
		writeDomainError(w, r, domain.ErrInvalidDomain)
		return
	}
	var response any
	switch action {
	case "verify":
		response, err = h.domains.Verify(r.Context(), p, wid, id, requestID)
	case "capture":
		response, err = h.domains.Capture(r.Context(), p, wid, id, requestID)
	case "revoke":
		response, err = h.domains.Revoke(r.Context(), p, wid, id, requestID)
	}
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, response)
}
func (h *DomainHandler) Verify(w http.ResponseWriter, r *http.Request)  { h.mutation(w, r, "verify") }
func (h *DomainHandler) Capture(w http.ResponseWriter, r *http.Request) { h.mutation(w, r, "capture") }
func (h *DomainHandler) Revoke(w http.ResponseWriter, r *http.Request)  { h.mutation(w, r, "revoke") }
