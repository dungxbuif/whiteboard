package handler

import (
	"context"
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"strings"
)

type AgentHandler struct {
	agents *usecase.AgentService
	rooms  domain.RoomService
	access domain.AccessService
}

func NewAgentHandler(a *usecase.AgentService, r domain.RoomService, access domain.AccessService) *AgentHandler {
	return &AgentHandler{a, r, access}
}
func agentError(w http.ResponseWriter, r *http.Request, e error) {
	switch {
	case errors.Is(e, domain.ErrAgentCredential):
		writeSharingError(w, r, 401, "UNAUTHENTICATED", "Invalid or revoked agent key")
	case errors.Is(e, domain.ErrAgentInput):
		writeSharingError(w, r, 400, "INVALID_INPUT", "Invalid agent request")
	case errors.Is(e, domain.ErrAgentProjection):
		writeSharingError(w, r, 413, "PROJECTION_TOO_LARGE", "Scene exceeds 500 elements or 256 KiB")
	default:
		middleware.WriteAccessError(w, r, e)
	}
}
func (h *AgentHandler) management(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	u, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, uuid.Nil, domain.ErrAccessDenied
	}
	w, e := h.access.ResolveWorkspaceID(r.Context(), chi.URLParam(r, "id"))
	return u, w, e
}
func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Scopes   []string    `json:"scopes"`
		SceneIDs []uuid.UUID `json:"sceneIds"`
	}
	if !decodeSharing(w, r, &in) {
		return
	}
	u, id, e := h.management(r)
	if e != nil {
		agentError(w, r, e)
		return
	}
	out, e := h.agents.CreateAgentKey(r.Context(), u, in.Scopes, domain.AgentTargets{WorkspaceID: id, SceneIDs: in.SceneIDs})
	if e != nil {
		agentError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, 201, out)
}
func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	u, id, e := h.management(r)
	if e != nil {
		agentError(w, r, e)
		return
	}
	out, e := h.agents.ListKeys(r.Context(), u, id)
	if e != nil {
		agentError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, 200, out)
}
func (h *AgentHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	u, id, e := h.management(r)
	if e != nil {
		agentError(w, r, e)
		return
	}
	key, e := uuid.Parse(chi.URLParam(r, "keyId"))
	if e != nil {
		agentError(w, r, domain.ErrAgentInput)
		return
	}
	if e = h.agents.RevokeKey(r.Context(), u, id, key); e != nil {
		agentError(w, r, e)
		return
	}
	w.WriteHeader(204)
}

type agentContextKey struct{}

func agentPrincipal(r *http.Request) domain.Principal {
	return r.Context().Value(agentContextKey{}).(domain.Principal)
}
func (h *AgentHandler) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.Header.Get("Authorization"), " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			agentError(w, r, domain.ErrAgentCredential)
			return
		}
		p, e := h.agents.AuthenticateAgent(r.Context(), parts[1])
		if e != nil {
			agentError(w, r, e)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), agentContextKey{}, p)))
	})
}
func agentPageInput(r *http.Request) (string, int, error) {
	limit := 50
	var e error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, e = strconv.Atoi(raw)
	}
	if e != nil || limit < 1 || limit > 50 {
		return "", 0, domain.ErrAgentInput
	}
	return r.URL.Query().Get("cursor"), limit, nil
}
func agentRef(r *http.Request) (domain.SceneRef, error) {
	w, e := uuid.Parse(chi.URLParam(r, "workspaceId"))
	if e != nil {
		return domain.SceneRef{}, domain.ErrAgentInput
	}
	id := uuid.Nil
	if raw := chi.URLParam(r, "sceneId"); raw != "" {
		id, e = uuid.Parse(raw)
		if e != nil {
			return domain.SceneRef{}, domain.ErrAgentInput
		}
	}
	return domain.SceneRef{WorkspaceID: w, SceneID: id}, nil
}
func (h *AgentHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(agentCorrelation, h.Authenticate)
	r.Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		p := agentPrincipal(r)
		writeJSONResponse(w, 200, map[string]string{"principalId": p.ID, "keyId": p.GrantID})
	})
	r.Get("/workspaces", func(w http.ResponseWriter, r *http.Request) {
		c, n, e := agentPageInput(r)
		if e != nil {
			agentError(w, r, e)
			return
		}
		out, e := h.agents.ListWorkspaces(r.Context(), agentPrincipal(r), c, n)
		if e != nil {
			agentError(w, r, e)
			return
		}
		writeJSONResponse(w, 200, out)
	})
	r.Get("/workspaces/{workspaceId}/scenes", func(w http.ResponseWriter, r *http.Request) {
		ref, e := agentRef(r)
		if e != nil {
			agentError(w, r, e)
			return
		}
		c, n, e := agentPageInput(r)
		if e != nil {
			agentError(w, r, e)
			return
		}
		out, e := h.agents.ListScenes(r.Context(), agentPrincipal(r), ref.WorkspaceID, c, n)
		if e != nil {
			agentError(w, r, e)
			return
		}
		writeJSONResponse(w, 200, out)
	})
	r.Post("/workspaces/{workspaceId}/scenes", func(w http.ResponseWriter, r *http.Request) {
		ref, e := agentRef(r)
		if e != nil {
			agentError(w, r, e)
			return
		}
		var in struct {
			Title string `json:"title"`
		}
		if !decodeSharing(w, r, &in) {
			return
		}
		out, e := h.agents.CreateScene(r.Context(), agentPrincipal(r), ref.WorkspaceID, in.Title)
		if e != nil {
			agentError(w, r, e)
			return
		}
		writeJSONResponse(w, 201, out)
	})
	r.Get("/workspaces/{workspaceId}/scenes/{sceneId}", func(w http.ResponseWriter, r *http.Request) {
		ref, e := agentRef(r)
		if e != nil {
			agentError(w, r, e)
			return
		}
		out, e := h.agents.GetScene(r.Context(), agentPrincipal(r), ref)
		if e != nil {
			agentError(w, r, e)
			return
		}
		writeJSONResponse(w, 200, out)
	})
	r.Post("/workspaces/{workspaceId}/scenes/{sceneId}/room-token", func(w http.ResponseWriter, r *http.Request) {
		ref, e := agentRef(r)
		if e != nil {
			agentError(w, r, e)
			return
		}
		out, e := h.rooms.MintRoomToken(r.Context(), agentPrincipal(r), ref)
		if e != nil {
			agentError(w, r, e)
			return
		}
		writeJSONResponse(w, 200, out)
	})
	return r
}

// The caller may correlate an event but cannot supply its principal or result.
func agentCorrelation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New()
		if raw := r.Header.Get("X-Agent-Call-ID"); raw != "" {
			parsed, e := uuid.Parse(raw)
			if e != nil || parsed == uuid.Nil || parsed.String() != raw {
				agentError(w, r, domain.ErrAgentInput)
				return
			}
			id = parsed
		}
		w.Header().Set("X-Agent-Call-ID", id.String())
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.AgentAuditCorrelationKey{}, id)))
	})
}
