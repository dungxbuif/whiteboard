package postgres

import (
	"context"
	"devtool/internal/domain"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestActivityFiltersBeforePaginationAndRetainedSince(t *testing.T) {
	db := accessTestDB(t, 19)
	owner, member, workspaceID := accessFixture(t, db)
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer')", workspaceID, member)
	visible, hidden := uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Visible'),(?,?,'Hidden')", visible, workspaceID, hidden, workspaceID)
	base := time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		mustExec(t, db, `INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,occurred_at,source,source_key) VALUES (?,?,'user',?,'comment.created',?,'activity_test',?)`, workspaceID, hidden, owner.String(), base.Add(time.Duration(i+10)*time.Second), uuid.NewString())
	}
	mustExec(t, db, "UPDATE scenes SET deleted_at=now() WHERE id=?", hidden)
	for i := 0; i < 2; i++ {
		mustExec(t, db, `INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,occurred_at,source,source_key) VALUES (?,?,'user',?,'comment.created',?,'activity_test',?)`, workspaceID, visible, owner.String(), base.Add(time.Duration(i)*time.Second), uuid.NewString())
	}
	mustExec(t, db, `INSERT INTO workspace_events(workspace_id,actor_kind,actor_id,kind,occurred_at,source,source_key) VALUES (?,'system','system','security.sso.updated',?,'activity_test',?)`, workspaceID, base.Add(100*time.Second), uuid.NewString())
	repo := NewActivityRepository(db)
	items, err := repo.ListActivity(context.Background(), member, workspaceID, nil, false, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("authorized items=%d want 2: %#v", len(items), items)
	}
	for _, item := range items {
		if item.SceneID == nil || *item.SceneID != visible || item.ActorName != "Test" {
			t.Fatalf("hidden/admin row leaked or actor unresolved: %#v", item)
		}
	}
	retained, err := repo.ActivityRetainedSince(context.Background(), member, workspaceID, nil, false)
	if err != nil || !retained.Equal(base) {
		t.Fatalf("retainedSince=%s err=%v want=%s", retained, err, base)
	}
	mustExec(t, db, "UPDATE users SET deleted_at=now() WHERE id=?", owner)
	items, err = repo.ListActivity(context.Background(), member, workspaceID, nil, false, nil, 2)
	if err != nil || len(items) != 2 || items[0].ActorName != "Former member" {
		t.Fatalf("deleted actor identity was retained: items=%#v err=%v", items, err)
	}
}

func TestActivityMixedSourcesUseStableTimestampIDCursor(t *testing.T) {
	db := accessTestDB(t, 19)
	owner, _, workspaceID := accessFixture(t, db)
	sceneID := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Scene')", sceneID, workspaceID)
	when := time.Date(2026, 10, 20, 2, 0, 0, 0, time.UTC)
	for _, source := range []string{"comment", "agent_audit", "collab_audit"} {
		mustExec(t, db, `INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,occurred_at,source,source_key) VALUES (?,?,'user',?,'scene.updated',?,?,?)`, workspaceID, sceneID, owner.String(), when, source, uuid.NewString())
	}
	repo := NewActivityRepository(db)
	first, err := repo.ListActivity(context.Background(), owner, workspaceID, &sceneID, true, nil, 2)
	if err != nil || len(first) != 2 {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	cursor := &domain.ActivityCursor{OccurredAt: first[1].OccurredAt, ID: first[1].ID}
	second, err := repo.ListActivity(context.Background(), owner, workspaceID, &sceneID, true, cursor, 2)
	if err != nil || len(second) != 1 {
		t.Fatalf("second=%#v err=%v", second, err)
	}
	seen := map[string]bool{}
	for _, item := range append(first, second...) {
		if seen[item.ID] {
			t.Fatalf("duplicate item across cursor: %s", item.ID)
		}
		seen[item.ID] = true
	}
}
