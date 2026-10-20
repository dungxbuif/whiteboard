package usecase

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"
)

type capabilityAccess struct {
	domain.AccessService
	decision domain.AccessDecision
	err      error
}

func (a capabilityAccess) Resolve(domain.Context, domain.Principal, domain.SceneRef) (domain.AccessDecision, error) {
	return a.decision, a.err
}
func (a capabilityAccess) WithWorkspaceLock(c domain.Context, _ uuid.UUID, _ uuid.UUID, f func(domain.Context) error) error {
	if a.err != nil {
		return a.err
	}
	return f(c)
}
func mutationFixture() (domain.Principal, domain.SceneMutation) {
	p := domain.UserPrincipal(uuid.New())
	p.CredentialExpiresAt = time.Now().Add(time.Minute)
	return p, domain.SceneMutation{WorkspaceID: uuid.NewString(), SceneID: uuid.NewString(), Action: domain.ActionSceneUpdate, RequestID: "request-1", PayloadHash: strings.Repeat("a", 64)}
}
func TestAuthorityUnavailable(t *testing.T) {
	p, m := mutationFixture()
	s, _ := NewCapabilityService(capabilityAccess{err: errors.New("database offline")}, nil, nil, []byte(strings.Repeat("g", 32)))
	g, e := s.AuthorizeCommand(context.Background(), p, m)
	if e == nil || g.Token != "" {
		t.Fatal("authority failure produced mutation grant")
	}
}
func TestAuthorityBoundGrant(t *testing.T) {
	p, m := mutationFixture()
	p.CredentialExpiresAt = time.Now().Add(3 * time.Second).Truncate(time.Second)
	s, _ := NewCapabilityService(capabilityAccess{decision: domain.AccessDecision{Access: domain.AccessEdit, ACLVersion: 42}}, nil, nil, []byte(strings.Repeat("g", 32)))
	g, e := s.AuthorizeCommand(context.Background(), p, m)
	if e != nil {
		t.Fatal(e)
	}
	c := &domain.MutationClaims{}
	_, e = jwt.ParseWithClaims(g.Token, c, func(*jwt.Token) (any, error) { return []byte(strings.Repeat("g", 32)), nil }, jwt.WithAudience("whiteboard-mutation"), jwt.WithIssuer("whiteboard-api"), jwt.WithValidMethods([]string{"HS256"}))
	if e != nil {
		t.Fatal(e)
	}
	if c.PrincipalID != p.ID || c.SceneMutation != m || c.ACLVersion != "42" || c.Purpose != "mutation-decision" || c.ExpiresAt.Time.After(p.CredentialExpiresAt) {
		t.Fatalf("unbound grant: %+v", c)
	}
}
func TestVersionCreateAuthorityBoundGrant(t *testing.T) {
	p, m := mutationFixture()
	m.Action, m.ExpectedSequence = domain.ActionVersionCreate, "42"
	s, _ := NewCapabilityService(capabilityAccess{decision: domain.AccessDecision{Access: domain.AccessEdit, ACLVersion: 7}}, nil, nil, []byte(strings.Repeat("g", 32)))
	g, err := s.AuthorizeCommand(context.Background(), p, m)
	if err != nil || g.Token == "" {
		t.Fatalf("version grant=%q err=%v", g.Token, err)
	}
}
func TestAuthorityExpiredViewerUnknownDenied(t *testing.T) {
	for _, kind := range []string{"expired", "viewer", "agent", "missing-expiry", "bad-hash"} {
		t.Run(kind, func(t *testing.T) {
			p, m := mutationFixture()
			d := domain.AccessDecision{Access: domain.AccessEdit, ACLVersion: 1}
			switch kind {
			case "expired":
				p.CredentialExpiresAt = time.Now().Add(-time.Second)
			case "viewer":
				d.Access = domain.AccessView
			case "agent":
				p.Kind = domain.PrincipalAgent
			case "missing-expiry":
				p.CredentialExpiresAt = time.Time{}
			case "bad-hash":
				m.PayloadHash = "bogus"
			}
			s, _ := NewCapabilityService(capabilityAccess{decision: d}, nil, nil, []byte(strings.Repeat("g", 32)))
			g, e := s.AuthorizeCommand(context.Background(), p, m)
			if e == nil || g.Token != "" {
				t.Fatal("invalid authority produced grant")
			}
		})
	}
}
