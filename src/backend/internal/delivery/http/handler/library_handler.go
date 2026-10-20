package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

const libraryRequestMaxBytes int64 = 25*1024*1024 + 64*1024

type libraryHandlerAccess interface {
	ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error)
	ResolveSceneID(domain.Context, uuid.UUID, string) (uuid.UUID, error)
}
type LibraryHandler struct {
	libraries domain.LibraryUsecase
	access    libraryHandlerAccess
}

func NewLibraryHandler(libraries domain.LibraryUsecase, access libraryHandlerAccess) *LibraryHandler {
	return &LibraryHandler{libraries: libraries, access: access}
}
func (h *LibraryHandler) Register(router chi.Router, auth func(http.Handler) http.Handler) {
	router.Group(func(r chi.Router) {
		r.Use(auth)
		r.Get("/workspaces/{id}/libraries", h.List)
		r.Post("/workspaces/{id}/libraries/import", h.Import)
		r.Put("/workspaces/{id}/libraries/{libraryId}", h.Publish)
		r.Get("/workspaces/{id}/libraries/{libraryId}/revisions/{revision}", h.GetRevision)
		r.Get("/workspaces/{id}/libraries/{libraryId}/revisions/{revision}/export", h.Export)
		r.Post("/workspaces/{id}/scenes/{sceneId}/library-insertions", h.Insert)
	})
}
func decodeLibraryRequest(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, libraryRequestMaxBytes))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil {
		return domain.ErrInvalidLibrary
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return domain.ErrInvalidLibrary
	}
	return nil
}
func writeLibraryError(w http.ResponseWriter, r *http.Request, err error) {
	code, message, status := "INVALID_LIBRARY", "Invalid library request", http.StatusBadRequest
	switch {
	case errors.Is(err, domain.ErrInvalidLibrary):
	case errors.Is(err, domain.ErrRevisionConflict):
		code, message, status = "REVISION_CONFLICT", "Library changed; reload and retry", http.StatusConflict
	case errors.Is(err, domain.ErrFeatureDisabled):
		code, message, status = "FEATURE_DISABLED", "Workspace libraries are disabled", http.StatusForbidden
	default:
		middleware.WriteAccessError(w, r, err)
		return
	}
	writeJSONResponse(w, status, map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}
func (h *LibraryHandler) identity(r *http.Request) (domain.Principal, uuid.UUID, error) {
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return domain.Principal{}, uuid.Nil, domain.ErrAccessDenied
	}
	workspace, err := h.access.ResolveWorkspaceID(r.Context(), chi.URLParam(r, "id"))
	principal := domain.UserPrincipal(user)
	principal.CredentialExpiresAt, _ = r.Context().Value(domain.ContextKeyCredentialExpiry).(time.Time)
	return principal, workspace, err
}
func parseLibraryID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "libraryId"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidLibrary
	}
	return id, nil
}
func parseLibraryRevision(r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "revision")
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 || strconv.FormatInt(n, 10) != raw {
		return 0, domain.ErrInvalidLibrary
	}
	return n, nil
}

type libraryWriteBody struct {
	RequestID        uuid.UUID                `json:"requestId"`
	ExpectedRevision string                   `json:"expectedRevision,omitempty"`
	Name             string                   `json:"name"`
	Content          json.RawMessage          `json:"content"`
	Assets           []domain.LibraryAssetPin `json:"assets,omitempty"`
}

func (h *LibraryHandler) Import(w http.ResponseWriter, r *http.Request) {
	p, workspace, err := h.identity(r)
	var body libraryWriteBody
	if err == nil {
		err = decodeLibraryRequest(w, r, &body)
	}
	if err != nil {
		writeLibraryError(w, r, err)
		return
	}
	item, revision, err := h.libraries.Import(r.Context(), p, workspace, domain.LibraryImportInput{RequestID: body.RequestID, Name: body.Name, Content: body.Content, Assets: body.Assets})
	if err != nil {
		writeLibraryError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, map[string]any{"item": item, "revision": revision})
}
func (h *LibraryHandler) Publish(w http.ResponseWriter, r *http.Request) {
	p, workspace, err := h.identity(r)
	id, idErr := parseLibraryID(r)
	var body libraryWriteBody
	if err == nil && idErr == nil {
		err = decodeLibraryRequest(w, r, &body)
	}
	expected, parseErr := strconv.ParseInt(body.ExpectedRevision, 10, 64)
	if err != nil || idErr != nil || parseErr != nil || expected < 1 {
		writeLibraryError(w, r, domain.ErrInvalidLibrary)
		return
	}
	item, revision, err := h.libraries.Publish(r.Context(), p, workspace, id, domain.LibraryPublishInput{RequestID: body.RequestID, ExpectedRevision: expected, Name: body.Name, Content: body.Content, Assets: body.Assets})
	if err != nil {
		writeLibraryError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]any{"item": item, "revision": revision})
}
func (h *LibraryHandler) List(w http.ResponseWriter, r *http.Request) {
	p, workspace, err := h.identity(r)
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
	}
	if err != nil {
		writeLibraryError(w, r, domain.ErrInvalidLibrary)
		return
	}
	page, err := h.libraries.List(r.Context(), p, workspace, r.URL.Query().Get("q"), limit)
	if err != nil {
		writeLibraryError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, page)
}
func (h *LibraryHandler) GetRevision(w http.ResponseWriter, r *http.Request) {
	p, workspace, err := h.identity(r)
	id, idErr := parseLibraryID(r)
	revision, revisionErr := parseLibraryRevision(r)
	if err != nil || idErr != nil || revisionErr != nil {
		writeLibraryError(w, r, domain.ErrInvalidLibrary)
		return
	}
	value, err := h.libraries.GetRevision(r.Context(), p, workspace, id, revision)
	if err != nil {
		writeLibraryError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, value)
}
func (h *LibraryHandler) Export(w http.ResponseWriter, r *http.Request) {
	p, workspace, err := h.identity(r)
	id, idErr := parseLibraryID(r)
	revision, revisionErr := parseLibraryRevision(r)
	if err != nil || idErr != nil || revisionErr != nil {
		writeLibraryError(w, r, domain.ErrInvalidLibrary)
		return
	}
	value, err := h.libraries.Export(r.Context(), p, workspace, id, revision)
	if err != nil {
		writeLibraryError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="library-%s-r%d.excalidrawlib"`, id, revision))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(value)
}
func (h *LibraryHandler) Insert(w http.ResponseWriter, r *http.Request) {
	p, workspace, err := h.identity(r)
	scene, sceneErr := h.access.ResolveSceneID(r.Context(), workspace, chi.URLParam(r, "sceneId"))
	var body struct {
		LibraryItemID    uuid.UUID `json:"libraryItemId"`
		NativeItemID     string    `json:"nativeItemId,omitempty"`
		Revision         int64     `json:"revision"`
		ExpectedSequence string    `json:"expectedSequence"`
		RequestID        uuid.UUID `json:"requestId"`
	}
	if err == nil && sceneErr == nil {
		err = decodeLibraryRequest(w, r, &body)
	}
	if err != nil || sceneErr != nil {
		writeLibraryError(w, r, domain.ErrInvalidLibrary)
		return
	}
	result, err := h.libraries.Insert(r.Context(), p, domain.SceneRef{WorkspaceID: workspace, SceneID: scene}, domain.LibraryInsertInput{LibraryItemID: body.LibraryItemID, NativeItemID: body.NativeItemID, Revision: body.Revision, ExpectedSequence: body.ExpectedSequence, RequestID: body.RequestID})
	if err != nil {
		writeLibraryError(w, r, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}
