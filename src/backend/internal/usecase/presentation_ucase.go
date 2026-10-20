package usecase

import (
	"crypto/rand"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type PresentationService struct {
	repo     domain.PresentationRepository
	features domain.FeatureService
	access   domain.AccessService
	secret   []byte
	runtime  domain.PresentationRuntimeVerifier
	now      func() time.Time
}

type presentationClaims struct {
	jwt.RegisteredClaims
	Purpose       string               `json:"purpose"`
	WorkspaceID   string               `json:"workspaceId"`
	SceneID       string               `json:"sceneId"`
	SessionID     string               `json:"sessionId"`
	PrincipalID   string               `json:"principalId"`
	PrincipalType domain.PrincipalKind `json:"principalType"`
	GrantID       string               `json:"grantId"`
	GrantVersion  string               `json:"grantVersion"`
	DisplayName   string               `json:"displayName"`
	Access        string               `json:"access"`
	ACLVersion    string               `json:"aclVersion"`
}

func NewPresentationService(repo domain.PresentationRepository, features domain.FeatureService, access domain.AccessService, secret []byte) (*PresentationService, error) {
	if repo == nil || features == nil || access == nil || len(secret) < 32 {
		return nil, errors.New("presentation service requires repository, authority and >=32-byte secret")
	}
	return &PresentationService{repo: repo, features: features, access: access, secret: append([]byte(nil), secret...), now: time.Now}, nil
}
func (s *PresentationService) SetRuntimeVerifier(runtime domain.PresentationRuntimeVerifier) {
	s.runtime = runtime
}

func presentationSecret() (string, error) {
	var b [32]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
func presentationHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *PresentationService) sign(purpose string, ref domain.SceneRef, session uuid.UUID, principal domain.Principal, grantID, grantVersion, access, aclVersion string, expires time.Time) (string, error) {
	now := s.now().UTC()
	claims := presentationClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: domain.RoomIssuer, Subject: principal.ID, Audience: jwt.ClaimStrings{domain.PresentationAudience}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires), ID: uuid.NewString()}, Purpose: purpose, WorkspaceID: ref.WorkspaceID.String(), SceneID: ref.SceneID.String(), SessionID: session.String(), PrincipalID: principal.ID, PrincipalType: principal.Kind, GrantID: grantID, GrantVersion: grantVersion, DisplayName: principal.DisplayName, Access: access, ACLVersion: aclVersion}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}
func presentationPrincipal(p domain.Principal) (uuid.UUID, error) {
	if p.Kind != domain.PrincipalUser || p.UserID == nil || *p.UserID == uuid.Nil {
		return uuid.Nil, domain.ErrAccessDenied
	}
	return *p.UserID, nil
}

func (s *PresentationService) currentHost(ctx domain.Context, p domain.Principal, ref domain.SceneRef, sessionID uuid.UUID) (domain.PresentationRuntimeDecision, error) {
	if s.runtime == nil {
		return domain.PresentationRuntimeDecision{}, domain.ErrPresentationEnded
	}
	decision, e := s.runtime.ResolveHost(ctx, ref, sessionID, p.ID)
	if e != nil {
		return domain.PresentationRuntimeDecision{}, e
	}
	if decision.SessionID != sessionID || decision.WorkspaceID != ref.WorkspaceID || decision.SceneID != ref.SceneID || decision.HostPrincipalID != p.ID || decision.Revision < 1 {
		return domain.PresentationRuntimeDecision{}, domain.ErrPresentationEnded
	}
	return decision, nil
}

func (s *PresentationService) Create(ctx domain.Context, p domain.Principal, ref domain.SceneRef, requestID uuid.UUID) (domain.SessionGrant, error) {
	user, e := presentationPrincipal(p)
	if e != nil || requestID == uuid.Nil {
		return domain.SessionGrant{}, domain.ErrInvalidPresentation
	}
	payload, _ := json.Marshal(map[string]string{"operation": "create", "sceneId": ref.SceneID.String()})
	var raw string
	stored, e := s.features.Mutate(ctx, p, ref, domain.FeaturePresentationHost, requestID, payload, func(tx domain.Context) (domain.StoredFeatureResponse, error) {
		decision, err := s.access.Resolve(tx, p, ref)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		session := uuid.New()
		expires := s.now().UTC().Add(domain.RoomTokenLifetime)
		principal := domain.UserPrincipal(user)
		principal.DisplayName = p.DisplayName
		token, err := s.sign(domain.PresentationHostPurpose, ref, session, principal, "membership:"+ref.WorkspaceID.String()+":"+user.String(), "", "edit", strconv.FormatInt(decision.ACLVersion, 10), expires)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		raw = token
		body, err := json.Marshal(struct {
			SessionID uuid.UUID `json:"sessionId"`
			ExpiresAt time.Time `json:"expiresAt"`
		}{SessionID: session, ExpiresAt: expires})
		return domain.StoredFeatureResponse{Status: 201, Body: body}, err
	})
	if e != nil {
		return domain.SessionGrant{}, e
	}
	if raw == "" {
		return domain.SessionGrant{}, domain.ErrRequestIDReused
	}
	var result domain.SessionGrant
	if json.Unmarshal(stored.Body, &result) != nil {
		return result, domain.ErrInvalidPresentation
	}
	result.Token = raw
	return result, nil
}

func (s *PresentationService) CreateLink(ctx domain.Context, p domain.Principal, ref domain.SceneRef, sessionID, requestID uuid.UUID) (domain.OneTimeShownLink, error) {
	user, e := presentationPrincipal(p)
	if e != nil || sessionID == uuid.Nil || requestID == uuid.Nil {
		return domain.OneTimeShownLink{}, domain.ErrInvalidPresentation
	}
	host, e := s.currentHost(ctx, p, ref, sessionID)
	if e != nil {
		return domain.OneTimeShownLink{}, e
	}
	payload, _ := json.Marshal(map[string]string{"operation": "create-link", "sessionId": sessionID.String()})
	var raw string
	stored, e := s.features.Mutate(ctx, p, ref, domain.FeaturePresentationHost, requestID, payload, func(tx domain.Context) (domain.StoredFeatureResponse, error) {
		secret, err := presentationSecret()
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		link := domain.PresentationLink{ID: uuid.New(), WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, SessionID: sessionID, TokenHash: presentationHash(secret), CreatedBy: user, Revision: 1, Active: true, CreatedAt: s.now().UTC(), ExpiresAt: s.now().UTC().Add(domain.PresentationLinkLifetime)}
		current, err := s.currentHost(tx, p, ref, sessionID)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		if current.Revision != host.Revision {
			return domain.StoredFeatureResponse{}, domain.ErrPresentationEnded
		}
		if err = s.repo.CreateLink(tx, &link); err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		raw = secret
		body, err := json.Marshal(link)
		return domain.StoredFeatureResponse{Status: 201, Body: body}, err
	})
	if e != nil {
		return domain.OneTimeShownLink{}, e
	}
	if raw == "" {
		return domain.OneTimeShownLink{}, domain.ErrRequestIDReused
	}
	var link domain.PresentationLink
	if json.Unmarshal(stored.Body, &link) != nil {
		return domain.OneTimeShownLink{}, domain.ErrInvalidPresentation
	}
	return domain.OneTimeShownLink{PresentationLink: link, Token: raw}, nil
}

func (s *PresentationService) Join(ctx domain.Context, token, name string) (domain.PendingPresentationGrant, error) {
	if token == "" || len(token) > 256 {
		return domain.PendingPresentationGrant{}, domain.ErrInvalidPresentation
	}
	name = strings.TrimSpace(name)
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > domain.MaxPublicDisplayName {
		return domain.PendingPresentationGrant{}, domain.ErrInvalidPresentation
	}
	link, e := s.repo.FindLinkByHash(ctx, presentationHash(token))
	if e != nil {
		return domain.PendingPresentationGrant{}, e
	}
	if !link.Active || !s.now().Before(link.ExpiresAt) {
		return domain.PendingPresentationGrant{}, domain.ErrInaccessible
	}
	secret, e := presentationSecret()
	if e != nil {
		return domain.PendingPresentationGrant{}, e
	}
	principal := domain.Principal{Kind: domain.PrincipalPublic, ID: "presentation:" + secret, DisplayName: name}
	ref := domain.SceneRef{WorkspaceID: link.WorkspaceID, SceneID: link.SceneID}
	expires := s.now().UTC().Add(domain.PresentationPendingLifetime)
	raw, e := s.sign(domain.PresentationPendingPurpose, ref, link.SessionID, principal, link.ID.String(), strconv.FormatInt(link.Revision, 10), "pending", "0", expires)
	if e != nil {
		return domain.PendingPresentationGrant{}, e
	}
	return domain.PendingPresentationGrant{SessionID: link.SessionID, WorkspaceID: link.WorkspaceID, SceneID: link.SceneID, PrincipalID: principal.ID, DisplayName: name, Token: raw, ExpiresAt: expires}, nil
}

func (s *PresentationService) Admit(ctx domain.Context, p domain.Principal, ref domain.SceneRef, sessionID uuid.UUID, connectionID string, revision int64) (domain.SessionGrant, error) {
	if _, e := presentationPrincipal(p); e != nil || sessionID == uuid.Nil || connectionID == "" || len(connectionID) > 128 || revision < 1 {
		return domain.SessionGrant{}, domain.ErrInvalidPresentation
	}
	if s.runtime == nil {
		return domain.SessionGrant{}, domain.ErrPresentationEnded
	}
	if e := s.features.Require(ctx, p, ref, domain.FeaturePresentationHost); e != nil {
		return domain.SessionGrant{}, e
	}
	decision, e := s.runtime.ResolvePending(ctx, ref, sessionID, p.ID, connectionID, revision)
	if e != nil {
		return domain.SessionGrant{}, e
	}
	if decision.SessionID != sessionID || decision.WorkspaceID != ref.WorkspaceID || decision.SceneID != ref.SceneID {
		return domain.SessionGrant{}, domain.ErrAccessDenied
	}
	principal := domain.Principal{Kind: domain.PrincipalPublic, ID: decision.PrincipalID, DisplayName: decision.DisplayName}
	accessDecision, e := s.access.Resolve(ctx, p, ref)
	if e != nil {
		return domain.SessionGrant{}, e
	}
	expires := s.now().UTC().Add(domain.RoomTokenLifetime)
	token, e := s.sign(domain.PresentationViewerPurpose, ref, sessionID, principal, "session:"+sessionID.String(), strconv.FormatInt(revision, 10), "view", strconv.FormatInt(accessDecision.ACLVersion, 10), expires)
	if e != nil {
		return domain.SessionGrant{}, e
	}
	return domain.SessionGrant{SessionID: sessionID, Token: token, ExpiresAt: expires}, nil
}

func (s *PresentationService) ResolveAudience(ctx domain.Context, p domain.Principal, ref domain.SceneRef, sessionID uuid.UUID) error {
	if p.Kind != domain.PrincipalPublic || !strings.HasPrefix(p.ID, "presentation:") || sessionID == uuid.Nil || s.runtime == nil {
		return domain.ErrAccessDenied
	}
	decision, e := s.runtime.ResolveAudience(ctx, ref, sessionID, p.ID)
	if e != nil {
		return e
	}
	hostID, e := uuid.Parse(decision.HostPrincipalID)
	if e != nil || hostID == uuid.Nil {
		return domain.ErrAccessDenied
	}
	return s.features.Require(ctx, domain.UserPrincipal(hostID), ref, domain.FeaturePresentationHost)
}

func (s *PresentationService) ParseAudience(raw string) (domain.SceneSessionIdentity, uuid.UUID, error) {
	var identity domain.SceneSessionIdentity
	claims := &presentationClaims{}
	token, e := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return s.secret, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(domain.RoomIssuer), jwt.WithAudience(domain.PresentationAudience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if e != nil || !token.Valid || claims.Purpose != domain.PresentationViewerPurpose || claims.IssuedAt == nil || claims.ExpiresAt == nil || claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) > domain.RoomTokenLifetime || len(claims.Audience) != 1 || claims.Subject != claims.PrincipalID || claims.ID == "" || claims.PrincipalType != domain.PrincipalPublic || !strings.HasPrefix(claims.PrincipalID, "presentation:") || claims.Access != domain.AccessView {
		return identity, uuid.Nil, domain.ErrInvalidSceneSession
	}
	workspace, we := uuid.Parse(claims.WorkspaceID)
	scene, se := uuid.Parse(claims.SceneID)
	session, pe := uuid.Parse(claims.SessionID)
	grantVersion, ge := strconv.ParseInt(claims.GrantVersion, 10, 64)
	aclVersion, ae := strconv.ParseInt(claims.ACLVersion, 10, 64)
	if we != nil || se != nil || pe != nil || ge != nil || ae != nil || workspace == uuid.Nil || scene == uuid.Nil || session == uuid.Nil || grantVersion < 1 || aclVersion < 1 || claims.GrantVersion != strconv.FormatInt(grantVersion, 10) || claims.ACLVersion != strconv.FormatInt(aclVersion, 10) || claims.GrantID != "session:"+session.String() || strings.TrimSpace(claims.DisplayName) == "" || utf8.RuneCountInString(claims.DisplayName) > domain.MaxPublicDisplayName {
		return identity, uuid.Nil, domain.ErrInvalidSceneSession
	}
	principal := domain.Principal{Kind: domain.PrincipalPublic, ID: claims.PrincipalID, GrantID: claims.GrantID, GrantVersion: claims.GrantVersion, DisplayName: claims.DisplayName, CredentialExpiresAt: claims.ExpiresAt.Time}
	return domain.SceneSessionIdentity{Principal: principal, SceneRef: domain.SceneRef{WorkspaceID: workspace, SceneID: scene}, ExpiresAt: claims.ExpiresAt.Time}, session, nil
}

func (s *PresentationService) RevokeLink(ctx domain.Context, p domain.Principal, ref domain.SceneRef, sessionID, linkID, requestID uuid.UUID) error {
	if _, e := presentationPrincipal(p); e != nil || sessionID == uuid.Nil || linkID == uuid.Nil || requestID == uuid.Nil {
		return domain.ErrInvalidPresentation
	}
	host, e := s.currentHost(ctx, p, ref, sessionID)
	if e != nil {
		return e
	}
	payload, _ := json.Marshal(map[string]string{"operation": "revoke-link", "sessionId": sessionID.String(), "linkId": linkID.String()})
	_, e = s.features.Mutate(ctx, p, ref, domain.FeaturePresentationHost, requestID, payload, func(tx domain.Context) (domain.StoredFeatureResponse, error) {
		current, err := s.currentHost(tx, p, ref, sessionID)
		if err != nil {
			return domain.StoredFeatureResponse{}, err
		}
		if current.Revision != host.Revision {
			return domain.StoredFeatureResponse{}, domain.ErrPresentationEnded
		}
		if e := s.repo.RevokeLink(tx, ref, sessionID, linkID); e != nil {
			return domain.StoredFeatureResponse{}, e
		}
		return domain.StoredFeatureResponse{Status: 204, Body: json.RawMessage(`{}`)}, nil
	})
	return e
}

var _ domain.PresentationUsecase = (*PresentationService)(nil)
var _ domain.PresentationAudienceAuthority = (*PresentationService)(nil)
