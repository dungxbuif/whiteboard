package usecase

import (
	"context"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type presentationRepoStub struct {
	link    *domain.PresentationLink
	rawHash string
	revoked bool
}

func (s *presentationRepoStub) CreateLink(_ domain.Context, link *domain.PresentationLink) error {
	s.link = link
	return nil
}
func (s *presentationRepoStub) FindLinkByHash(_ domain.Context, hash string) (*domain.PresentationLink, error) {
	s.rawHash = hash
	if s.link == nil {
		return nil, domain.ErrInaccessible
	}
	return s.link, nil
}
func (s *presentationRepoStub) FindLink(domain.Context, domain.SceneRef, uuid.UUID, uuid.UUID) (*domain.PresentationLink, error) {
	return s.link, nil
}
func (s *presentationRepoStub) RevokeLink(domain.Context, domain.SceneRef, uuid.UUID, uuid.UUID) error {
	s.revoked = true
	return nil
}

type presentationFeatures struct{}

func (presentationFeatures) Require(domain.Context, domain.Principal, domain.SceneRef, domain.FeatureAction) error {
	return nil
}

type presentationDisabledFeatures struct{ presentationFeatures }

func (presentationDisabledFeatures) Require(domain.Context, domain.Principal, domain.SceneRef, domain.FeatureAction) error {
	return domain.ErrFeatureDisabled
}

type presentationCaptureFeatures struct {
	presentationFeatures
	body []byte
}

func (s *presentationCaptureFeatures) Mutate(ctx domain.Context, _ domain.Principal, _ domain.SceneRef, _ domain.FeatureAction, _ uuid.UUID, _ []byte, fn func(domain.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	stored, err := fn(ctx)
	s.body = append([]byte(nil), stored.Body...)
	return stored, err
}
func (presentationFeatures) Policy(domain.Context, domain.Principal, uuid.UUID) (domain.WorkspaceFeaturePolicy, error) {
	return domain.WorkspaceFeaturePolicy{}, nil
}
func (presentationFeatures) UpdatePolicy(domain.Context, domain.Principal, uuid.UUID, int64, uuid.UUID, domain.WorkspaceFeaturePolicyPatch) (domain.WorkspaceFeaturePolicy, error) {
	return domain.WorkspaceFeaturePolicy{}, nil
}
func (presentationFeatures) Mutate(ctx domain.Context, _ domain.Principal, _ domain.SceneRef, _ domain.FeatureAction, _ uuid.UUID, _ []byte, fn func(domain.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	return fn(ctx)
}

type presentationRuntime struct {
	decision     domain.PresentationRuntimeDecision
	expectedHost string
	resolveHost  func(domain.SceneRef, uuid.UUID, string) (domain.PresentationRuntimeDecision, error)
}

func (s presentationRuntime) ResolveHost(_ domain.Context, ref domain.SceneRef, session uuid.UUID, principal string) (domain.PresentationRuntimeDecision, error) {
	if s.resolveHost != nil {
		return s.resolveHost(ref, session, principal)
	}
	if s.expectedHost != "" && principal != s.expectedHost {
		return domain.PresentationRuntimeDecision{}, domain.ErrPresentationEnded
	}
	return domain.PresentationRuntimeDecision{SessionID: session, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, HostPrincipalID: principal, Revision: 1}, nil
}

func (s presentationRuntime) ResolvePending(_ domain.Context, _ domain.SceneRef, _ uuid.UUID, host string, _ string, _ int64) (domain.PresentationRuntimeDecision, error) {
	if s.expectedHost != "" && host != s.expectedHost {
		return domain.PresentationRuntimeDecision{}, domain.ErrAccessDenied
	}
	return s.decision, nil
}
func (s presentationRuntime) ResolveAudience(_ domain.Context, ref domain.SceneRef, session uuid.UUID, principal string) (domain.PresentationRuntimeDecision, error) {
	return domain.PresentationRuntimeDecision{SessionID: session, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, PrincipalID: principal, HostPrincipalID: s.expectedHost}, nil
}

func TestPendingTokenCannotMintRoomOrReadAssets(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	session, link := uuid.New(), uuid.New()
	repo := &presentationRepoStub{link: &domain.PresentationLink{ID: link, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, SessionID: session, Revision: 1, Active: true, ExpiresAt: time.Now().Add(time.Hour)}}
	service, _ := NewPresentationService(repo, presentationFeatures{}, featureAccessStub{decision: domain.AccessDecision{Access: domain.AccessEdit, ACLVersion: 7}}, []byte(strings.Repeat("p", 32)))
	out, err := service.Join(context.Background(), "raw-secret", "Guest")
	if err != nil {
		t.Fatal(err)
	}
	claims := &presentationClaims{}
	token, err := jwt.ParseWithClaims(out.Token, claims, func(*jwt.Token) (any, error) { return []byte(strings.Repeat("p", 32)), nil }, jwt.WithAudience(domain.PresentationAudience), jwt.WithIssuer(domain.RoomIssuer))
	if err != nil || !token.Valid {
		t.Fatal(err)
	}
	if claims.Purpose != domain.PresentationPendingPurpose || claims.SessionID != session.String() || claims.Access != "pending" || claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) > domain.PresentationPendingLifetime || repo.rawHash == "raw-secret" || len(repo.rawHash) != 64 {
		t.Fatalf("claims=%+v hash=%q", claims, repo.rawHash)
	}
	room, _ := NewRoomService(nil, nil, []byte(strings.Repeat("p", 32)))
	if _, err = room.ParseSceneSession(out.Token); err == nil {
		t.Fatal("pending token parsed as scene session")
	}
	if _, err = room.VerifyRoomToken(out.Token); err == nil {
		t.Fatal("pending token parsed as a room credential")
	}
	if _, _, err = service.ParseAudience(out.Token); err == nil {
		t.Fatal("pending token parsed as an admitted audience credential")
	}
}

func TestPresentationLinkPersistsHashWithoutOneTimeSecret(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	session := uuid.New()
	principal := domain.UserPrincipal(uuid.New())
	repo := &presentationRepoStub{}
	features := &presentationCaptureFeatures{}
	service, _ := NewPresentationService(repo, features, featureAccessStub{}, []byte(strings.Repeat("p", 32)))
	service.SetRuntimeVerifier(presentationRuntime{})
	out, err := service.CreateLink(context.Background(), principal, ref, session, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if out.Token == "" || repo.link == nil || repo.link.TokenHash == "" || strings.Contains(string(features.body), out.Token) || strings.Contains(string(features.body), repo.link.TokenHash) {
		t.Fatalf("secret persistence boundary failed: token=%t hash=%q body=%s", out.Token != "", repo.link.TokenHash, features.body)
	}
}

func TestPresentationLinkRequiresCurrentHostAtTheSameRuntimeRevision(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	session, principal := uuid.New(), domain.UserPrincipal(uuid.New())
	repo := &presentationRepoStub{}
	calls := 0
	runtime := presentationRuntime{resolveHost: func(domain.SceneRef, uuid.UUID, string) (domain.PresentationRuntimeDecision, error) {
		calls++
		revision := int64(7)
		if calls > 1 {
			revision = 8
		}
		return domain.PresentationRuntimeDecision{SessionID: session, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, HostPrincipalID: principal.ID, Revision: revision}, nil
	}}
	service, _ := NewPresentationService(repo, presentationFeatures{}, featureAccessStub{}, []byte(strings.Repeat("p", 32)))
	service.SetRuntimeVerifier(runtime)
	if _, err := service.CreateLink(context.Background(), principal, ref, session, uuid.New()); !errors.Is(err, domain.ErrPresentationEnded) {
		t.Fatalf("link survived host transfer: %v", err)
	}
	if calls != 2 || repo.link != nil {
		t.Fatalf("calls=%d link=%+v", calls, repo.link)
	}
}

func TestPresentationLinkRejectsNonHostAndRevokeRechecksTransferRace(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	session, link, principal := uuid.New(), uuid.New(), domain.UserPrincipal(uuid.New())
	repo := &presentationRepoStub{}
	service, _ := NewPresentationService(repo, presentationFeatures{}, featureAccessStub{}, []byte(strings.Repeat("p", 32)))
	service.SetRuntimeVerifier(presentationRuntime{expectedHost: uuid.NewString()})
	if _, err := service.CreateLink(context.Background(), principal, ref, session, uuid.New()); !errors.Is(err, domain.ErrPresentationEnded) {
		t.Fatalf("non-host created link: %v", err)
	}
	calls := 0
	service.SetRuntimeVerifier(presentationRuntime{resolveHost: func(domain.SceneRef, uuid.UUID, string) (domain.PresentationRuntimeDecision, error) {
		calls++
		if calls == 1 {
			return domain.PresentationRuntimeDecision{SessionID: session, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, HostPrincipalID: principal.ID, Revision: 4}, nil
		}
		return domain.PresentationRuntimeDecision{}, domain.ErrPresentationEnded
	}})
	if err := service.RevokeLink(context.Background(), principal, ref, session, link, uuid.New()); !errors.Is(err, domain.ErrPresentationEnded) {
		t.Fatalf("revoke survived host transfer: %v", err)
	}
	if calls != 2 || repo.revoked {
		t.Fatalf("calls=%d revoked=%t", calls, repo.revoked)
	}
}

func TestPresentationHostCredentialIsNotPersistedInMutationReceipt(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	principal := domain.UserPrincipal(uuid.New())
	features := &presentationCaptureFeatures{}
	service, _ := NewPresentationService(&presentationRepoStub{}, features, featureAccessStub{decision: domain.AccessDecision{Access: domain.AccessEdit, ACLVersion: 3}}, []byte(strings.Repeat("p", 32)))
	out, err := service.Create(context.Background(), principal, ref, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if out.Token == "" || strings.Contains(string(features.body), out.Token) {
		t.Fatalf("host credential persistence boundary failed: token=%t body=%s", out.Token != "", features.body)
	}
}

func TestPresentationAdmitBindsRuntimeAudience(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	session := uuid.New()
	principal := domain.UserPrincipal(uuid.New())
	runtime := presentationRuntime{expectedHost: principal.ID, decision: domain.PresentationRuntimeDecision{SessionID: session, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, PrincipalID: "presentation:guest", DisplayName: "Guest"}}
	service, _ := NewPresentationService(&presentationRepoStub{}, presentationFeatures{}, featureAccessStub{decision: domain.AccessDecision{Access: domain.AccessEdit, ACLVersion: 9}}, []byte(strings.Repeat("p", 32)))
	service.SetRuntimeVerifier(runtime)
	out, err := service.Admit(context.Background(), principal, ref, session, "connection", 3)
	if err != nil {
		t.Fatal(err)
	}
	claims := &presentationClaims{}
	_, err = jwt.ParseWithClaims(out.Token, claims, func(*jwt.Token) (any, error) { return []byte(strings.Repeat("p", 32)), nil }, jwt.WithAudience(domain.PresentationAudience), jwt.WithIssuer(domain.RoomIssuer))
	if err != nil {
		t.Fatal(err)
	}
	if claims.Purpose != domain.PresentationViewerPurpose || claims.PrincipalID != "presentation:guest" || claims.SessionID != session.String() || claims.ACLVersion != "9" || claims.GrantVersion != "3" {
		body, _ := json.Marshal(claims)
		t.Fatalf("claims=%s", body)
	}
	identity, parsedSession, err := service.ParseAudience(out.Token)
	if err != nil || parsedSession != session || identity.SceneRef != ref || identity.Principal.ID != "presentation:guest" {
		t.Fatalf("identity=%+v session=%s err=%v", identity, parsedSession, err)
	}
}

func TestPresentationAudienceRechecksCurrentHostFeature(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	host := uuid.New()
	service, _ := NewPresentationService(&presentationRepoStub{}, presentationDisabledFeatures{}, featureAccessStub{}, []byte(strings.Repeat("p", 32)))
	service.SetRuntimeVerifier(presentationRuntime{expectedHost: host.String()})
	err := service.ResolveAudience(context.Background(), domain.Principal{Kind: domain.PrincipalPublic, ID: "presentation:guest"}, ref, uuid.New())
	if !errors.Is(err, domain.ErrFeatureDisabled) {
		t.Fatalf("audience remained authorized after presentation disable: %v", err)
	}
}
