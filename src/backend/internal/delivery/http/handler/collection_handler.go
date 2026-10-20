package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

const collectionRequestMaxBytes int64 = 32 * 1024

type collectionHandlerAccess interface {
	ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error)
}

type CollectionHandler struct {
	collections domain.CollectionUsecase
	access      collectionHandlerAccess
}

func NewCollectionHandler(collections domain.CollectionUsecase, access collectionHandlerAccess) *CollectionHandler {
	return &CollectionHandler{collections: collections, access: access}
}

func (h *CollectionHandler) Register(router chi.Router, auth func(http.Handler) http.Handler) {
	router.Group(func(router chi.Router) {
		router.Use(auth)
		router.Get("/workspaces/{id}/collections", h.List)
		router.Post("/workspaces/{id}/collections", h.Create)
		router.Get("/workspaces/{id}/collections/{collectionId}", h.Get)
		router.Patch("/workspaces/{id}/collections/{collectionId}", h.Update)
		router.Delete("/workspaces/{id}/collections/{collectionId}", h.Delete)
		router.Post("/workspaces/{id}/collections/{collectionId}/scenes", h.AddScene)
		router.Delete("/workspaces/{id}/collections/{collectionId}/scenes/{sceneId}", h.RemoveScene)
		router.Put("/workspaces/{id}/collections/{collectionId}/order", h.Reorder)
	})
}

func writeCollectionError(w http.ResponseWriter, r *http.Request, err error) {
	code, message, status := "INVALID_INPUT", "Invalid collection request", http.StatusBadRequest
	switch {
	case errors.Is(err, domain.ErrInvalidCollection):
	case errors.Is(err, domain.ErrRequestIDReused):
		code, message, status = "REQUEST_ID_REUSED", "Request ID was already used", http.StatusConflict
	case errors.Is(err, domain.ErrRevisionConflict):
		code, message, status = "REVISION_CONFLICT", "Collection changed; reload and retry", http.StatusConflict
	case errors.Is(err, domain.ErrFeatureDisabled):
		code, message, status = "FEATURE_DISABLED", "Collections are disabled", http.StatusForbidden
	default:
		middleware.WriteAccessError(w, r, err)
		return
	}
	writeJSONResponse(w, status, map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}

func decodeCollectionRequest(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, collectionRequestMaxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return domain.ErrInvalidCollection
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return domain.ErrInvalidCollection
	}
	return nil
}

func (h *CollectionHandler) identity(r *http.Request) (domain.Principal, uuid.UUID, error) {
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

func collectionID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "collectionId"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidCollection
	}
	return id, nil
}

func requestRevision(requestID, revision string) (uuid.UUID, int64, error) {
	id, idErr := uuid.Parse(requestID)
	value, revisionErr := strconv.ParseInt(revision, 10, 64)
	if idErr != nil || revisionErr != nil || value < 1 || strconv.FormatInt(value, 10) != revision {
		return uuid.Nil, 0, domain.ErrInvalidCollection
	}
	return id, value, nil
}

func (h *CollectionHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, workspaceID, err := h.identity(r)
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			err = domain.ErrInvalidCollection
		}
	}
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	page, err := h.collections.List(r.Context(), principal, workspaceID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, page)
}

func (h *CollectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, workspaceID, err := h.identity(r)
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	var body struct {
		RequestID  string                      `json:"requestId"`
		Name       string                      `json:"name"`
		Visibility domain.CollectionVisibility `json:"visibility"`
	}
	err = decodeCollectionRequest(w, r, &body)
	requestID, parseErr := uuid.Parse(body.RequestID)
	if err != nil || parseErr != nil {
		writeCollectionError(w, r, domain.ErrInvalidCollection)
		return
	}
	value, err := h.collections.Create(r.Context(), principal, workspaceID, domain.CollectionCreateInput{RequestID: requestID, Name: body.Name, Visibility: body.Visibility})
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, value)
}

func (h *CollectionHandler) Get(w http.ResponseWriter, r *http.Request) {
	principal, workspaceID, err := h.identity(r)
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	id, parseErr := collectionID(r)
	if parseErr != nil {
		writeCollectionError(w, r, domain.ErrInvalidCollection)
		return
	}
	value, err := h.collections.Get(r.Context(), principal, workspaceID, id)
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, value)
}

func (h *CollectionHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal, workspaceID, err := h.identity(r)
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	id, parseErr := collectionID(r)
	var body struct {
		RequestID        string                      `json:"requestId"`
		ExpectedRevision string                      `json:"expectedRevision"`
		Name             string                      `json:"name"`
		Visibility       domain.CollectionVisibility `json:"visibility"`
	}
	if parseErr == nil {
		err = decodeCollectionRequest(w, r, &body)
	}
	requestID, revision, inputErr := requestRevision(body.RequestID, body.ExpectedRevision)
	if err != nil || parseErr != nil || inputErr != nil {
		writeCollectionError(w, r, domain.ErrInvalidCollection)
		return
	}
	value, err := h.collections.Update(r.Context(), principal, workspaceID, id, domain.CollectionUpdateInput{RequestID: requestID, ExpectedRevision: revision, Name: body.Name, Visibility: body.Visibility})
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, value)
}

func (h *CollectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	principal, workspaceID, err := h.identity(r)
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	id, parseErr := collectionID(r)
	var body struct {
		RequestID        string `json:"requestId"`
		ExpectedRevision string `json:"expectedRevision"`
	}
	if parseErr == nil {
		err = decodeCollectionRequest(w, r, &body)
	}
	requestID, revision, inputErr := requestRevision(body.RequestID, body.ExpectedRevision)
	if err != nil || parseErr != nil || inputErr != nil {
		writeCollectionError(w, r, domain.ErrInvalidCollection)
		return
	}
	if err = h.collections.Delete(r.Context(), principal, workspaceID, id, domain.CollectionRevisionInput{RequestID: requestID, ExpectedRevision: revision}); err != nil {
		writeCollectionError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CollectionHandler) AddScene(w http.ResponseWriter, r *http.Request) {
	principal, workspaceID, err := h.identity(r)
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	id, parseErr := collectionID(r)
	var body struct {
		RequestID        string `json:"requestId"`
		ExpectedRevision string `json:"expectedRevision"`
		SceneID          string `json:"sceneId"`
	}
	if parseErr == nil {
		err = decodeCollectionRequest(w, r, &body)
	}
	requestID, revision, inputErr := requestRevision(body.RequestID, body.ExpectedRevision)
	sceneID, sceneErr := uuid.Parse(body.SceneID)
	if err != nil || parseErr != nil || inputErr != nil || sceneErr != nil {
		writeCollectionError(w, r, domain.ErrInvalidCollection)
		return
	}
	value, err := h.collections.AddScene(r.Context(), principal, workspaceID, id, domain.CollectionSceneInput{RequestID: requestID, ExpectedRevision: revision, SceneID: sceneID})
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, value)
}

func (h *CollectionHandler) RemoveScene(w http.ResponseWriter, r *http.Request) {
	principal, workspaceID, err := h.identity(r)
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	id, parseErr := collectionID(r)
	sceneID, sceneErr := uuid.Parse(chi.URLParam(r, "sceneId"))
	var body struct {
		RequestID        string `json:"requestId"`
		ExpectedRevision string `json:"expectedRevision"`
	}
	if parseErr == nil && sceneErr == nil {
		err = decodeCollectionRequest(w, r, &body)
	}
	requestID, revision, inputErr := requestRevision(body.RequestID, body.ExpectedRevision)
	if err != nil || parseErr != nil || sceneErr != nil || inputErr != nil {
		writeCollectionError(w, r, domain.ErrInvalidCollection)
		return
	}
	value, err := h.collections.RemoveScene(r.Context(), principal, workspaceID, id, domain.CollectionSceneInput{RequestID: requestID, ExpectedRevision: revision, SceneID: sceneID})
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, value)
}

func (h *CollectionHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	principal, workspaceID, err := h.identity(r)
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	id, parseErr := collectionID(r)
	var body struct {
		RequestID        string   `json:"requestId"`
		ExpectedRevision string   `json:"expectedRevision"`
		SceneIDs         []string `json:"sceneIds"`
	}
	if parseErr == nil {
		err = decodeCollectionRequest(w, r, &body)
	}
	requestID, revision, inputErr := requestRevision(body.RequestID, body.ExpectedRevision)
	sceneIDs := make([]uuid.UUID, len(body.SceneIDs))
	for index, raw := range body.SceneIDs {
		sceneIDs[index], parseErr = uuid.Parse(raw)
		if parseErr != nil {
			break
		}
	}
	if err != nil || parseErr != nil || inputErr != nil {
		writeCollectionError(w, r, domain.ErrInvalidCollection)
		return
	}
	value, err := h.collections.Reorder(r.Context(), principal, workspaceID, id, domain.CollectionReorderInput{RequestID: requestID, ExpectedRevision: revision, SceneIDs: sceneIDs})
	if err != nil {
		writeCollectionError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, value)
}
