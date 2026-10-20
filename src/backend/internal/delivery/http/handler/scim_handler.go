package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const scimRequestLimit int64 = 1024 * 1024

type scimHandlerAccess interface {
	ResolveWorkspaceID(domain.Context, string) (uuid.UUID, error)
}
type SCIMHandler struct {
	service  domain.SCIMUsecase
	access   scimHandlerAccess
	scimAuth func(http.Handler) http.Handler
}

func NewSCIMHandler(service domain.SCIMUsecase, access scimHandlerAccess, scimAuth func(http.Handler) http.Handler) *SCIMHandler {
	return &SCIMHandler{service: service, access: access, scimAuth: scimAuth}
}
func (h *SCIMHandler) Register(router chi.Router, loginAuth func(http.Handler) http.Handler) {
	router.Group(func(r chi.Router) {
		r.Use(loginAuth)
		r.Get("/workspaces/{id}/scim/credential", h.TokenStatus)
		r.Post("/workspaces/{id}/scim/credential", h.RotateToken)
		r.Delete("/workspaces/{id}/scim/credential", h.RevokeToken)
		r.Post("/workspaces/{id}/scim/users/{scimUserId}/adopt", h.AdoptUser)
	})
	router.Route("/scim/v2", func(r chi.Router) {
		r.Use(h.scimAuth)
		r.Get("/ServiceProviderConfig", h.ServiceProviderConfig)
		r.Get("/ResourceTypes", h.ResourceTypes)
		r.Get("/Schemas", h.Schemas)
		r.MethodFunc(http.MethodPost, "/Bulk", func(w http.ResponseWriter, r *http.Request) { writeSCIMError(w, domain.ErrSCIMUnsupported) })
		r.Get("/Users", h.ListUsers)
		r.Post("/Users", h.CreateUser)
		r.Get("/Users/{id}", h.GetUser)
		r.Put("/Users/{id}", h.ReplaceUser)
		r.Patch("/Users/{id}", h.PatchUser)
		r.Delete("/Users/{id}", h.DeleteUser)
		r.Get("/Groups", h.ListGroups)
		r.Post("/Groups", h.CreateGroup)
		r.Get("/Groups/{id}", h.GetGroup)
		r.Put("/Groups/{id}", h.ReplaceGroup)
		r.Patch("/Groups/{id}", h.PatchGroup)
		r.Delete("/Groups/{id}", h.DeleteGroup)
	})
}
func decodeSCIM(w http.ResponseWriter, r *http.Request, out any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, scimRequestLimit))
	d.DisallowUnknownFields()
	if e := d.Decode(out); e != nil {
		return domain.ErrSCIMInvalid
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return domain.ErrSCIMInvalid
	}
	return nil
}
func writeSCIM(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeSCIMError(w http.ResponseWriter, e error) {
	status, kind, detail := http.StatusInternalServerError, "", "SCIM request failed"
	switch {
	case errors.Is(e, domain.ErrSCIMInvalid):
		status, kind, detail = http.StatusBadRequest, "invalidValue", "Invalid SCIM request"
	case errors.Is(e, domain.ErrSCIMUnauthorized):
		status, detail = http.StatusUnauthorized, "Invalid SCIM credential"
	case errors.Is(e, domain.ErrSCIMNotFound):
		status, detail = http.StatusNotFound, "SCIM resource not found"
	case errors.Is(e, domain.ErrSCIMConflict):
		status, kind, detail = http.StatusConflict, "uniqueness", "SCIM resource conflicts with an existing resource"
	case errors.Is(e, domain.ErrSCIMPrecondition):
		status, detail = http.StatusPreconditionFailed, "SCIM resource version changed"
	case errors.Is(e, domain.ErrSCIMUnsupported):
		status, detail = http.StatusNotImplemented, "SCIM operation is not supported"
	case errors.Is(e, domain.ErrAccessDenied), errors.Is(e, domain.ErrInaccessible):
		status, detail = http.StatusForbidden, "Workspace owner access required"
	}
	writeSCIM(w, status, domain.SCIMError{Schemas: []string{domain.SCIMErrorSchema}, Status: strconv.Itoa(status), SCIMType: kind, Detail: detail})
}
func (h *SCIMHandler) owner(r *http.Request) (domain.Principal, uuid.UUID, error) {
	p, ok := middleware.UserPrincipalFromContext(r.Context())
	if !ok {
		return domain.Principal{}, uuid.Nil, domain.ErrAccessDenied
	}
	w, e := h.access.ResolveWorkspaceID(r.Context(), chi.URLParam(r, "id"))
	return p, w, e
}
func (h *SCIMHandler) TokenStatus(w http.ResponseWriter, r *http.Request) {
	p, id, e := h.owner(r)
	if e == nil {
		var v domain.SCIMTokenStatus
		v, e = h.service.TokenStatus(r.Context(), p, id)
		if e == nil {
			writeJSONResponse(w, http.StatusOK, v)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) RotateToken(w http.ResponseWriter, r *http.Request) {
	p, id, e := h.owner(r)
	if e == nil {
		var v domain.SCIMTokenResult
		v, e = h.service.RotateToken(r.Context(), p, id)
		if e == nil {
			w.Header().Set("Cache-Control", "no-store")
			writeJSONResponse(w, http.StatusCreated, v)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	p, id, e := h.owner(r)
	if e == nil {
		e = h.service.RevokeToken(r.Context(), p, id)
	}
	if e != nil {
		writeSCIMError(w, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *SCIMHandler) AdoptUser(w http.ResponseWriter, r *http.Request) {
	p, workspaceID, e := h.owner(r)
	userID, parseErr := uuid.Parse(chi.URLParam(r, "scimUserId"))
	if e == nil && (parseErr != nil || userID == uuid.Nil) {
		e = domain.ErrSCIMNotFound
	}
	if e == nil {
		var out domain.SCIMUser
		out, e = h.service.AdoptUser(r.Context(), p, workspaceID, userID)
		if e == nil {
			writeJSONResponse(w, http.StatusOK, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func scimScope(r *http.Request) (uuid.UUID, error) {
	w, ok := middleware.SCIMWorkspaceFromContext(r.Context())
	if !ok {
		return uuid.Nil, domain.ErrSCIMUnauthorized
	}
	return w, nil
}
func resourceID(r *http.Request) (uuid.UUID, error) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrSCIMNotFound
	}
	return id, nil
}
func setResourceHeaders(w http.ResponseWriter, base string, id string, meta domain.SCIMMeta) {
	w.Header().Set("ETag", meta.Version)
	w.Header().Set("Location", strings.TrimRight(base, "/")+"/"+id)
}
func (h *SCIMHandler) ServiceProviderConfig(w http.ResponseWriter, _ *http.Request) {
	writeSCIM(w, http.StatusOK, map[string]any{"schemas": []string{domain.SCIMServiceProviderSchema}, "patch": map[string]bool{"supported": true}, "bulk": map[string]any{"supported": false, "maxOperations": 0, "maxPayloadSize": 0}, "filter": map[string]any{"supported": true, "maxResults": domain.SCIMMaxPageSize}, "changePassword": map[string]bool{"supported": false}, "sort": map[string]bool{"supported": false}, "etag": map[string]bool{"supported": true}, "authenticationSchemes": []map[string]any{{"type": "oauthbearertoken", "name": "Bearer Token", "description": "Dedicated workspace SCIM token", "primary": true}}})
}
func (h *SCIMHandler) ResourceTypes(w http.ResponseWriter, _ *http.Request) {
	writeSCIM(w, http.StatusOK, domain.SCIMPage[map[string]any]{Schemas: []string{domain.SCIMListSchema}, TotalResults: 2, StartIndex: 1, ItemsPerPage: 2, Resources: []map[string]any{{"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"}, "id": "User", "name": "User", "endpoint": "/Users", "schema": domain.SCIMUserSchema}, {"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"}, "id": "Group", "name": "Group", "endpoint": "/Groups", "schema": domain.SCIMGroupSchema}}})
}
func (h *SCIMHandler) Schemas(w http.ResponseWriter, _ *http.Request) {
	writeSCIM(w, http.StatusOK, domain.SCIMPage[map[string]any]{Schemas: []string{domain.SCIMListSchema}, TotalResults: 2, StartIndex: 1, ItemsPerPage: 2, Resources: []map[string]any{{"id": domain.SCIMUserSchema, "name": "User", "description": "Workspace user", "attributes": []any{}}, {"id": domain.SCIMGroupSchema, "name": "Group", "description": "Workspace group", "attributes": []any{}}}})
}
func queryPage(r *http.Request) (int, int, error) {
	start, count := 1, domain.SCIMMaxPageSize
	var e error
	if raw := r.URL.Query().Get("startIndex"); raw != "" {
		start, e = strconv.Atoi(raw)
		if e != nil || start < 1 {
			return 0, 0, domain.ErrSCIMInvalid
		}
	}
	if raw := r.URL.Query().Get("count"); raw != "" {
		count, e = strconv.Atoi(raw)
		if e != nil || count < 0 {
			return 0, 0, domain.ErrSCIMInvalid
		}
	}
	return start, count, nil
}
func (h *SCIMHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	var in domain.SCIMUser
	if e == nil {
		e = decodeSCIM(w, r, &in)
	}
	if e == nil {
		var out domain.SCIMUser
		out, e = h.service.CreateUser(r.Context(), scope, in)
		if e == nil {
			setResourceHeaders(w, "/api/v1/scim/v2/Users", out.ID, out.Meta)
			writeSCIM(w, http.StatusCreated, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	id, ie := resourceID(r)
	if e == nil {
		e = ie
	}
	if e == nil {
		var out domain.SCIMUser
		out, e = h.service.GetUser(r.Context(), scope, id)
		if e == nil {
			setResourceHeaders(w, "/api/v1/scim/v2/Users", out.ID, out.Meta)
			writeSCIM(w, http.StatusOK, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	start, count, pe := queryPage(r)
	if e == nil {
		e = pe
	}
	if e == nil {
		var out domain.SCIMPage[domain.SCIMUser]
		out, e = h.service.ListUsers(r.Context(), scope, r.URL.Query().Get("filter"), start, count)
		if e == nil {
			writeSCIM(w, http.StatusOK, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) ReplaceUser(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	id, ie := resourceID(r)
	if e == nil {
		e = ie
	}
	var in domain.SCIMUser
	if e == nil {
		e = decodeSCIM(w, r, &in)
	}
	if e == nil {
		var out domain.SCIMUser
		out, e = h.service.ReplaceUser(r.Context(), scope, id, r.Header.Get("If-Match"), in)
		if e == nil {
			setResourceHeaders(w, "/api/v1/scim/v2/Users", out.ID, out.Meta)
			writeSCIM(w, http.StatusOK, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) PatchUser(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	id, ie := resourceID(r)
	if e == nil {
		e = ie
	}
	var in domain.SCIMPatchRequest
	if e == nil {
		e = decodeSCIM(w, r, &in)
	}
	if e == nil {
		var out domain.SCIMUser
		out, e = h.service.PatchUser(r.Context(), scope, id, r.Header.Get("If-Match"), in)
		if e == nil {
			setResourceHeaders(w, "/api/v1/scim/v2/Users", out.ID, out.Meta)
			writeSCIM(w, http.StatusOK, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	id, ie := resourceID(r)
	if e == nil {
		e = ie
	}
	if e == nil {
		var out domain.SCIMUser
		out, e = h.service.DeleteUser(r.Context(), scope, id, r.Header.Get("If-Match"))
		if e == nil {
			setResourceHeaders(w, "/api/v1/scim/v2/Users", out.ID, out.Meta)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	var in domain.SCIMGroup
	if e == nil {
		e = decodeSCIM(w, r, &in)
	}
	if e == nil {
		var out domain.SCIMGroup
		out, e = h.service.CreateGroup(r.Context(), scope, in)
		if e == nil {
			setResourceHeaders(w, "/api/v1/scim/v2/Groups", out.ID, out.Meta)
			writeSCIM(w, http.StatusCreated, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	id, ie := resourceID(r)
	if e == nil {
		e = ie
	}
	if e == nil {
		var out domain.SCIMGroup
		out, e = h.service.GetGroup(r.Context(), scope, id)
		if e == nil {
			setResourceHeaders(w, "/api/v1/scim/v2/Groups", out.ID, out.Meta)
			writeSCIM(w, http.StatusOK, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	start, count, pe := queryPage(r)
	if e == nil {
		e = pe
	}
	if e == nil {
		var out domain.SCIMPage[domain.SCIMGroup]
		out, e = h.service.ListGroups(r.Context(), scope, r.URL.Query().Get("filter"), start, count)
		if e == nil {
			writeSCIM(w, http.StatusOK, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) ReplaceGroup(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	id, ie := resourceID(r)
	if e == nil {
		e = ie
	}
	var in domain.SCIMGroup
	if e == nil {
		e = decodeSCIM(w, r, &in)
	}
	if e == nil {
		var out domain.SCIMGroup
		out, e = h.service.ReplaceGroup(r.Context(), scope, id, r.Header.Get("If-Match"), in)
		if e == nil {
			setResourceHeaders(w, "/api/v1/scim/v2/Groups", out.ID, out.Meta)
			writeSCIM(w, http.StatusOK, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) PatchGroup(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	id, ie := resourceID(r)
	if e == nil {
		e = ie
	}
	var in domain.SCIMPatchRequest
	if e == nil {
		e = decodeSCIM(w, r, &in)
	}
	if e == nil {
		var out domain.SCIMGroup
		out, e = h.service.PatchGroup(r.Context(), scope, id, r.Header.Get("If-Match"), in)
		if e == nil {
			setResourceHeaders(w, "/api/v1/scim/v2/Groups", out.ID, out.Meta)
			writeSCIM(w, http.StatusOK, out)
			return
		}
	}
	writeSCIMError(w, e)
}
func (h *SCIMHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	scope, e := scimScope(r)
	id, ie := resourceID(r)
	if e == nil {
		e = ie
	}
	if e == nil {
		e = h.service.DeleteGroup(r.Context(), scope, id, r.Header.Get("If-Match"))
	}
	if e != nil {
		writeSCIMError(w, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
