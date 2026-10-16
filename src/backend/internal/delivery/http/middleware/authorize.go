package middleware

import (
	"bytes"
	"context"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func WriteAccessError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := http.StatusServiceUnavailable, "AUTHORITY_UNAVAILABLE", "Authorization unavailable"
	if errors.Is(err, domain.ErrInaccessible) {
		status, code, message = http.StatusNotFound, "NOT_FOUND", "Resource not found"
	}
	if errors.Is(err, domain.ErrAccessDenied) || errors.Is(err, domain.ErrOwnerProtected) {
		status, code, message = http.StatusForbidden, "FORBIDDEN", "Operation is not permitted"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
}

type accessResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *accessResponse) Header() http.Header { return r.header }
func (r *accessResponse) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}
func (r *accessResponse) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(b)
}

const (
	scopedBodyReadTimeout         = 10 * time.Second
	sceneRequestMaxBytes    int64 = 26 << 20 // 25 MiB document plus JSON/metadata envelope.
	metadataRequestMaxBytes int64 = 1 << 20
)

// Fully consume network input before taking an authority lock. The real HTTP
// server must support read deadlines; unsupported transports fail closed.
func bufferScopedBody(w http.ResponseWriter, r *http.Request, resource string) bool {
	if r.Body == nil || r.Body == http.NoBody {
		return true
	}
	limit := metadataRequestMaxBytes
	if resource == "scenes" {
		limit = sceneRequestMaxBytes
	}
	controller := http.NewResponseController(w)
	if err := controller.SetReadDeadline(time.Now().Add(scopedBodyReadTimeout)); err != nil {
		WriteAccessError(w, r, err)
		return false
	}
	reject := func(status int, code, message string) {
		// An incomplete body must not be drained indefinitely by net/http after the
		// handler returns. Keep the failed connection expired and request closure.
		_ = controller.SetReadDeadline(time.Now())
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message, "requestId": chimiddleware.GetReqID(r.Context())})
	}
	if r.ContentLength > limit {
		reject(http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body exceeds limit")
		return false
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		var size *http.MaxBytesError
		var timeout net.Error
		switch {
		case errors.As(err, &size):
			reject(http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "Request body exceeds limit")
		case errors.As(err, &timeout) && timeout.Timeout():
			reject(http.StatusRequestTimeout, "BODY_TIMEOUT", "Request body read timed out")
		default:
			reject(http.StatusBadRequest, "INVALID_BODY", "Unable to read request body")
		}
		return false
	}
	_ = r.Body.Close()
	if err = controller.SetReadDeadline(time.Time{}); err != nil {
		WriteAccessError(w, r, err)
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return true
}

var errHandlerRejected = errors.New("handler rejected operation")

// CollaborationAccessMiddleware holds authority through commit and buffers the
// response so a failed transaction can never acknowledge a successful write.
func CollaborationAccessMiddleware(access domain.AccessService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			if len(parts) < 3 {
				next.ServeHTTP(w, r)
				return
			}
			resource := parts[2]
			scoped := resource == "scenes" || resource == "folders" || resource == "assets" || resource == "search"
			management := resource == "workspaces" && len(parts) > 4 && parts[4] == "members"
			if !scoped && !management {
				next.ServeHTTP(w, r)
				return
			}
			user, ok := UserIDFromContext(r.Context())
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !bufferScopedBody(w, r, resource) {
				return
			}
			raw := r.Header.Get(domain.HeaderWorkspaceID)
			if management {
				raw = parts[3]
			}
			workspace, e := access.ResolveWorkspaceID(r.Context(), raw)
			if e != nil {
				WriteAccessError(w, r, e)
				return
			}
			response := &accessResponse{header: make(http.Header)}
			err := access.WithWorkspaceLock(r.Context(), workspace, user, func(ctx context.Context) error {
				p := domain.UserPrincipal(user)
				d, e := access.ResolveWorkspace(ctx, p, workspace)
				if e != nil {
					return e
				}
				ownerOnly := management || r.URL.Query().Get("trash") == "true" || r.URL.Query().Get("view") == domain.SearchViewTrash
				if len(parts) > 4 && (parts[4] == "members" || parts[4] == "share" || parts[4] == "share-links" || parts[4] == "restore") {
					ownerOnly = true
				}
				if ownerOnly {
					if e := access.RequireOwner(ctx, user, workspace); e != nil {
						return e
					}
				}
				if resource == "scenes" && len(parts) > 3 {
					id, e := access.ResolveSceneID(ctx, workspace, parts[3])
					if e != nil {
						return e
					}
					if !(len(parts) > 4 && parts[4] == "restore") {
						d, e = access.Resolve(ctx, p, domain.SceneRef{WorkspaceID: workspace, SceneID: id})
						if e != nil {
							return e
						}
					}
					ctx = context.WithValue(ctx, domain.ContextKeySceneID, id)
				}
				read := r.Method == http.MethodGet || r.Method == http.MethodHead || resource == "assets" && len(parts) > 3 && parts[3] == "presign-view"
				if !read && !ownerOnly && resource != "assets" && d.Access != domain.AccessEdit {
					return domain.ErrAccessDenied
				}
				if resource == "folders" && len(parts) > 3 && !read && !ownerOnly {
					id, e := uuid.Parse(parts[3])
					if e != nil {
						return domain.ErrInaccessible
					}
					if e = access.RequireFolderEdit(ctx, p, workspace, id); e != nil {
						return e
					}
				}
				r.Header.Set(domain.HeaderWorkspaceID, workspace.String())
				ctx = context.WithValue(ctx, domain.ContextKeyWorkspaceID, workspace)
				next.ServeHTTP(response, r.WithContext(ctx))
				if response.status >= http.StatusBadRequest {
					return errHandlerRejected
				}
				return nil
			})
			if err != nil && !errors.Is(err, errHandlerRejected) {
				WriteAccessError(w, r, err)
				return
			}
			for k, v := range response.header {
				w.Header()[k] = v
			}
			if response.status == 0 {
				response.status = http.StatusOK
			}
			w.WriteHeader(response.status)
			_, _ = w.Write(response.body.Bytes())
		})
	}
}
