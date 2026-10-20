package postgres

import (
	"context"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
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

func TestAppendEventWithoutCallerTransactionRollsBackOutboxFailure(t *testing.T) {
	db := accessTestDB(t, 15)
	owner, _, workspaceID := accessFixture(t, db)
	mustExec(t, db, `CREATE FUNCTION reject_event_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'outbox rejected'; END $$`)
	mustExec(t, db, `CREATE TRIGGER reject_event_outbox BEFORE INSERT ON workspace_event_outbox FOR EACH ROW EXECUTE FUNCTION reject_event_outbox()`)
	repo := NewEventRepository(db)
	_, err := repo.AppendEvent(context.Background(), domain.EventInput{WorkspaceID: workspaceID, ActorKind: domain.EventActorUser, ActorID: owner.String(), Kind: "workspace.updated", Source: "workspace", SourceKey: uuid.NewString()})
	if err == nil {
		t.Fatal("outbox failure unexpectedly committed")
	}
	var events int64
	if err := db.Table("workspace_events").Count(&events).Error; err != nil || events != 0 {
		t.Fatalf("event survived outbox failure: count=%d err=%v", events, err)
	}
}

func TestAppendEventIdenticalRetryHasSingleOutboxNotification(t *testing.T) {
	db := accessTestDB(t, 15)
	owner, _, workspaceID := accessFixture(t, db)
	repo := NewEventRepository(db)
	input := domain.EventInput{WorkspaceID: workspaceID, ActorKind: domain.EventActorUser, ActorID: owner.String(), Kind: "workspace.updated", SafeMetadata: map[string]any{"revision": "2"}, Source: "workspace", SourceKey: uuid.NewString()}
	first, err := repo.AppendEvent(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := repo.AppendEvent(context.Background(), input)
	if err != nil || retry.ID != first.ID {
		t.Fatalf("retry=%+v first=%+v err=%v", retry, first, err)
	}
	var events, outbox int64
	db.Table("workspace_events").Count(&events)
	db.Table("workspace_event_outbox").Count(&outbox)
	if events != 1 || outbox != 1 {
		t.Fatalf("identical retry duplicated notify rows: events=%d outbox=%d", events, outbox)
	}
}

func TestProjectionRecoversLateCommittedACLRowAndStaleCursor(t *testing.T) {
	db := accessTestDB(t, 15)
	owner, _, workspaceID := accessFixture(t, db)
	mustExec(t, db, "DELETE FROM workspace_event_source_cursors")
	mustExec(t, db, "DELETE FROM workspace_event_outbox")
	mustExec(t, db, "DELETE FROM workspace_events")
	mustExec(t, db, "DELETE FROM acl_audit_events")
	var schema string
	if err := db.Raw("SELECT current_schema()").Scan(&schema).Error; err != nil {
		t.Fatal(err)
	}
	lateDB, err := gorm.Open(gormpostgres.Open(os.Getenv("ACCESS_TEST_DATABASE_URL")+"&search_path="+schema), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	late := lateDB.Begin()
	if late.Error != nil {
		t.Fatal(late.Error)
	}
	defer late.Rollback()
	var lateID int64
	if err := late.Raw("INSERT INTO acl_audit_events(workspace_id,actor_id,event_type,acl_version) VALUES (?,?, 'late',2) RETURNING id", workspaceID, owner).Scan(&lateID).Error; err != nil {
		late.Rollback()
		t.Fatal(err)
	}
	mustExec(t, db, "INSERT INTO acl_audit_events(workspace_id,actor_id,event_type,acl_version) VALUES (?,?, 'visible',3)", workspaceID, owner)
	repo := NewEventRepository(db)
	if count, err := repo.ProjectAuditBatch(context.Background(), 100); err != nil || count != 1 {
		late.Rollback()
		t.Fatalf("visible projection: count=%d err=%v", count, err)
	}
	if err := late.Commit().Error; err != nil {
		t.Fatal(err)
	}
	if count, err := repo.ProjectAuditBatch(context.Background(), 100); err != nil || count != 1 {
		t.Fatalf("late projection: count=%d err=%v", count, err)
	}
	mustExec(t, db, "UPDATE workspace_event_source_cursors SET cursor_key='999999999999' WHERE source='acl_audit'")
	mustExec(t, db, "INSERT INTO acl_audit_events(workspace_id,actor_id,event_type,acl_version) VALUES (?,?, 'stale_cursor',4)", workspaceID, owner)
	if count, err := repo.ProjectAuditBatch(context.Background(), 100); err != nil || count != 1 {
		t.Fatalf("stale cursor recovery: count=%d err=%v", count, err)
	}
	var events int64
	db.Table("workspace_events").Where("source='acl_audit'").Count(&events)
	if events != 3 {
		t.Fatalf("projected events=%d, want 3 (late id %d)", events, lateID)
	}
}
