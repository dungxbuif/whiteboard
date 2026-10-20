package usecase

import (
	"context"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type retentionAccessStub struct {
	decision domain.AccessDecision
	checks   int
}

func (s *retentionAccessStub) ResolveWorkspace(domain.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error) {
	s.checks++
	return s.decision, nil
}
func (s *retentionAccessStub) WithWorkspaceLock(ctx domain.Context, _ uuid.UUID, _ uuid.UUID, fn func(domain.Context) error) error {
	return fn(ctx)
}

type retentionFeatureStub struct{ enabled bool }

func (s retentionFeatureStub) Policy(_ domain.Context, _ domain.Principal, w uuid.UUID) (domain.WorkspaceFeaturePolicy, error) {
	return domain.WorkspaceFeaturePolicy{WorkspaceID: w, RetentionEnabled: s.enabled}, nil
}

type retentionRepoStub struct {
	policy      domain.RetentionPolicy
	preview     domain.RetentionPreview
	counts      domain.RetentionCounts
	seenInput   domain.RetentionPolicyInput
	scheduledAt time.Time
	hold        domain.RetentionHold
	purged      bool
}

func (s *retentionRepoStub) CurrentPolicy(_ domain.Context, w uuid.UUID) (domain.RetentionPolicy, error) {
	if s.policy.WorkspaceID == uuid.Nil {
		s.policy = domain.RetentionPolicy{WorkspaceID: w, Revision: 0, Policy: domain.DefaultRetentionPolicyInput()}
	}
	return s.policy, nil
}
func (s *retentionRepoStub) PreviewCounts(_ domain.Context, _ uuid.UUID, input domain.RetentionPolicyInput, _ time.Time) (domain.RetentionCounts, error) {
	s.seenInput = input
	return s.counts, nil
}
func (s *retentionRepoStub) SavePreview(_ domain.Context, value domain.RetentionPreview) error {
	s.preview = value
	return nil
}
func (s *retentionRepoStub) Preview(_ domain.Context, _ uuid.UUID, _ uuid.UUID) (domain.RetentionPreview, error) {
	return s.preview, nil
}
func (s *retentionRepoStub) WithMutation(ctx domain.Context, _ domain.Principal, _ uuid.UUID, _ uuid.UUID, _ string, _ []byte, fn func(domain.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	return fn(ctx)
}
func (s *retentionRepoStub) Schedule(_ domain.Context, w uuid.UUID, p domain.RetentionPreview, _ int64, now time.Time) (domain.RetentionPolicy, error) {
	s.scheduledAt = now
	s.policy = domain.RetentionPolicy{WorkspaceID: w, Revision: p.PolicyRevision + 1, Policy: p.Policy, PolicyHash: p.PolicyHash, EffectiveAt: now.Add(domain.MinimumRetentionActivationDelay), ScheduledAt: now}
	return s.policy, nil
}
func (s *retentionRepoStub) SetHold(_ domain.Context, ref domain.RetentionReference, v *uuid.UUID, held bool, now time.Time) (domain.RetentionHold, error) {
	s.hold = domain.RetentionHold{ID: uuid.New(), Reference: ref, VersionID: v, Held: held, UpdatedAt: now}
	return s.hold, nil
}
func (s *retentionRepoStub) PurgeBatch(domain.Context, time.Time, int) (domain.PurgeResult, error) {
	s.purged = true
	return domain.PurgeResult{Scanned: 1, Items: []domain.RetentionPurgeItem{}}, nil
}

func newRetentionUnitService(w uuid.UUID, repo *retentionRepoStub) (*RetentionService, *retentionAccessStub, domain.Principal) {
	u := uuid.New()
	access := &retentionAccessStub{decision: domain.AccessDecision{Access: domain.AccessEdit, Owner: true}}
	service := NewRetentionService(repo, access, retentionFeatureStub{enabled: true})
	return service, access, domain.UserPrincipal(u)
}

func TestRetentionDefaultCutoffs(t *testing.T) {
	w := uuid.New()
	repo := &retentionRepoStub{counts: domain.RetentionCounts{Activity: 3, Notifications: 2, Versions: 1}}
	service, _, p := newRetentionUnitService(w, repo)
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	input := domain.DefaultRetentionPolicyInput()
	preview, err := service.PreviewPolicy(context.Background(), p, w, input)
	if err != nil {
		t.Fatal(err)
	}
	if repo.seenInput.ActivityDays != 90 || repo.seenInput.NotificationDays != 30 || repo.seenInput.AutomaticVersionLimit != 100 || repo.seenInput.ManualVersionDays != nil || repo.seenInput.CommentsDays != nil || repo.seenInput.TrashHardPurge {
		t.Fatalf("defaults=%+v", repo.seenInput)
	}
	if preview.Counts != repo.counts || !preview.EarliestDestructiveAt.Equal(now.Add(7*24*time.Hour)) {
		t.Fatalf("preview=%+v", preview)
	}
	if _, err = service.PreviewPolicy(context.Background(), p, w, domain.RetentionPolicyInput{ActivityDays: 90, NotificationDays: 30, AutomaticVersionLimit: 100, TrashHardPurge: true}); !errors.Is(err, domain.ErrInvalidRetention) {
		t.Fatalf("trash hard purge err=%v", err)
	}
}

func TestRetentionManualVersionDaysAndSceneHoldValidation(t *testing.T) {
	w := uuid.New()
	repo := &retentionRepoStub{}
	service, _, p := newRetentionUnitService(w, repo)
	days := 30
	input := domain.DefaultRetentionPolicyInput()
	input.ManualVersionDays = &days
	if _, err := service.PreviewPolicy(context.Background(), p, w, input); err != nil {
		t.Fatalf("explicit manual retention rejected: %v", err)
	}
	sceneID := uuid.New()
	hold, err := service.SetHold(context.Background(), p, domain.RetentionReference{WorkspaceID: w, Kind: "scene", ID: sceneID.String()}, nil, true, uuid.New())
	if err != nil || !hold.Held {
		t.Fatalf("scene hold=%+v err=%v", hold, err)
	}
	if _, err = service.SetHold(context.Background(), p, domain.RetentionReference{WorkspaceID: w, Kind: "scene", ID: "not-a-uuid"}, nil, true, uuid.New()); !errors.Is(err, domain.ErrInvalidRetention) {
		t.Fatalf("invalid scene hold err=%v", err)
	}
}

func TestRetentionSevenDayActivationAndCurrentOwner(t *testing.T) {
	w := uuid.New()
	repo := &retentionRepoStub{}
	service, access, p := newRetentionUnitService(w, repo)
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	preview, err := service.PreviewPolicy(context.Background(), p, w, domain.DefaultRetentionPolicyInput())
	if err != nil {
		t.Fatal(err)
	}
	policy, err := service.Schedule(context.Background(), p, w, preview.ID, 0, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if policy.EffectiveAt.Before(now.Add(domain.MinimumRetentionActivationDelay)) {
		t.Fatalf("effective=%v", policy.EffectiveAt)
	}
	if access.checks < 3 {
		t.Fatalf("owner was not rechecked inside lock: %d", access.checks)
	}
}

func TestRetentionMutationResponseRoundTrip(t *testing.T) {
	w := uuid.New()
	repo := &retentionRepoStub{}
	service, _, p := newRetentionUnitService(w, repo)
	now := time.Now().UTC()
	service.now = func() time.Time { return now }
	input := domain.DefaultRetentionPolicyInput()
	hash, _ := policyDigest(input)
	repo.preview = domain.RetentionPreview{ID: uuid.New(), WorkspaceID: w, PolicyRevision: 0, Policy: input, PolicyHash: hash, EarliestDestructiveAt: now.Add(7 * 24 * time.Hour), ExpiresAt: now.Add(time.Hour)}
	value, err := service.Schedule(context.Background(), p, w, repo.preview.ID, 0, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(value)
	if len(raw) == 0 || value.Revision != 1 {
		t.Fatalf("value=%+v", value)
	}
	hold, err := service.SetHold(context.Background(), p, domain.RetentionReference{WorkspaceID: w, Kind: "version", ID: uuid.NewString()}, nil, true, uuid.New())
	if err != nil || !hold.Held {
		t.Fatalf("hold=%+v err=%v", hold, err)
	}
}

func TestRetentionOwnerAndFeatureRequired(t *testing.T) {
	w := uuid.New()
	u := uuid.New()
	repo := &retentionRepoStub{}
	access := &retentionAccessStub{decision: domain.AccessDecision{Access: domain.AccessEdit}}
	service := NewRetentionService(repo, access, retentionFeatureStub{enabled: true})
	if _, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(u), w, domain.DefaultRetentionPolicyInput()); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("nonowner err=%v", err)
	}
	access.decision.Owner = true
	service = NewRetentionService(repo, access, retentionFeatureStub{})
	if _, err := service.Policy(context.Background(), domain.UserPrincipal(u), w); !errors.Is(err, domain.ErrFeatureDisabled) {
		t.Fatalf("disabled err=%v", err)
	}
}
