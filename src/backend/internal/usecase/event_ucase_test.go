package usecase

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type eventRepositoryStub struct {
	events []domain.WorkspaceEvent
}

func (r *eventRepositoryStub) AppendEvent(_ domain.Context, input domain.EventInput) (domain.WorkspaceEvent, error) {
	event := domain.WorkspaceEvent{ID: "1", WorkspaceID: input.WorkspaceID, SceneID: input.SceneID, ActorKind: input.ActorKind, ActorID: input.ActorID, Kind: input.Kind, SubjectID: input.SubjectID, SafeMetadata: input.SafeMetadata}
	r.events = append(r.events, event)
	return event, nil
}

func (r *eventRepositoryStub) ProjectAuditBatch(domain.Context, int) (int, error) { return 0, nil }
func (r *eventRepositoryStub) ListWorkspaceEvents(_ domain.Context, workspaceID uuid.UUID, cursor string, limit int) ([]domain.WorkspaceEvent, error) {
	start := 0
	if cursor == "1" {
		start = 1
	}
	end := min(len(r.events), start+limit)
	return append([]domain.WorkspaceEvent(nil), r.events[start:end]...), nil
}

type eventAccessStub struct {
	workspace domain.AccessDecision
	denied    map[uuid.UUID]bool
}

func (a eventAccessStub) ResolveWorkspace(domain.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error) {
	if a.workspace.Access == "" {
		return domain.AccessDecision{}, domain.ErrInaccessible
	}
	return a.workspace, nil
}

func (a eventAccessStub) Resolve(_ domain.Context, _ domain.Principal, ref domain.SceneRef) (domain.AccessDecision, error) {
	if a.denied[ref.SceneID] {
		return domain.AccessDecision{}, domain.ErrInaccessible
	}
	return domain.AccessDecision{Access: domain.AccessView}, nil
}

func TestEventContainsNoSecrets(t *testing.T) {
	service := NewEventService(&eventRepositoryStub{}, eventAccessStub{workspace: domain.AccessDecision{Access: domain.AccessEdit, Owner: true}})
	base := domain.EventInput{WorkspaceID: uuid.New(), ActorKind: domain.EventActorUser, ActorID: uuid.NewString(), Kind: "comment.created", SafeMetadata: map[string]any{"threadId": uuid.NewString()}, Source: "comment", SourceKey: uuid.NewString()}
	if _, err := service.AppendEvent(context.Background(), base); err != nil {
		t.Fatalf("safe event rejected: %v", err)
	}
	for _, unsafe := range []map[string]any{
		{"body": "private comment"},
		{"nested": map[string]any{"token": "secret"}},
		{"candidate": "candidate:1"},
		{"sdp": "v=0"},
	} {
		input := base
		input.SafeMetadata = unsafe
		if _, err := service.AppendEvent(context.Background(), input); !errors.Is(err, domain.ErrUnsafeEventMetadata) {
			t.Fatalf("unsafe metadata accepted: %#v (%v)", unsafe, err)
		}
	}
}

func TestRestrictedSceneDoesNotLeak(t *testing.T) {
	workspaceID, visible, restricted := uuid.New(), uuid.New(), uuid.New()
	repo := &eventRepositoryStub{events: []domain.WorkspaceEvent{
		{ID: "1", WorkspaceID: workspaceID, SceneID: &restricted, ActorKind: domain.EventActorUser, ActorID: uuid.NewString(), Kind: "comment.created"},
		{ID: "2", WorkspaceID: workspaceID, SceneID: &visible, ActorKind: domain.EventActorUser, ActorID: uuid.NewString(), Kind: "comment.created"},
		{ID: "3", WorkspaceID: workspaceID, ActorKind: domain.EventActorSystem, ActorID: "system", Kind: "security.sso.updated"},
	}}
	service := NewEventService(repo, eventAccessStub{workspace: domain.AccessDecision{Access: domain.AccessView}, denied: map[uuid.UUID]bool{restricted: true}})
	page, err := service.ListWorkspace(context.Background(), domain.UserPrincipal(uuid.New()), workspaceID, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].SceneID == nil || *page.Items[0].SceneID != visible {
		t.Fatalf("restricted or owner event leaked: %#v", page.Items)
	}
}
