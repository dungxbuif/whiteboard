package postgres

import (
	"context"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestEventRollbackIsAtomic(t *testing.T) {
	db := accessTestDB(t, 15)
	owner, _, workspaceID := accessFixture(t, db)
	access, err := usecase.NewAccessService(NewAccessRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewEventRepository(db)
	sentinel := errors.New("rollback")
	err = access.WithWorkspaceLock(context.Background(), workspaceID, owner, func(txCtx domain.Context) error {
		_, appendErr := repo.AppendEvent(txCtx, domain.EventInput{WorkspaceID: workspaceID, ActorKind: domain.EventActorUser, ActorID: owner.String(), Kind: "workspace.updated", SubjectID: uuid.NewString(), SafeMetadata: map[string]any{"revision": "2"}, Source: "workspace", SourceKey: uuid.NewString()})
		if appendErr != nil {
			return appendErr
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("unexpected transaction result: %v", err)
	}
	for _, table := range []string{"workspace_events", "workspace_event_outbox"} {
		var count int64
		if queryErr := db.Table(table).Count(&count).Error; queryErr != nil {
			t.Fatal(queryErr)
		}
		if count != 0 {
			t.Fatalf("%s retained rolled-back rows: %d", table, count)
		}
	}
}

func TestProjectionCrashRetryDeduplicates(t *testing.T) {
	db := accessTestDB(t, 15)
	owner, _, workspaceID := accessFixture(t, db)
	mustExec(t, db, "DELETE FROM acl_audit_events")
	mustExec(t, db, "INSERT INTO acl_audit_events(workspace_id,actor_id,event_type,acl_version) VALUES (?,?, 'workspace_members.update',2)", workspaceID, owner)
	repo := NewEventRepository(db)
	count, err := repo.ProjectAuditBatch(context.Background(), 100)
	if err != nil || count != 1 {
		t.Fatalf("initial projection: %d %v", count, err)
	}
	count, err = repo.ProjectAuditBatch(context.Background(), 100)
	if err != nil || count != 0 {
		t.Fatalf("retry projected duplicate: %d %v", count, err)
	}
	var events, outbox int64
	db.Table("workspace_events").Where("source='acl_audit'").Count(&events)
	db.Table("workspace_event_outbox").Count(&outbox)
	if events != 1 || outbox != 1 {
		t.Fatalf("projection duplicates: events=%d outbox=%d", events, outbox)
	}
}
