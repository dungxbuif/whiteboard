package usecase

import (
	"context"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"testing"
	"time"
)

type publicAccessRepo struct {
	accessStateRepo
	ref        domain.SceneRef
	grant      uuid.UUID
	access     string
	revoked    bool
	beforeLock func()
}

func (r *publicAccessRepo) PublicSceneState(ctx context.Context, p domain.Principal, ref domain.SceneRef) (domain.AccessDecision, error) {
	if ref != r.ref || p.GrantID != r.grant.String() || r.revoked {
		return domain.AccessDecision{}, domain.ErrInaccessible
	}
	return domain.AccessDecision{Access: r.access, ACLVersion: 42}, nil
}
func (r *publicAccessRepo) WithWorkspaceLock(ctx context.Context, w, u uuid.UUID, fn func(context.Context) error) error {
	if r.beforeLock != nil {
		r.beforeLock()
	}
	return fn(ctx)
}

type publicLinkRepo struct {
	domain.ShareLinkRepository
	link domain.ShareLink
}

func (r *publicLinkRepo) FindByHash(ctx context.Context, hash string) (*domain.ShareLink, error) {
	if hash != r.link.TokenHash {
		return nil, domain.ErrInaccessible
	}
	return &r.link, nil
}
func (r *publicLinkRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.ShareLink, error) {
	if id != r.link.ID {
		return nil, domain.ErrInaccessible
	}
	return &r.link, nil
}
func (r *publicLinkRepo) UserDisplayName(context.Context, uuid.UUID) (string, error) {
	return "Member", nil
}
func publicRoomFixture(t *testing.T) (*RoomService, *publicAccessRepo, *publicLinkRepo) {
	t.Helper()
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	grant := uuid.New()
	repo := &publicAccessRepo{ref: ref, grant: grant, access: "view"}
	access, e := NewAccessService(repo)
	if e != nil {
		t.Fatal(e)
	}
	links := &publicLinkRepo{link: domain.ShareLink{ID: grant, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, Active: true, Revision: 1, Access: "view", TokenHash: "2bb80d537b1da3e38bd30361aa855686bde0eacd7162fef6a25fe97bf527a25b"}}
	service, e := NewRoomService(links, access, []byte("0123456789abcdef0123456789abcdef"))
	if e != nil {
		t.Fatal(e)
	}
	return service, repo, links
}
func TestPublicSessionScope(t *testing.T) {
	service, repo, links := publicRoomFixture(t)
	ctx := context.Background()
	session, e := service.ExchangePublicSession(ctx, "secret", "", "")
	if e != nil {
		t.Fatalf("valid view exchange failed: %v", e)
	}
	identity, e := service.ParseSceneSession(session.Token)
	if e != nil {
		t.Fatal(e)
	}
	decision, e := service.access.Resolve(ctx, identity.Principal, repo.ref)
	if e != nil || decision.Access != "view" || decision.Owner {
		t.Fatalf("view grants writes: %+v %v", decision, e)
	}
	foreign := domain.SceneRef{WorkspaceID: repo.ref.WorkspaceID, SceneID: uuid.New()}
	if _, e = service.MintRoomToken(ctx, identity.Principal, foreign); e == nil {
		t.Fatal("guest selected another scene")
	}
	forged := identity.Principal
	user := uuid.New()
	forged.UserID = &user
	if _, e = service.access.Resolve(ctx, forged, repo.ref); !errors.Is(e, domain.ErrAccessDenied) {
		t.Fatal("public user-ID substitution accepted")
	}
	links.link.Access = "edit"
	repo.access = "edit"
	if _, e = service.ExchangePublicSession(ctx, "secret", "", "   "); !errors.Is(e, domain.ErrInvalidShareInput) {
		t.Fatalf("edit without display name: %v", e)
	}
	session, e = service.ExchangePublicSession(ctx, "secret", "", " Guest ")
	if e != nil || session.DisplayName != "Guest" {
		t.Fatalf("edit guest rejected: %+v %v", session, e)
	}
	identity, e = service.ParseSceneSession(session.Token)
	if e != nil {
		t.Fatal(e)
	}
	repo.revoked = true
	if _, e = service.MintRoomToken(ctx, identity.Principal, repo.ref); e == nil {
		t.Fatal("revoked link refreshed room token")
	}
}
func TestRoomTokenClaims(t *testing.T) {
	service, repo, _ := publicRoomFixture(t)
	session, e := service.ExchangePublicSession(context.Background(), "secret", "", "Reader")
	if e != nil {
		t.Fatal(e)
	}
	identity, e := service.ParseSceneSession(session.Token)
	if e != nil {
		t.Fatal(e)
	}
	room, e := service.MintRoomToken(context.Background(), identity.Principal, repo.ref)
	if e != nil {
		t.Fatal(e)
	}
	parsed, _, e := jwt.NewParser().ParseUnverified(room.Token, jwt.MapClaims{})
	if e != nil {
		t.Fatal(e)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["iss"] != "whiteboard-api" || claims["principalType"] != "public" || claims["grantId"] != repo.grant.String() || claims["aclVersion"] != "42" || claims["workspaceId"] != repo.ref.WorkspaceID.String() || claims["sceneId"] != repo.ref.SceneID.String() || claims["exp"].(float64)-claims["iat"].(float64) != 600 {
		t.Fatalf("wrong room claims: %+v", claims)
	}
	verified, e := service.VerifyRoomToken(room.Token)
	if e != nil || verified.ExpiresAt.Unix() != int64(claims["exp"].(float64)) || !verified.Principal.CredentialExpiresAt.Equal(verified.ExpiresAt) {
		t.Fatalf("room expiry not preserved: %+v %v", verified, e)
	}
	if _, e = service.ParseSceneSession(room.Token); e == nil {
		t.Fatal("room token accepted as scene session")
	}
	for _, change := range []struct {
		key   string
		value any
		alg   jwt.SigningMethod
	}{{"aud", "other", jwt.SigningMethodHS256}, {"iss", "other", jwt.SigningMethodHS256}, {"exp", time.Now().Add(-time.Minute).Unix(), jwt.SigningMethodHS256}, {"purpose", "scene-session", jwt.SigningMethodHS256}, {"alg", nil, jwt.SigningMethodHS384}, {"exp", nil, jwt.SigningMethodHS256}} {
		data, _ := json.Marshal(claims)
		copy := jwt.MapClaims{}
		json.Unmarshal(data, &copy)
		if change.key != "alg" {
			if change.value == nil {
				delete(copy, change.key)
			} else {
				copy[change.key] = change.value
			}
		}
		raw, e := jwt.NewWithClaims(change.alg, copy).SignedString(service.secret)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = service.VerifyRoomToken(raw); e == nil {
			t.Fatalf("invalid %s accepted", change.key)
		}
	}
}

func TestPublicSessionExpiryDuringAuthorityWait(t *testing.T) {
	service, repo, _ := publicRoomFixture(t)
	session, e := service.ExchangePublicSession(context.Background(), "secret", "", "Guest")
	if e != nil {
		t.Fatal(e)
	}
	token, _, e := jwt.NewParser().ParseUnverified(session.Token, jwt.MapClaims{})
	if e != nil {
		t.Fatal(e)
	}
	claims := token.Claims.(jwt.MapClaims)
	expires := time.Now().Add(2 * time.Second)
	claims["exp"] = expires.Unix()
	raw, e := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(service.secret)
	if e != nil {
		t.Fatal(e)
	}
	identity, e := service.ParseSceneSession(raw)
	if e != nil {
		t.Fatal(e)
	}
	repo.beforeLock = func() { time.Sleep(time.Until(expires) + 10*time.Millisecond) }
	if _, e = service.MintRoomToken(context.Background(), identity.Principal, repo.ref); !errors.Is(e, domain.ErrInvalidSceneSession) {
		t.Fatalf("expired session refreshed after authority wait: %v", e)
	}
}
