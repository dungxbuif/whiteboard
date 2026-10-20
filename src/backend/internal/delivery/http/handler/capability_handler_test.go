package handler

import (
	"bytes"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type authorityUnavailable struct{ domain.CapabilityService }

func (authorityUnavailable) AuthorizeCommand(domain.Context, domain.Principal, domain.SceneMutation) (domain.SignedGrant, error) {
	return domain.SignedGrant{}, errors.New("authority offline")
}

type invalidRoom struct{}

func (invalidRoom) VerifyRoomToken(string) (domain.SceneSessionIdentity, error) {
	return domain.SceneSessionIdentity{}, domain.ErrInvalidSceneSession
}
func (invalidRoom) ParseSceneSession(string) (domain.SceneSessionIdentity, error) {
	return domain.SceneSessionIdentity{}, domain.ErrInvalidSceneSession
}
func TestAuthorityServiceAuthentication(t *testing.T) {
	h, e := NewCapabilityHandler(authorityUnavailable{}, invalidRoom{}, nil, nil, []byte(strings.Repeat("s", 32)))
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"/internal/authority/decisions", "/internal/authority/features"} {
		for _, secret := range []string{"", "wrong"} {
			r := httptest.NewRequest("POST", path, strings.NewReader(`{}`))
			r.Header.Set("Authorization", "Bearer "+secret)
			w := httptest.NewRecorder()
			h.AuthorityHandler().ServeHTTP(w, r)
			if w.Code != 401 {
				t.Fatalf("unauthenticated authority path=%s returned %d", path, w.Code)
			}
		}
	}
}

type validRoom struct{ identity domain.SceneSessionIdentity }

func (v validRoom) VerifyRoomToken(string) (domain.SceneSessionIdentity, error) {
	return v.identity, nil
}
func (v validRoom) ParseSceneSession(string) (domain.SceneSessionIdentity, error) {
	return v.identity, nil
}

type screenFeatureAuthority struct {
	domain.FeatureService
	err       error
	principal domain.Principal
	ref       domain.SceneRef
	action    domain.FeatureAction
}

type presentationAuthorityStub struct {
	identity domain.SceneSessionIdentity
	session  uuid.UUID
	resolved int
	err      error
}

func (s *presentationAuthorityStub) ParseAudience(string) (domain.SceneSessionIdentity, uuid.UUID, error) {
	return s.identity, s.session, nil
}
func (s *presentationAuthorityStub) ResolveAudience(domain.Context, domain.Principal, domain.SceneRef, uuid.UUID) error {
	s.resolved++
	return s.err
}

type capabilityPresentationAccessStub struct{ domain.AccessService }

func (capabilityPresentationAccessStub) WithWorkspaceLock(ctx domain.Context, _ uuid.UUID, _ uuid.UUID, fn func(domain.Context) error) error {
	return fn(ctx)
}

type presentationAssetViewer struct {
	domain.AssetUsecase
	called bool
}

func (s *presentationAssetViewer) PresignScopedView(domain.Context, domain.SceneRef, []string, time.Time) (map[string]domain.AssetViewResponse, error) {
	s.called = true
	return map[string]domain.AssetViewResponse{}, nil
}

func (s *screenFeatureAuthority) Require(_ domain.Context, principal domain.Principal, ref domain.SceneRef, action domain.FeatureAction) error {
	s.principal, s.ref, s.action = principal, ref, action
	return s.err
}
func TestAuthorityUnavailableHTTP(t *testing.T) {
	p := domain.UserPrincipal(uuid.New())
	p.CredentialExpiresAt = time.Now().Add(time.Minute)
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	h, _ := NewCapabilityHandler(authorityUnavailable{}, validRoom{domain.SceneSessionIdentity{Principal: p, SceneRef: ref}}, nil, nil, []byte(strings.Repeat("s", 32)))
	server := httptest.NewServer(h.AuthorityHandler())
	defer server.Close()
	body, _ := json.Marshal(map[string]any{"roomToken": "room", "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "action": "scene:update", "requestId": "request", "payloadHash": strings.Repeat("a", 64)})
	r, _ := http.NewRequest("POST", server.URL+"/internal/authority/decisions", bytes.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+strings.Repeat("s", 32))
	response, e := http.DefaultClient.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != 503 || bytes.Contains(raw, []byte(`"token"`)) {
		t.Fatalf("authority failure returned %d %s", response.StatusCode, raw)
	}
}

func TestScreenFeatureAuthorityFailsClosedWithoutFeatureService(t *testing.T) {
	principal := domain.UserPrincipal(uuid.New())
	principal.CredentialExpiresAt = time.Now().Add(time.Minute)
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	h, _ := NewCapabilityHandler(authorityUnavailable{}, validRoom{domain.SceneSessionIdentity{Principal: principal, SceneRef: ref}}, nil, nil, []byte(strings.Repeat("s", 32)))
	body, _ := json.Marshal(map[string]any{"roomToken": "room", "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "action": "screen:publish"})
	server := httptest.NewServer(h.AuthorityHandler())
	defer server.Close()
	r, _ := http.NewRequest("POST", server.URL+"/internal/authority/features", bytes.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+strings.Repeat("s", 32))
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusServiceUnavailable || bytes.Contains(raw, []byte(`"allowed"`)) {
		t.Fatalf("unbound feature authority returned %d %s", response.StatusCode, raw)
	}
}

func TestScreenFeatureAuthorityUsesCurrentFeaturePolicy(t *testing.T) {
	principal := domain.UserPrincipal(uuid.New())
	principal.CredentialExpiresAt = time.Now().Add(time.Minute)
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{name: "enabled", status: http.StatusNoContent},
		{name: "disabled", err: domain.ErrFeatureDisabled, status: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			features := &screenFeatureAuthority{err: tc.err}
			h, _ := NewCapabilityHandler(authorityUnavailable{}, validRoom{domain.SceneSessionIdentity{Principal: principal, SceneRef: ref}}, nil, nil, []byte(strings.Repeat("s", 32)))
			h.SetFeatureService(features)
			if h.features == nil {
				t.Fatal("feature service was not bound")
			}
			server := httptest.NewServer(h.AuthorityHandler())
			defer server.Close()
			body, _ := json.Marshal(map[string]any{"roomToken": "room", "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "action": "screen:publish"})
			r, _ := http.NewRequest("POST", server.URL+"/internal/authority/features", bytes.NewReader(body))
			r.Header.Set("Authorization", "Bearer "+strings.Repeat("s", 32))
			response, requestErr := http.DefaultClient.Do(r)
			if requestErr != nil {
				t.Fatal(requestErr)
			}
			defer response.Body.Close()
			raw, _ := io.ReadAll(response.Body)
			if response.StatusCode != tc.status || features.principal.ID != principal.ID || features.ref != ref || features.action != domain.FeatureScreenPublish {
				t.Fatalf("status=%d principal=%s ref=%+v action=%s body=%s", response.StatusCode, features.principal.ID, features.ref, features.action, raw)
			}
		})
	}
}

func TestPresentationFeatureAuthorityUsesPresentationPolicy(t *testing.T) {
	principal := domain.UserPrincipal(uuid.New())
	principal.CredentialExpiresAt = time.Now().Add(time.Minute)
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	features := &screenFeatureAuthority{}
	h, _ := NewCapabilityHandler(authorityUnavailable{}, validRoom{domain.SceneSessionIdentity{Principal: principal, SceneRef: ref}}, nil, nil, []byte(strings.Repeat("s", 32)))
	h.SetFeatureService(features)
	if h.features == nil {
		t.Fatal("feature service was not bound")
	}
	body, _ := json.Marshal(map[string]any{"roomToken": "room", "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "action": "presentation:host"})
	server := httptest.NewServer(h.AuthorityHandler())
	defer server.Close()
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/internal/authority/features", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("s", 32))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent || features.action != domain.FeaturePresentationHost {
		raw, _ := io.ReadAll(response.Body)
		t.Fatalf("status=%d action=%s body=%s", response.StatusCode, features.action, raw)
	}
}

func TestAdmittedPresentationAudienceCanReadScopedAssets(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	session := uuid.New()
	principal := domain.Principal{Kind: domain.PrincipalPublic, ID: "presentation:guest", GrantID: "session:" + session.String(), GrantVersion: "1", CredentialExpiresAt: time.Now().Add(time.Minute)}
	authority := &presentationAuthorityStub{identity: domain.SceneSessionIdentity{Principal: principal, SceneRef: ref, ExpiresAt: principal.CredentialExpiresAt}, session: session}
	assets := &presentationAssetViewer{}
	h, _ := NewCapabilityHandler(authorityUnavailable{}, invalidRoom{}, capabilityPresentationAccessStub{}, assets, []byte(strings.Repeat("s", 32)))
	h.SetPresentationAuthority(authority)
	request := httptest.NewRequest(http.MethodPost, "/public/scenes/scene/assets/presign-view", strings.NewReader(`{"fileIds":["image"]}`))
	request.Header.Set("Authorization", "Bearer audience")
	response := httptest.NewRecorder()
	h.PublicView(response, request)
	if response.Code != http.StatusOK || !assets.called || authority.resolved < 2 {
		t.Fatalf("status=%d assets=%t resolved=%d body=%s", response.Code, assets.called, authority.resolved, response.Body.String())
	}
}

func TestDisabledPresentationAudienceCannotReadScopedAssets(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	session := uuid.New()
	principal := domain.Principal{Kind: domain.PrincipalPublic, ID: "presentation:guest", GrantID: "session:" + session.String(), GrantVersion: "1", CredentialExpiresAt: time.Now().Add(time.Minute)}
	authority := &presentationAuthorityStub{identity: domain.SceneSessionIdentity{Principal: principal, SceneRef: ref, ExpiresAt: principal.CredentialExpiresAt}, session: session, err: domain.ErrFeatureDisabled}
	assets := &presentationAssetViewer{}
	h, _ := NewCapabilityHandler(authorityUnavailable{}, invalidRoom{}, capabilityPresentationAccessStub{}, assets, []byte(strings.Repeat("s", 32)))
	h.SetPresentationAuthority(authority)
	request := httptest.NewRequest(http.MethodPost, "/public/scenes/scene/assets/presign-view", strings.NewReader(`{"fileIds":["image"]}`))
	request.Header.Set("Authorization", "Bearer audience")
	response := httptest.NewRecorder()
	h.PublicView(response, request)
	if response.Code != http.StatusForbidden || assets.called {
		t.Fatalf("status=%d assets=%t body=%s", response.Code, assets.called, response.Body.String())
	}
}

func TestPresentationFeatureAuthorityAcceptsActiveAudienceCredential(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	session := uuid.New()
	principal := domain.Principal{Kind: domain.PrincipalPublic, ID: "presentation:guest", GrantID: "session:" + session.String(), GrantVersion: "1", CredentialExpiresAt: time.Now().Add(time.Minute)}
	authority := &presentationAuthorityStub{identity: domain.SceneSessionIdentity{Principal: principal, SceneRef: ref, ExpiresAt: principal.CredentialExpiresAt}, session: session}
	h, _ := NewCapabilityHandler(authorityUnavailable{}, invalidRoom{}, capabilityPresentationAccessStub{}, &presentationAssetViewer{}, []byte(strings.Repeat("s", 32)))
	h.SetPresentationAuthority(authority)
	body, _ := json.Marshal(map[string]any{"roomToken": "audience", "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "action": "presentation:host"})
	request := httptest.NewRequest(http.MethodPost, "/internal/authority/features", bytes.NewReader(body))
	response := httptest.NewRecorder()
	h.featureDecision(response, request)
	if response.Code != http.StatusNoContent || authority.resolved != 1 {
		t.Fatalf("status=%d resolved=%d body=%s", response.Code, authority.resolved, response.Body.String())
	}
}
