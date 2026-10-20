package handler

import (
	"context"
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
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const commentRequestMaxBytes int64 = 32768

type commentAccess interface {
	ResolveWorkspaceID(context.Context, string) (uuid.UUID, error)
	ResolveSceneID(context.Context, uuid.UUID, string) (uuid.UUID, error)
}

type CommentHandler struct {
	comments domain.CommentUsecase
	rooms    domain.RoomService
	access   commentAccess
	validate *validator.Validate
}

func NewCommentHandler(comments domain.CommentUsecase, rooms domain.RoomService, access commentAccess) *CommentHandler {
	return &CommentHandler{comments: comments, rooms: rooms, access: access, validate: validator.New()}
}

type commentIdentity struct {
	principal domain.Principal
	ref       domain.SceneRef
}

type commentIdentityKey struct{}

func (h *CommentHandler) identify(r *http.Request) (commentIdentity, bool) {
	identity, ok := r.Context().Value(commentIdentityKey{}).(commentIdentity)
	return identity, ok
}

func (h *CommentHandler) loginIdentity(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			middleware.WriteAccessError(w, r, domain.ErrAccessDenied)
			return
		}
		workspace, err := h.access.ResolveWorkspaceID(r.Context(), r.Header.Get(domain.HeaderWorkspaceID))
		if err != nil {
			middleware.WriteAccessError(w, r, err)
			return
		}
		scene, err := h.access.ResolveSceneID(r.Context(), workspace, chi.URLParam(r, "id"))
		if err != nil {
			middleware.WriteAccessError(w, r, err)
			return
		}
		value := commentIdentity{principal: domain.UserPrincipal(userID), ref: domain.SceneRef{WorkspaceID: workspace, SceneID: scene}}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), commentIdentityKey{}, value)))
	}
}

func (h *CommentHandler) principalHandler(auth func(http.Handler) http.Handler, next http.Handler) http.Handler {
	login := auth(h.loginIdentity(next))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get(domain.HeaderAuthorization)
		if strings.HasPrefix(raw, domain.BearerPrefix) {
			identity, err := h.rooms.ParseSceneSession(strings.TrimPrefix(raw, domain.BearerPrefix))
			if err == nil && identity.Principal.Kind == domain.PrincipalPublic && time.Now().Before(identity.ExpiresAt) {
				scene, resolveErr := h.access.ResolveSceneID(r.Context(), identity.SceneRef.WorkspaceID, chi.URLParam(r, "id"))
				if resolveErr != nil || scene != identity.SceneRef.SceneID {
					middleware.WriteAccessError(w, r, domain.ErrInaccessible)
					return
				}
				if header := r.Header.Get(domain.HeaderWorkspaceID); header != "" {
					workspace, headerErr := h.access.ResolveWorkspaceID(r.Context(), header)
					if headerErr != nil || workspace != identity.SceneRef.WorkspaceID {
						middleware.WriteAccessError(w, r, domain.ErrInaccessible)
						return
					}
				}
				value := commentIdentity{principal: identity.Principal, ref: identity.SceneRef}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), commentIdentityKey{}, value)))
				return
			}
		}
		login.ServeHTTP(w, r)
	})
}

func (h *CommentHandler) Routes(auth func(http.Handler) http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Method(http.MethodGet, "/scenes/{id}/comments", h.principalHandler(auth, http.HandlerFunc(h.List)))
	router.Method(http.MethodPost, "/scenes/{id}/comments", h.principalHandler(auth, http.HandlerFunc(h.Create)))
	router.Method(http.MethodGet, "/scenes/{id}/comments/{threadId}", h.principalHandler(auth, http.HandlerFunc(h.Get)))
	router.Method(http.MethodPost, "/scenes/{id}/comments/{threadId}/replies", h.principalHandler(auth, http.HandlerFunc(h.Reply)))
	router.Method(http.MethodPatch, "/scenes/{id}/comments/{threadId}/messages/{messageId}", h.principalHandler(auth, http.HandlerFunc(h.UpdateMessage)))
	router.Method(http.MethodDelete, "/scenes/{id}/comments/{threadId}/messages/{messageId}", h.principalHandler(auth, http.HandlerFunc(h.DeleteMessage)))
	router.Method(http.MethodPatch, "/scenes/{id}/comments/{threadId}", h.principalHandler(auth, http.HandlerFunc(h.UpdateThread)))
	return router
}

type createCommentRequest struct {
	RequestID string                `json:"requestId" validate:"required,uuid"`
	Anchor    *domain.CommentAnchor `json:"anchor" validate:"required"`
	Body      string                `json:"body" validate:"required"`
}
type replyCommentRequest struct {
	RequestID string `json:"requestId" validate:"required,uuid"`
	Body      string `json:"body" validate:"required"`
}
type editCommentRequest struct {
	RequestID        string `json:"requestId" validate:"required,uuid"`
	ExpectedRevision string `json:"expectedRevision" validate:"required"`
	Body             string `json:"body" validate:"required"`
}
type revisionCommentRequest struct {
	RequestID        string `json:"requestId" validate:"required,uuid"`
	ExpectedRevision string `json:"expectedRevision" validate:"required"`
}
type editThreadRequest struct {
	RequestID        string `json:"requestId" validate:"required,uuid"`
	ExpectedRevision string `json:"expectedRevision" validate:"required"`
	Resolved         *bool  `json:"resolved,omitempty"`
	Subscribed       *bool  `json:"subscribed,omitempty"`
}

func (h *CommentHandler) decode(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, commentRequestMaxBytes))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(value)
	if err == nil {
		var extra any
		err = decoder.Decode(&extra)
		if err == io.EOF {
			err = nil
		} else if err == nil {
			err = domain.ErrInvalidCommentInput
		}
	}
	if err != nil || h.validate.Struct(value) != nil {
		writeCommentError(w, r, domain.ErrInvalidCommentInput)
		return false
	}
	return true
}

func parseCommentRevision(value string) (int64, error) {
	revision, err := strconv.ParseInt(value, 10, 64)
	if err != nil || revision < 1 || strconv.FormatInt(revision, 10) != value {
		return 0, domain.ErrInvalidCommentInput
	}
	return revision, nil
}

func pathUUID(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, name))
}

func writeCommentError(w http.ResponseWriter, r *http.Request, err error) {
	code, message, status := "INVALID_INPUT", "Invalid comment request", http.StatusBadRequest
	switch {
	case errors.Is(err, domain.ErrRequestIDReused):
		code, message, status = "REQUEST_ID_REUSED", "Request ID was already used for a different mutation", http.StatusConflict
	case errors.Is(err, domain.ErrRevisionConflict):
		code, message, status = "REVISION_CONFLICT", "Comment changed; reload and retry", http.StatusConflict
	case errors.Is(err, domain.ErrCommentRateLimited):
		code, message, status = "RATE_LIMITED", "Comment write limit reached", http.StatusTooManyRequests
	case errors.Is(err, domain.ErrFeatureDisabled):
		code, message, status = "FEATURE_DISABLED", "Comments are disabled", http.StatusForbidden
	case errors.Is(err, domain.ErrInvalidCommentInput):
	default:
		middleware.WriteAccessError(w, r, err)
		return
	}
	writeJSONResponse(w, status, map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}

type commentResponse struct {
	ID         uuid.UUID            `json:"id"`
	ThreadID   uuid.UUID            `json:"threadId"`
	AuthorKind domain.PrincipalKind `json:"authorKind"`
	AuthorName string               `json:"authorName"`
	Body       string               `json:"body"`
	Revision   string               `json:"revision"`
	Mine       bool                 `json:"mine"`
	EditedAt   *time.Time           `json:"editedAt,omitempty"`
	DeletedAt  *time.Time           `json:"deletedAt,omitempty"`
	CreatedAt  time.Time            `json:"createdAt"`
	UpdatedAt  time.Time            `json:"updatedAt"`
}
type commentThreadResponse struct {
	ID          uuid.UUID            `json:"id"`
	SceneID     uuid.UUID            `json:"sceneId"`
	CreatorKind domain.PrincipalKind `json:"creatorKind"`
	CreatorName string               `json:"creatorName"`
	Anchor      domain.CommentAnchor `json:"anchor"`
	Status      string               `json:"status"`
	Revision    string               `json:"revision"`
	Subscribed  bool                 `json:"subscribed"`
	Messages    []commentResponse    `json:"messages"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
}

func commentPayload(value domain.Comment) commentResponse {
	return commentResponse{ID: value.ID, ThreadID: value.ThreadID, AuthorKind: value.AuthorKind, AuthorName: value.AuthorName, Body: value.Body, Revision: strconv.FormatInt(value.Revision, 10), Mine: value.Mine, EditedAt: value.EditedAt, DeletedAt: value.DeletedAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func threadPayload(value domain.CommentThread) commentThreadResponse {
	messages := make([]commentResponse, len(value.Messages))
	for i, message := range value.Messages {
		messages[i] = commentPayload(message)
	}
	return commentThreadResponse{ID: value.ID, SceneID: value.SceneID, CreatorKind: value.CreatorKind, CreatorName: value.CreatorName, Anchor: value.Anchor, Status: value.Status, Revision: strconv.FormatInt(value.Revision, 10), Subscribed: value.Subscribed, Messages: messages, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identify(r)
	if !ok {
		writeCommentError(w, r, domain.ErrAccessDenied)
		return
	}
	limit := 50
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
	}
	if err != nil {
		writeCommentError(w, r, domain.ErrInvalidCommentInput)
		return
	}
	page, err := h.comments.List(r.Context(), identity.principal, identity.ref, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeCommentError(w, r, err)
		return
	}
	items := make([]commentThreadResponse, len(page.Items))
	for i, item := range page.Items {
		items[i] = threadPayload(item)
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, map[string]any{"items": items, "nextCursor": page.NextCursor})
}

func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identify(r)
	if !ok {
		writeCommentError(w, r, domain.ErrAccessDenied)
		return
	}
	var input createCommentRequest
	if !h.decode(w, r, &input) {
		return
	}
	requestID, _ := uuid.Parse(input.RequestID)
	thread, err := h.comments.CreateThread(r.Context(), identity.principal, identity.ref, domain.CreateThreadInput{RequestID: requestID, Anchor: *input.Anchor, Body: input.Body})
	if err != nil {
		writeCommentError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusCreated, threadPayload(thread))
}

func (h *CommentHandler) Get(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identify(r)
	if !ok {
		writeCommentError(w, r, domain.ErrAccessDenied)
		return
	}
	threadID, err := pathUUID(r, "threadId")
	if err != nil {
		writeCommentError(w, r, domain.ErrInaccessible)
		return
	}
	thread, err := h.comments.GetThread(r.Context(), identity.principal, identity.ref, threadID)
	if err != nil {
		writeCommentError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, threadPayload(thread))
}

func (h *CommentHandler) Reply(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identify(r)
	if !ok {
		writeCommentError(w, r, domain.ErrAccessDenied)
		return
	}
	threadID, err := pathUUID(r, "threadId")
	if err != nil {
		writeCommentError(w, r, domain.ErrInaccessible)
		return
	}
	var input replyCommentRequest
	if !h.decode(w, r, &input) {
		return
	}
	requestID, _ := uuid.Parse(input.RequestID)
	message, err := h.comments.Reply(r.Context(), identity.principal, identity.ref, threadID, domain.ReplyInput{RequestID: requestID, Body: input.Body})
	if err != nil {
		writeCommentError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusCreated, commentPayload(message))
}

func (h *CommentHandler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identify(r)
	if !ok {
		writeCommentError(w, r, domain.ErrAccessDenied)
		return
	}
	threadID, e1 := pathUUID(r, "threadId")
	messageID, e2 := pathUUID(r, "messageId")
	if e1 != nil || e2 != nil {
		writeCommentError(w, r, domain.ErrInaccessible)
		return
	}
	var input editCommentRequest
	if !h.decode(w, r, &input) {
		return
	}
	requestID, _ := uuid.Parse(input.RequestID)
	revision, err := parseCommentRevision(input.ExpectedRevision)
	if err != nil {
		writeCommentError(w, r, err)
		return
	}
	message, err := h.comments.UpdateMessage(r.Context(), identity.principal, identity.ref, threadID, messageID, domain.CommentEditInput{RequestID: requestID, ExpectedRevision: revision, Body: input.Body})
	if err != nil {
		writeCommentError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, commentPayload(message))
}

func (h *CommentHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identify(r)
	if !ok {
		writeCommentError(w, r, domain.ErrAccessDenied)
		return
	}
	threadID, e1 := pathUUID(r, "threadId")
	messageID, e2 := pathUUID(r, "messageId")
	if e1 != nil || e2 != nil {
		writeCommentError(w, r, domain.ErrInaccessible)
		return
	}
	var input revisionCommentRequest
	if !h.decode(w, r, &input) {
		return
	}
	requestID, _ := uuid.Parse(input.RequestID)
	revision, err := parseCommentRevision(input.ExpectedRevision)
	if err != nil {
		writeCommentError(w, r, err)
		return
	}
	if err = h.comments.DeleteMessage(r.Context(), identity.principal, identity.ref, threadID, messageID, domain.RevisionInput{RequestID: requestID, ExpectedRevision: revision}); err != nil {
		writeCommentError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h *CommentHandler) UpdateThread(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.identify(r)
	if !ok {
		writeCommentError(w, r, domain.ErrAccessDenied)
		return
	}
	threadID, err := pathUUID(r, "threadId")
	if err != nil {
		writeCommentError(w, r, domain.ErrInaccessible)
		return
	}
	var input editThreadRequest
	if !h.decode(w, r, &input) {
		return
	}
	requestID, _ := uuid.Parse(input.RequestID)
	revision, err := parseCommentRevision(input.ExpectedRevision)
	if err != nil || input.Resolved == nil && input.Subscribed == nil {
		writeCommentError(w, r, domain.ErrInvalidCommentInput)
		return
	}
	thread, err := h.comments.UpdateThread(r.Context(), identity.principal, identity.ref, threadID, domain.ThreadEditInput{RequestID: requestID, ExpectedRevision: revision, Resolved: input.Resolved, Subscribed: input.Subscribed})
	if err != nil {
		writeCommentError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, threadPayload(thread))
}
