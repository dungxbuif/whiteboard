package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const invitationRequestMaxBytes int64 = 4096

type InvitationHandler struct {
	uc       domain.InvitationUsecase
	validate *validator.Validate
}

func NewInvitationHandler(uc domain.InvitationUsecase) *InvitationHandler {
	return &InvitationHandler{uc: uc, validate: validator.New()}
}

type createInvitationRequest struct {
	Email string               `json:"email" validate:"required"`
	Role  domain.WorkspaceRole `json:"role" validate:"required,oneof=editor viewer"`
}
type acceptInvitationRequest struct {
	Token string `json:"token" validate:"required,len=43"`
}
type registerInvitationRequest struct {
	Token string `json:"token" validate:"required,len=43"`
	domain.RegistrationInput
}

func writeInvitationError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}
func invitationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvitationExpired):
		writeInvitationError(w, r, http.StatusGone, "INVITATION_EXPIRED", "Invitation expired")
	case errors.Is(err, domain.ErrInvitationEmail):
		writeInvitationError(w, r, http.StatusForbidden, "INVITATION_EMAIL_MISMATCH", "Log in with the invited email address")
	case errors.Is(err, domain.ErrRegistrationRejected):
		writeInvitationError(w, r, http.StatusBadRequest, "REGISTRATION_REJECTED", domain.ErrRegistrationRejected.Error())
	case errors.Is(err, domain.ErrInvitationInvalid):
		writeInvitationError(w, r, http.StatusBadRequest, "INVITATION_INVALID", "Invitation is invalid or unavailable")
	case errors.Is(err, domain.ErrInvitationInput):
		writeInvitationError(w, r, http.StatusBadRequest, "INVALID_INPUT", "Invalid invitation request")
	default:
		middleware.WriteAccessError(w, r, err)
	}
}
func (h *InvitationHandler) decode(w http.ResponseWriter, r *http.Request, input interface{}) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, invitationRequestMaxBytes))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(input)
	if err == nil {
		var extra interface{}
		err = decoder.Decode(&extra)
		if err == io.EOF {
			err = nil
		} else if err == nil {
			err = domain.ErrInvitationInput
		}
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeInvitationError(w, r, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body exceeds limit")
		return false
	}
	if registration, ok := input.(*registerInvitationRequest); ok {
		registration.Email = strings.ToLower(strings.TrimSpace(registration.Email))
		registration.Name = strings.TrimSpace(registration.Name)
	}
	if err != nil || h.validate.Struct(input) != nil {
		invitationError(w, r, domain.ErrInvitationInput)
		return false
	}
	return true
}
func (h *InvitationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input createInvitationRequest
	if !h.decode(w, r, &input) {
		return
	}
	actor, _ := middleware.UserIDFromContext(r.Context())
	workspace, _ := middleware.WorkspaceIDFromContext(r.Context())
	result, err := h.uc.CreateInvitation(r.Context(), actor, workspace, input.Email, input.Role)
	if err != nil {
		invitationError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, result)
}
func (h *InvitationHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, _ := middleware.UserIDFromContext(r.Context())
	workspace, _ := middleware.WorkspaceIDFromContext(r.Context())
	result, err := h.uc.ListInvitations(r.Context(), actor, workspace)
	if err != nil {
		invitationError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}
func (h *InvitationHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "inviteId"))
	if err != nil {
		invitationError(w, r, domain.ErrInvitationInput)
		return
	}
	actor, _ := middleware.UserIDFromContext(r.Context())
	workspace, _ := middleware.WorkspaceIDFromContext(r.Context())
	if err = h.uc.RevokeInvitation(r.Context(), actor, workspace, id); err != nil {
		invitationError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *InvitationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeInvitationError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required")
		return
	}
	var input acceptInvitationRequest
	if !h.decode(w, r, &input) {
		return
	}
	result, err := h.uc.AcceptInvitation(r.Context(), actor, input.Token)
	if err != nil {
		invitationError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}
func (h *InvitationHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input registerInvitationRequest
	if !h.decode(w, r, &input) {
		return
	}
	result, err := h.uc.RegisterWithInvitation(r.Context(), input.Token, input.RegistrationInput)
	if err != nil {
		invitationError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, result)
}
