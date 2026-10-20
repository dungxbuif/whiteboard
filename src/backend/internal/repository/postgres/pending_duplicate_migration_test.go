package postgres

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestPendingDuplicateMigrationAbortAndBackfill(t *testing.T) {
	db := accessTestDB(t, 28)
	owner, _, workspace := accessFixture(t, db)
	ambiguous, staged := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{ambiguous, staged} {
		mustExec(t, db, `INSERT INTO scenes(id,workspace_id,title,content,created_by,deleted_at)
			VALUES(?,?,?,?::jsonb,?,NOW())`, id, workspace, "Hidden", `{"elements":[],"appState":{},"files":{}}`, owner)
	}
	mustExec(t, db, `INSERT INTO library_asset_stages
		(workspace_id,target_scene_id,request_id,source_asset_id,source_file_id,target_asset_id,target_file_id,storage_bucket,storage_key,state,cleanup_after)
		VALUES (?,?,?,?,?,?,?,'bucket','object','pending',NOW()+INTERVAL '1 hour')`,
		workspace, staged, uuid.New(), uuid.New(), "source-file", uuid.New(), "target-file")
	migration, err := os.ReadFile("../../../migrations/000029_pending_scene_duplicate.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	apply := func() error {
		return db.Transaction(func(tx *gorm.DB) error { return tx.Exec(string(migration)).Error })
	}
	if err := apply(); err == nil || !strings.Contains(err.Error(), "Ambiguous hidden empty scenes") {
		t.Fatalf("ambiguous candidate should abort migration: %v", err)
	}
	var columns int64
	if err := db.Raw(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema=current_schema() AND table_name='scenes'
		AND column_name IN ('pending_duplicate','duplicate_cleanup_started')`).Scan(&columns).Error; err != nil || columns != 0 {
		t.Fatalf("aborted migration left DDL behind: columns=%d err=%v", columns, err)
	}
	mustExec(t, db, "DELETE FROM scenes WHERE id=?", ambiguous)
	if err := apply(); err != nil {
		t.Fatalf("classified candidates should permit migration: %v", err)
	}
	var state struct {
		Pending bool `gorm:"column:pending_duplicate"`
		Claimed bool `gorm:"column:duplicate_cleanup_started"`
	}
	if err := db.Raw("SELECT pending_duplicate, duplicate_cleanup_started FROM scenes WHERE id=?", staged).Scan(&state).Error; err != nil || !state.Pending || state.Claimed {
		t.Fatalf("stage-backed hidden copy not backfilled: %+v err=%v", state, err)
	}
}
