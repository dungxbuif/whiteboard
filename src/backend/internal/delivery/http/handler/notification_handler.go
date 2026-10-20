package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type NotificationHandler struct{ notifications domain.NotificationUsecase }

func NewNotificationHandler(notifications domain.NotificationUsecase) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

func (h *NotificationHandler) Routes(auth func(http.Handler) http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(auth)
	router.Get("/notifications", h.List)
	router.Post("/notifications/{id}/read", h.MarkRead)
	return router
}

func writeNotificationError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, domain.ErrInvalidNotification) {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Invalid notification request", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	if errors.Is(err, domain.ErrRequestIDReused) {
		writeJSONResponse(w, http.StatusConflict, map[string]string{"code": "REQUEST_ID_REUSED", "message": "Request ID was already used", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	middleware.WriteAccessError(w, r, err)
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeNotificationError(w, r, domain.ErrAccessDenied)
		return
	}
	limit := 50
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
	}
	if err != nil {
		writeNotificationError(w, r, domain.ErrInvalidNotification)
		return
	}
	page, err := h.notifications.List(r.Context(), userID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeNotificationError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, page)
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeNotificationError(w, r, domain.ErrAccessDenied)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeNotificationError(w, r, domain.ErrInvalidNotification)
		return
	}
	var body struct {
		RequestID string `json:"requestId"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&body); err == nil {
		var extra any
		if extraErr := decoder.Decode(&extra); extraErr != io.EOF {
			err = domain.ErrInvalidNotification
		}
	}
	requestID, parseErr := uuid.Parse(body.RequestID)
	if err != nil || parseErr != nil {
		writeNotificationError(w, r, domain.ErrInvalidNotification)
		return
	}
	if err = h.notifications.MarkRead(r.Context(), userID, id, requestID); err != nil {
		writeNotificationError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
