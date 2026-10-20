package handler

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RoomTokenVerifier interface {
	VerifyRoomToken(string) (domain.SceneSessionIdentity, error)
	ParseSceneSession(string) (domain.SceneSessionIdentity, error)
}
type CapabilityHandler struct {
	capabilities  domain.CapabilityService
	rooms         RoomTokenVerifier
	presentations domain.PresentationAudienceAuthority
	access        domain.AccessService
	assets        domain.AssetUsecase
	features      domain.FeatureService
	serviceSecret []byte
	turnURL       string
	turnSecret    []byte
}

func (h *CapabilityHandler) SetPresentationAuthority(p domain.PresentationAudienceAuthority) {
	h.presentations = p
}

func NewCapabilityHandler(c domain.CapabilityService, r RoomTokenVerifier, a domain.AccessService, u domain.AssetUsecase, secret []byte) (*CapabilityHandler, error) {
	if c == nil || r == nil || len(secret) < 32 {
		return nil, errors.New("authority handler requires service, room verifier and >=32-byte service secret")
	}
	return &CapabilityHandler{capabilities: c, rooms: r, access: a, assets: u, serviceSecret: append([]byte(nil), secret...)}, nil
}
func capabilityError(w http.ResponseWriter, r *http.Request, e error) {
	if errors.Is(e, domain.ErrAssetConflict) {
		writeSharingError(w, r, http.StatusConflict, "ASSET_CONFLICT", "File ID already has incompatible image metadata")
		return
	}
	if errors.Is(e, domain.ErrInvalidCapability) {
		writeSharingError(w, r, http.StatusBadRequest, "INVALID_INPUT", "Invalid capability request")
		return
	}
	if errors.Is(e, domain.ErrFeatureDisabled) {
		writeFeatureError(w, r, e)
		return
	}
	sharingError(w, r, e)
}
func (h *CapabilityHandler) AuthorityHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /internal/authority/decisions", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get(domain.HeaderAuthorization)
		expected := domain.BearerPrefix + string(h.serviceSecret)
		if subtle.ConstantTimeCompare([]byte(raw), []byte(expected)) != 1 {
			sharingError(w, r, domain.ErrInvalidSceneSession)
			return
		}
		middleware.InvitationBodyMiddleware(http.HandlerFunc(h.decision)).ServeHTTP(w, r)
	}))
	mux.Handle("POST /internal/authority/features", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get(domain.HeaderAuthorization)
		expected := domain.BearerPrefix + string(h.serviceSecret)
		if subtle.ConstantTimeCompare([]byte(raw), []byte(expected)) != 1 {
			sharingError(w, r, domain.ErrInvalidSceneSession)
			return
		}
		middleware.InvitationBodyMiddleware(http.HandlerFunc(h.featureDecision)).ServeHTTP(w, r)
	}))
	return mux
}

func (h *CapabilityHandler) SetFeatureService(features domain.FeatureService) { h.features = features }

func (h *CapabilityHandler) featureDecision(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RoomToken   string `json:"roomToken" validate:"required,max=8192"`
		WorkspaceID string `json:"workspaceId" validate:"required,uuid"`
		SceneID     string `json:"sceneId" validate:"required,uuid"`
		Action      string `json:"action" validate:"required,max=64"`
	}
	if !decodeSharing(w, r, &in) {
		return
	}
	if in.Action != string(domain.FeatureScreenPublish) && in.Action != string(domain.FeaturePresentationHost) {
		capabilityError(w, r, domain.ErrInvalidCapability)
		return
	}
	identity, err := h.rooms.VerifyRoomToken(in.RoomToken)
	audience := false
	if err != nil && in.Action == string(domain.FeaturePresentationHost) && h.presentations != nil {
		var session uuid.UUID
		identity, session, err = h.presentations.ParseAudience(in.RoomToken)
		if err == nil {
			err = h.presentations.ResolveAudience(r.Context(), identity.Principal, identity.SceneRef, session)
			audience = err == nil
		}
	}
	if err != nil {
		capabilityError(w, r, err)
		return
	}
	if in.WorkspaceID != identity.SceneRef.WorkspaceID.String() || in.SceneID != identity.SceneRef.SceneID.String() {
		capabilityError(w, r, domain.ErrAccessDenied)
		return
	}
	if !audience && h.features == nil {
		writeFeatureError(w, r, errors.New("feature authority unavailable"))
		return
	}
	if !audience {
		action := domain.FeatureScreenPublish
		if in.Action == string(domain.FeaturePresentationHost) {
			action = domain.FeaturePresentationHost
		}
		if err = h.features.Require(r.Context(), identity.Principal, identity.SceneRef, action); err != nil {
			writeFeatureError(w, r, err)
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
func (h *CapabilityHandler) decision(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RoomToken string `json:"roomToken" validate:"required,max=8192"`
		domain.SceneMutation
	}
	if !decodeSharing(w, r, &in) {
		return
	}
	identity, e := h.rooms.VerifyRoomToken(in.RoomToken)
	if e != nil {
		capabilityError(w, r, e)
		return
	}
	if in.WorkspaceID != identity.SceneRef.WorkspaceID.String() || in.SceneID != identity.SceneRef.SceneID.String() {
		capabilityError(w, r, domain.ErrAccessDenied)
		return
	}
	result, e := h.capabilities.AuthorizeCommand(r.Context(), identity.Principal, in.SceneMutation)
	if e != nil {
		capabilityError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, result)
}
func (h *CapabilityHandler) identity(r *http.Request) (domain.SceneSessionIdentity, error) {
	raw := r.Header.Get(domain.HeaderAuthorization)
	if !strings.HasPrefix(raw, domain.BearerPrefix) {
		return domain.SceneSessionIdentity{}, domain.ErrInvalidSceneSession
	}
	raw = strings.TrimPrefix(raw, domain.BearerPrefix)
	identity, e := h.rooms.ParseSceneSession(raw)
	if e != nil {
		identity, e = h.rooms.VerifyRoomToken(raw)
	}
	if e != nil && h.presentations != nil {
		var session uuid.UUID
		identity, session, e = h.presentations.ParseAudience(raw)
		if e == nil {
			e = h.presentations.ResolveAudience(r.Context(), identity.Principal, identity.SceneRef, session)
		}
	}
	if e != nil {
		return identity, e
	}
	if raw := chi.URLParam(r, "id"); raw != "" {
		scene, e := h.access.ResolveSceneID(r.Context(), identity.SceneRef.WorkspaceID, raw)
		if e != nil || scene != identity.SceneRef.SceneID {
			return identity, domain.ErrInaccessible
		}
	}
	if raw := r.Header.Get(domain.HeaderWorkspaceID); raw != "" {
		w, e := h.access.ResolveWorkspaceID(r.Context(), raw)
		if e != nil || w != identity.SceneRef.WorkspaceID {
			return identity, domain.ErrInaccessible
		}
	}
	return identity, nil
}

func (h *CapabilityHandler) authorizeScene(ctx domain.Context, identity domain.SceneSessionIdentity, ttl time.Duration) (time.Time, error) {
	if !strings.HasPrefix(identity.Principal.ID, "presentation:") {
		return h.capabilities.AuthorizeScene(ctx, identity.Principal, identity.SceneRef, ttl)
	}
	if h.presentations == nil || !strings.HasPrefix(identity.Principal.GrantID, "session:") {
		return time.Time{}, domain.ErrAccessDenied
	}
	session, e := uuid.Parse(strings.TrimPrefix(identity.Principal.GrantID, "session:"))
	if e != nil || session == uuid.Nil {
		return time.Time{}, domain.ErrAccessDenied
	}
	if e = h.presentations.ResolveAudience(ctx, identity.Principal, identity.SceneRef, session); e != nil {
		return time.Time{}, e
	}
	expires := time.Now().Add(ttl)
	if identity.ExpiresAt.Before(expires) {
		expires = identity.ExpiresAt
	}
	return expires, nil
}
func (h *CapabilityHandler) PublicUpload(w http.ResponseWriter, r *http.Request) {
	identity, e := h.identity(r)
	if e != nil {
		capabilityError(w, r, e)
		return
	}
	var in struct {
		Assets []domain.AssetUploadRequest `json:"assets" validate:"required,min=1,max=100,dive"`
	}
	if !decodeCapability(w, r, &in) {
		return
	}
	result, e := h.capabilities.CreateAssetCapability(r.Context(), identity.Principal, identity.SceneRef, in.Assets)
	if e != nil {
		capabilityError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, result)
}
func decodeCapability(w http.ResponseWriter, r *http.Request, in any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if e := d.Decode(in); e != nil {
		capabilityError(w, r, domain.ErrInvalidCapability)
		return false
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF || validator.New().Struct(in) != nil {
		capabilityError(w, r, domain.ErrInvalidCapability)
		return false
	}
	return true
}
func (h *CapabilityHandler) PublicView(w http.ResponseWriter, r *http.Request) {
	identity, e := h.identity(r)
	if e != nil {
		capabilityError(w, r, e)
		return
	}
	var in struct {
		FileIDs []string `json:"fileIds" validate:"required,min=1,max=100,dive,required,max=128"`
	}
	if !decodeCapability(w, r, &in) {
		return
	}
	viewer, ok := h.assets.(domain.ScopedAssetViewer)
	if !ok {
		capabilityError(w, r, domain.ErrAccessDenied)
		return
	}
	var result map[string]domain.AssetViewResponse
	e = h.access.WithWorkspaceLock(r.Context(), identity.SceneRef.WorkspaceID, uuid.Nil, func(c domain.Context) error {
		expires, e := h.authorizeScene(c, identity, domain.AssetCapabilityLifetime)
		if e != nil {
			return e
		}
		result, e = viewer.PresignScopedView(c, identity.SceneRef, in.FileIDs, expires)
		return e
	})
	if e != nil {
		capabilityError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, map[string]any{"urls": result})
}
func (h *CapabilityHandler) SetTURNConfig(url, secret string) error {
	if url != "" && len(secret) < 32 {
		return errors.New("TURN_URL requires >=32-byte TURN_AUTH_SECRET")
	}
	h.turnURL = url
	h.turnSecret = []byte(secret)
	return nil
}
func (h *CapabilityHandler) RealtimeConfig(w http.ResponseWriter, r *http.Request) {
	identity, e := h.identity(r)
	if e != nil {
		capabilityError(w, r, e)
		return
	}
	expires, e := h.authorizeScene(r.Context(), identity, domain.RoomTokenLifetime)
	if e != nil {
		capabilityError(w, r, e)
		return
	}
	servers := []map[string]any{}
	if h.turnURL != "" {
		user := strconv.FormatInt(expires.Unix(), 10) + ":" + identity.Principal.ID
		mac := hmac.New(sha1.New, h.turnSecret)
		_, _ = mac.Write([]byte(user))
		servers = append(servers, map[string]any{"urls": []string{h.turnURL}, "username": user, "credential": base64.StdEncoding.EncodeToString(mac.Sum(nil))})
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, map[string]any{"iceServers": servers, "expiresAt": expires})
}

func (h *CapabilityHandler) UploadHandler(auth func(http.Handler) http.Handler) http.Handler {
	login := auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := middleware.UserPrincipalFromContext(r.Context())
		if !ok {
			capabilityError(w, r, domain.ErrInvalidSceneSession)
			return
		}
		workspace, e := h.access.ResolveWorkspaceID(r.Context(), r.Header.Get(domain.HeaderWorkspaceID))
		if e != nil {
			capabilityError(w, r, e)
			return
		}
		scene, e := h.access.ResolveSceneID(r.Context(), workspace, chi.URLParam(r, "id"))
		if e != nil {
			capabilityError(w, r, e)
			return
		}
		var in struct {
			Assets []domain.AssetUploadRequest `json:"assets" validate:"required,min=1,max=100,dive"`
		}
		if !decodeCapability(w, r, &in) {
			return
		}
		result, e := h.capabilities.CreateAssetCapability(r.Context(), p, domain.SceneRef{WorkspaceID: workspace, SceneID: scene}, in.Assets)
		if e != nil {
			capabilityError(w, r, e)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSONResponse(w, http.StatusOK, result)
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, e := h.identity(r); e == nil {
			h.PublicUpload(w, r)
			return
		}
		login.ServeHTTP(w, r)
	})
}
