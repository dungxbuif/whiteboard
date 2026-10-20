package usecase

import (
	"context"
	"crypto/sha256"
	"devtool/internal/domain"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type domainRepoStub struct {
	values   map[uuid.UUID]domain.WorkspaceDomain
	byName   map[string]uuid.UUID
	requests map[uuid.UUID]struct{}
}

func newDomainRepoStub() *domainRepoStub {
	return &domainRepoStub{values: map[uuid.UUID]domain.WorkspaceDomain{}, byName: map[string]uuid.UUID{}, requests: map[uuid.UUID]struct{}{}}
}
func (r *domainRepoStub) ClaimRequest(_ domain.Context, _ uuid.UUID, id uuid.UUID, _, _ string) error {
	if _, ok := r.requests[id]; ok {
		return domain.ErrRequestIDReused
	}
	r.requests[id] = struct{}{}
	return nil
}
func (r *domainRepoStub) Create(_ domain.Context, v domain.WorkspaceDomain) (domain.WorkspaceDomain, error) {
	if id, ok := r.byName[v.Domain]; ok {
		existing := r.values[id]
		if existing.WorkspaceID != v.WorkspaceID {
			if existing.Status != domain.DomainPending || v.CreatedAt.Before(existing.ChallengeExpiresAt) {
				return domain.WorkspaceDomain{}, domain.ErrDomainConflict
			}
			delete(r.values, id)
			r.values[v.ID] = v
			r.byName[v.Domain] = v.ID
			return v, nil
		}
		v.ID = existing.ID
		v.Revision = existing.Revision + 1
		r.values[id] = v
		return v, nil
	}
	r.values[v.ID] = v
	r.byName[v.Domain] = v.ID
	return v, nil
}
func (r *domainRepoStub) Get(_ domain.Context, w, id uuid.UUID) (domain.WorkspaceDomain, error) {
	v, ok := r.values[id]
	if !ok || v.WorkspaceID != w {
		return domain.WorkspaceDomain{}, domain.ErrInaccessible
	}
	return v, nil
}
func (r *domainRepoStub) List(_ domain.Context, w uuid.UUID) ([]domain.WorkspaceDomain, error) {
	out := []domain.WorkspaceDomain{}
	for _, v := range r.values {
		if v.WorkspaceID == w {
			out = append(out, v)
		}
	}
	return out, nil
}
func (r *domainRepoStub) Discover(_ domain.Context, name string) (domain.DomainDiscovery, error) {
	id, ok := r.byName[name]
	if !ok || r.values[id].Status != domain.DomainCaptured {
		return domain.DomainDiscovery{}, domain.ErrInaccessible
	}
	return domain.DomainDiscovery{Domain: name, Workspace: r.values[id].WorkspaceID.String()}, nil
}
func (r *domainRepoStub) Transition(_ domain.Context, w, id uuid.UUID, rev int64, status domain.DomainStatus, now time.Time) (domain.WorkspaceDomain, error) {
	v, e := r.Get(context.Background(), w, id)
	if e != nil {
		return v, e
	}
	if v.Revision != rev {
		return v, domain.ErrRevisionConflict
	}
	v.Status = status
	v.Revision++
	v.UpdatedAt = now
	if status == domain.DomainVerified {
		v.VerifiedAt = &now
	}
	if status == domain.DomainCaptured {
		v.CapturedAt = &now
	}
	r.values[id] = v
	return v, nil
}

type resolverStub struct {
	records map[string][]string
	err     error
	names   []string
}

func (r *resolverStub) LookupTXT(_ domain.Context, name string) ([]string, error) {
	r.names = append(r.names, name)
	return r.records[name], r.err
}

type accessStub struct {
	owner       uuid.UUID
	denied      bool
	checks      *int
	failOnCheck int
}

func (a accessStub) RequireOwner(_ domain.Context, u, w uuid.UUID) error {
	if a.checks != nil {
		*a.checks++
	}
	if a.denied || u != a.owner || (a.checks != nil && a.failOnCheck > 0 && *a.checks == a.failOnCheck) {
		return domain.ErrAccessDenied
	}
	return nil
}
func (a accessStub) WithWorkspaceLock(ctx domain.Context, _ uuid.UUID, _ uuid.UUID, fn func(domain.Context) error) error {
	return fn(ctx)
}

func TestDomainExactIDNAChallenge(t *testing.T) {
	now := time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC)
	owner := uuid.New()
	workspace := uuid.New()
	repo := newDomainRepoStub()
	dns := &resolverStub{records: map[string][]string{}}
	s := NewDomainService(repo, accessStub{owner: owner}, dns, WithDomainClock(func() time.Time { return now }), WithDomainRandom(func(p []byte) (int, error) {
		for i := range p {
			p[i] = byte(i)
		}
		return len(p), nil
	}))
	challenge, err := s.Begin(context.Background(), domain.UserPrincipal(owner), workspace, " BÜCHER.Example. ", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if challenge.Domain != "xn--bcher-kva.example" {
		t.Fatalf("normalized domain=%q", challenge.Domain)
	}
	if challenge.DNSName != "_whiteboard-verification.xn--bcher-kva.example" {
		t.Fatalf("dns name=%q", challenge.DNSName)
	}
	if challenge.TXTValue == "" || len(challenge.ChallengeHash) != sha256.Size {
		t.Fatal("missing one-time challenge or digest")
	}
	if got := challenge.ChallengeExpiresAt.Sub(now); got != 24*time.Hour {
		t.Fatalf("expiry=%v", got)
	}
	dns.records[challenge.DNSName] = []string{challenge.TXTValue}
	verified, err := s.Verify(context.Background(), domain.UserPrincipal(owner), workspace, challenge.ID, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if verified.Status != domain.DomainVerified || len(dns.names) != 1 || dns.names[0] != challenge.DNSName {
		t.Fatalf("verification=%+v lookups=%v", verified, dns.names)
	}
}

func TestDomainExpiredReplayAndConflict(t *testing.T) {
	now := time.Now().UTC()
	owner := uuid.New()
	w1, w2 := uuid.New(), uuid.New()
	repo := newDomainRepoStub()
	dns := &resolverStub{records: map[string][]string{}}
	s := NewDomainService(repo, accessStub{owner: owner}, dns, WithDomainClock(func() time.Time { return now }))
	c, err := s.Begin(context.Background(), domain.UserPrincipal(owner), w1, "example.com", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Begin(context.Background(), domain.UserPrincipal(owner), w2, "EXAMPLE.COM.", uuid.New()); !errors.Is(err, domain.ErrDomainConflict) {
		t.Fatalf("conflict=%v", err)
	}
	now = now.Add(24*time.Hour + time.Nanosecond)
	dns.records[c.DNSName] = []string{c.TXTValue}
	if _, err = s.Verify(context.Background(), domain.UserPrincipal(owner), w1, c.ID, uuid.New()); !errors.Is(err, domain.ErrDomainChallengeExpired) {
		t.Fatalf("expired=%v", err)
	}
}

func TestDomainParentAndSubdomainProofAreSeparate(t *testing.T) {
	now := time.Now().UTC()
	owner, workspace := uuid.New(), uuid.New()
	repo := newDomainRepoStub()
	dns := &resolverStub{records: map[string][]string{}}
	service := NewDomainService(repo, accessStub{owner: owner}, dns, WithDomainClock(func() time.Time { return now }))
	challenge, err := service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "team.example.com", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	dns.records["_whiteboard-verification.example.com"] = []string{challenge.TXTValue}
	if _, err = service.Verify(context.Background(), domain.UserPrincipal(owner), workspace, challenge.ID, uuid.New()); !errors.Is(err, domain.ErrDomainChallengeFailed) {
		t.Fatalf("parent proof error=%v", err)
	}
	if len(dns.names) != 1 || dns.names[0] != "_whiteboard-verification.team.example.com" {
		t.Fatalf("lookups=%v", dns.names)
	}
}

func TestDomainOwnerCanIssueFreshChallengeForSameClaim(t *testing.T) {
	now := time.Now().UTC()
	owner, workspace := uuid.New(), uuid.New()
	repo := newDomainRepoStub()
	service := NewDomainService(repo, accessStub{owner: owner}, &resolverStub{records: map[string][]string{}}, WithDomainClock(func() time.Time { return now }))
	first, err := service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "example.com", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	second, err := service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "EXAMPLE.COM.", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.Revision != first.Revision+1 || second.TXTValue == first.TXTValue {
		t.Fatalf("fresh challenge first=%+v second=%+v", first, second)
	}
}

func TestCaptureDoesNotMoveExistingData(t *testing.T) {
	now := time.Now().UTC()
	owner := uuid.New()
	w := uuid.New()
	repo := newDomainRepoStub()
	dns := &resolverStub{records: map[string][]string{}}
	s := NewDomainService(repo, accessStub{owner: owner}, dns, WithDomainClock(func() time.Time { return now }))
	c, _ := s.Begin(context.Background(), domain.UserPrincipal(owner), w, "example.com", uuid.New())
	dns.records[c.DNSName] = []string{c.TXTValue}
	v, _ := s.Verify(context.Background(), domain.UserPrincipal(owner), w, c.ID, uuid.New())
	result, err := s.Capture(context.Background(), domain.UserPrincipal(owner), w, v.ID, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if result.DataMoved || result.Domain.Status != domain.DomainCaptured {
		t.Fatalf("capture=%+v", result)
	}
	if _, err = s.Verify(context.Background(), domain.UserPrincipal(owner), w, c.ID, uuid.New()); !errors.Is(err, domain.ErrDomainChallengeExpired) {
		t.Fatalf("consumed challenge=%v", err)
	}
}

func TestCaptureRequiresCurrentUnexpiredExactTXTProof(t *testing.T) {
	now := time.Now().UTC()
	owner, workspace := uuid.New(), uuid.New()
	repo := newDomainRepoStub()
	dns := &resolverStub{records: map[string][]string{}}
	service := NewDomainService(repo, accessStub{owner: owner}, dns, WithDomainClock(func() time.Time { return now }))
	challenge, _ := service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "example.com", uuid.New())
	dns.records[challenge.DNSName] = []string{challenge.TXTValue}
	verified, _ := service.Verify(context.Background(), domain.UserPrincipal(owner), workspace, challenge.ID, uuid.New())
	delete(dns.records, challenge.DNSName)
	if _, err := service.Capture(context.Background(), domain.UserPrincipal(owner), workspace, verified.ID, uuid.New()); !errors.Is(err, domain.ErrDomainChallengeFailed) {
		t.Fatalf("removed proof capture=%v", err)
	}
	current, _ := repo.Get(context.Background(), workspace, verified.ID)
	if current.Status != domain.DomainVerified {
		t.Fatalf("failed capture changed status=%s", current.Status)
	}
	dns.records[challenge.DNSName] = []string{challenge.TXTValue}
	now = challenge.ChallengeExpiresAt
	if _, err := service.Capture(context.Background(), domain.UserPrincipal(owner), workspace, verified.ID, uuid.New()); !errors.Is(err, domain.ErrDomainChallengeExpired) {
		t.Fatalf("expired proof capture=%v", err)
	}
}

func TestExpiredPendingDomainCanBeReclaimedButVerifiedDomainCannot(t *testing.T) {
	now := time.Now().UTC()
	owner := uuid.New()
	firstWorkspace, secondWorkspace, thirdWorkspace := uuid.New(), uuid.New(), uuid.New()
	repo := newDomainRepoStub()
	dns := &resolverStub{records: map[string][]string{}}
	service := NewDomainService(repo, accessStub{owner: owner}, dns, WithDomainClock(func() time.Time { return now }))
	first, _ := service.Begin(context.Background(), domain.UserPrincipal(owner), firstWorkspace, "example.com", uuid.New())
	now = first.ChallengeExpiresAt
	second, err := service.Begin(context.Background(), domain.UserPrincipal(owner), secondWorkspace, "EXAMPLE.COM.", uuid.New())
	if err != nil {
		t.Fatalf("expired reclaim=%v", err)
	}
	if second.WorkspaceID != secondWorkspace || second.ID == first.ID {
		t.Fatalf("reclaim=%+v", second)
	}
	dns.records[second.DNSName] = []string{second.TXTValue}
	if _, err = service.Verify(context.Background(), domain.UserPrincipal(owner), secondWorkspace, second.ID, uuid.New()); err != nil {
		t.Fatal(err)
	}
	now = second.ChallengeExpiresAt.Add(time.Hour)
	if _, err = service.Begin(context.Background(), domain.UserPrincipal(owner), thirdWorkspace, "example.com", uuid.New()); !errors.Is(err, domain.ErrDomainConflict) {
		t.Fatalf("verified takeover=%v", err)
	}
}

func TestDomainMutationsRecheckOwnerInsideWorkspaceLock(t *testing.T) {
	owner, workspace := uuid.New(), uuid.New()
	tests := []struct {
		name    string
		prepare func(*DomainService, *domainRepoStub, *resolverStub) func() error
	}{
		{name: "begin", prepare: func(service *DomainService, _ *domainRepoStub, _ *resolverStub) func() error {
			return func() error {
				_, err := service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "example.com", uuid.New())
				return err
			}
		}},
		{name: "verify", prepare: func(service *DomainService, _ *domainRepoStub, dns *resolverStub) func() error {
			challenge, _ := service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "example.com", uuid.New())
			dns.records[challenge.DNSName] = []string{challenge.TXTValue}
			return func() error {
				_, err := service.Verify(context.Background(), domain.UserPrincipal(owner), workspace, challenge.ID, uuid.New())
				return err
			}
		}},
		{name: "capture", prepare: func(service *DomainService, _ *domainRepoStub, dns *resolverStub) func() error {
			challenge, _ := service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "example.com", uuid.New())
			dns.records[challenge.DNSName] = []string{challenge.TXTValue}
			verified, _ := service.Verify(context.Background(), domain.UserPrincipal(owner), workspace, challenge.ID, uuid.New())
			return func() error {
				_, err := service.Capture(context.Background(), domain.UserPrincipal(owner), workspace, verified.ID, uuid.New())
				return err
			}
		}},
		{name: "revoke", prepare: func(service *DomainService, _ *domainRepoStub, dns *resolverStub) func() error {
			challenge, _ := service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "example.com", uuid.New())
			dns.records[challenge.DNSName] = []string{challenge.TXTValue}
			verified, _ := service.Verify(context.Background(), domain.UserPrincipal(owner), workspace, challenge.ID, uuid.New())
			captured, _ := service.Capture(context.Background(), domain.UserPrincipal(owner), workspace, verified.ID, uuid.New())
			return func() error {
				_, err := service.Revoke(context.Background(), domain.UserPrincipal(owner), workspace, captured.Domain.ID, uuid.New())
				return err
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newDomainRepoStub()
			dns := &resolverStub{records: map[string][]string{}}
			service := NewDomainService(repo, accessStub{owner: owner}, dns)
			mutate := tt.prepare(service, repo, dns)
			checks := 0
			service.access = accessStub{owner: owner, checks: &checks, failOnCheck: 2}
			if err := mutate(); !errors.Is(err, domain.ErrAccessDenied) {
				t.Fatalf("mutation error=%v checks=%d", err, checks)
			}
			if checks != 2 {
				t.Fatalf("owner checks=%d", checks)
			}
		})
	}
}

func TestDomainRequiresOwner(t *testing.T) {
	s := NewDomainService(newDomainRepoStub(), accessStub{denied: true}, &resolverStub{})
	if _, err := s.Begin(context.Background(), domain.UserPrincipal(uuid.New()), uuid.New(), "example.com", uuid.New()); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("error=%v", err)
	}
}

func TestDomainRequestReplayConflictsWithoutRotatingChallenge(t *testing.T) {
	owner, workspace, requestID := uuid.New(), uuid.New(), uuid.New()
	repo := newDomainRepoStub()
	service := NewDomainService(repo, accessStub{owner: owner}, &resolverStub{records: map[string][]string{}})
	first, err := service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "example.com", requestID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "example.com", requestID); !errors.Is(err, domain.ErrRequestIDReused) {
		t.Fatalf("replay=%v", err)
	}
	stored, _ := repo.Get(context.Background(), workspace, first.ID)
	if stored.Revision != first.Revision {
		t.Fatalf("challenge rotated revision=%d", stored.Revision)
	}
}

func TestDomainDiscoveryReturnsCapturedWorkspaceWithoutUsers(t *testing.T) {
	owner, workspace := uuid.New(), uuid.New()
	repo := newDomainRepoStub()
	dns := &resolverStub{records: map[string][]string{}}
	service := NewDomainService(repo, accessStub{owner: owner}, dns)
	challenge, _ := service.Begin(context.Background(), domain.UserPrincipal(owner), workspace, "BÜCHER.example", uuid.New())
	if _, err := service.Discover(context.Background(), "bücher.example"); !errors.Is(err, domain.ErrInaccessible) {
		t.Fatalf("pending discovery=%v", err)
	}
	dns.records[challenge.DNSName] = []string{challenge.TXTValue}
	verified, _ := service.Verify(context.Background(), domain.UserPrincipal(owner), workspace, challenge.ID, uuid.New())
	_, _ = service.Capture(context.Background(), domain.UserPrincipal(owner), workspace, verified.ID, uuid.New())
	hint, err := service.Discover(context.Background(), "BÜCHER.EXAMPLE.")
	if err != nil {
		t.Fatal(err)
	}
	if hint.Domain != "xn--bcher-kva.example" || hint.Workspace != workspace.String() {
		t.Fatalf("hint=%+v", hint)
	}
}
