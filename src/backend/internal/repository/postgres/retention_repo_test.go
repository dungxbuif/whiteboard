package postgres

import (
	"context"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func retentionFixture(t *testing.T) (*RetentionRepository, *usecase.RetentionService, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := accessTestDB(t, 26)
	owner, _, w := accessFixture(t, db)
	mustExec(t, db, "UPDATE workspace_feature_policies SET retention_enabled=true WHERE workspace_id=?", w)
	access, err := usecase.NewAccessService(NewAccessRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewRetentionRepository(db)
	service := usecase.NewRetentionService(repo, access, usecase.NewFeatureService(NewFeatureRepository(db), access))
	return repo, service, owner, w
}

func TestRetentionMigrationUpDown(t *testing.T) {
	db := accessTestDB(t, 25)
	up, err := os.ReadFile("../../../migrations/000026_workspace_retention.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(up)).Error; err != nil {
		t.Fatal(err)
	}
	var table *string
	if err = db.Raw("SELECT to_regclass('workspace_retention_policies')::text").Scan(&table).Error; err != nil || table == nil {
		t.Fatalf("up table=%v err=%v", table, err)
	}
	down, err := os.ReadFile("../../../migrations/000026_workspace_retention.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(down)).Error; err != nil {
		t.Fatal(err)
	}
	table = nil
	if err = db.Raw("SELECT to_regclass('workspace_retention_policies')::text").Scan(&table).Error; err != nil || table != nil {
		t.Fatalf("down table=%v err=%v", table, err)
	}
}

func TestRetentionSevenDayActivation(t *testing.T) {
	repo, service, owner, w := retentionFixture(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	serviceNow := now
	_ = serviceNow
	preview, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(owner), w, domain.DefaultRetentionPolicyInput())
	if err != nil {
		t.Fatal(err)
	}
	policy, err := service.Schedule(context.Background(), domain.UserPrincipal(owner), w, preview.ID, 0, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if policy.Revision != 1 || policy.EffectiveAt.Before(preview.CreatedAt.Add(domain.MinimumRetentionActivationDelay)) {
		t.Fatalf("policy=%+v preview=%+v", policy, preview)
	}
	stored, err := repo.CurrentPolicy(context.Background(), w)
	if err != nil || stored.Revision != 1 {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
}

func TestHoldWinsAgainstClaimedPurge(t *testing.T) {
	repo, service, owner, w := retentionFixture(t)
	db := repo.db
	old := time.Now().UTC().AddDate(0, 0, -200)
	mustExec(t, db, "INSERT INTO workspace_events(workspace_id,actor_kind,actor_id,kind,occurred_at,source,source_key) VALUES (?,'system','retention-test','test.old',?,'retention-test',?)", w, old, uuid.NewString())
	var eventID int64
	if err := db.Raw("SELECT id FROM workspace_events WHERE workspace_id=? ORDER BY id DESC LIMIT 1", w).Scan(&eventID).Error; err != nil {
		t.Fatal(err)
	}
	preview, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(owner), w, domain.DefaultRetentionPolicyInput())
	if err != nil {
		t.Fatal(err)
	}
	policy, err := service.Schedule(context.Background(), domain.UserPrincipal(owner), w, preview.ID, 0, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, "UPDATE workspace_retention_policies SET effective_at=? WHERE workspace_id=?", time.Now().UTC().Add(-time.Hour), w)
	_, err = service.SetHold(context.Background(), domain.UserPrincipal(owner), domain.RetentionReference{WorkspaceID: w, Kind: "activity", ID: fmt.Sprint(eventID)}, nil, true, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	heldPreview, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(owner), w, domain.DefaultRetentionPolicyInput())
	if err != nil || heldPreview.Counts.Activity != 0 {
		t.Fatalf("held preview=%+v err=%v", heldPreview.Counts, err)
	}
	result, err := repo.PurgeBatch(context.Background(), time.Now().UTC(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if result.Purged.Activity != 0 {
		t.Fatalf("held event purged result=%+v", result)
	}
	var count int64
	db.Table("workspace_events").Where("id=?", eventID).Count(&count)
	if count != 1 {
		t.Fatal("held event missing")
	}
	_, err = service.SetHold(context.Background(), domain.UserPrincipal(owner), domain.RetentionReference{WorkspaceID: w, Kind: "activity", ID: fmt.Sprint(eventID)}, nil, false, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	result, err = repo.PurgeBatch(context.Background(), time.Now().UTC(), 100)
	if err != nil || result.Purged.Activity != 1 {
		t.Fatalf("unheld result=%+v err=%v policy=%+v", result, err, policy)
	}
}

func TestPolicyRevisionCancelsStaleJob(t *testing.T) {
	repo, service, owner, w := retentionFixture(t)
	first, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(owner), w, domain.DefaultRetentionPolicyInput())
	if err != nil {
		t.Fatal(err)
	}
	p1, err := service.Schedule(context.Background(), domain.UserPrincipal(owner), w, first.ID, 0, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	changed := domain.DefaultRetentionPolicyInput()
	changed.ActivityDays = 120
	second, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(owner), w, changed)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := service.Schedule(context.Background(), domain.UserPrincipal(owner), w, second.ID, p1.Revision, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if p2.Revision != p1.Revision+1 || p2.Policy.ActivityDays != 120 {
		t.Fatalf("p1=%+v p2=%+v", p1, p2)
	}
	mustExec(t, repo.db, "UPDATE feature_jobs SET available_at=now()-interval '1 hour' WHERE workspace_id=? AND idempotency_key='policy:1'", w)
	result, err := repo.PurgeBatch(context.Background(), time.Now().UTC(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 0 {
		t.Fatalf("stale job activated replacement result=%+v", result)
	}
}

func TestRetentionNeverDeletesCanonicalHistory(t *testing.T) {
	repo, service, owner, w := retentionFixture(t)
	db := repo.db
	scene := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title,writer_mode) VALUES (?,?,'Retention canonical','v2')", scene, w)
	mustExec(t, db, "INSERT INTO collab_documents(scene_id,workspace_id,snapshot,legacy_content,legacy_content_version) VALUES (?,?,?,'{}',1)", scene, w, []byte{1})
	mustExec(t, db, "INSERT INTO collab_updates(scene_id,sequence,update_bytes,created_at) VALUES (?,1,?,?)", scene, []byte{2}, time.Now().UTC().AddDate(-1, 0, 0))
	mustExec(t, db, "INSERT INTO collab_receipts(scene_id,request_id,principal_type,principal_id,payload_hash,action,sequence,state_vector,created_at) VALUES (?,'retention-proof','user',?,repeat('a',64),'scene:update',1,'sv',?)", scene, owner.String(), time.Now().UTC().AddDate(-1, 0, 0))
	preview, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(owner), w, domain.DefaultRetentionPolicyInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Schedule(context.Background(), domain.UserPrincipal(owner), w, preview.ID, 0, uuid.New()); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, "UPDATE workspace_retention_policies SET effective_at=now()-interval '1 hour' WHERE workspace_id=?", w)
	if _, err = repo.PurgeBatch(context.Background(), time.Now().UTC(), 100); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"collab_documents", "collab_updates", "collab_receipts"} {
		var count int64
		if err = db.Table(table).Count(&count).Error; err != nil || count != 1 {
			t.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
}

func TestRetentionMutationRetryBindsCanonicalBody(t *testing.T) {
	repo, service, owner, w := retentionFixture(t)
	preview, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(owner), w, domain.DefaultRetentionPolicyInput())
	if err != nil {
		t.Fatal(err)
	}
	requestID := uuid.New()
	first, err := service.Schedule(context.Background(), domain.UserPrincipal(owner), w, preview.ID, 0, requestID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Schedule(context.Background(), domain.UserPrincipal(owner), w, preview.ID, 0, requestID)
	if err != nil || first.Revision != second.Revision || !first.EffectiveAt.Equal(second.EffectiveAt) {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
	other, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(owner), w, domain.DefaultRetentionPolicyInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Schedule(context.Background(), domain.UserPrincipal(owner), w, other.ID, first.Revision, requestID); !errors.Is(err, domain.ErrRequestIDReused) {
		t.Fatalf("changed retry err=%v", err)
	}
	var policies int64
	if err = repo.db.Table("workspace_retention_policies").Where("workspace_id=?", w).Count(&policies).Error; err != nil || policies != 1 {
		t.Fatalf("policies=%d err=%v", policies, err)
	}
}

func TestRetentionDisabledStopsDuePurge(t *testing.T) {
	repo, service, owner, w := retentionFixture(t)
	preview, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(owner), w, domain.DefaultRetentionPolicyInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Schedule(context.Background(), domain.UserPrincipal(owner), w, preview.ID, 0, uuid.New()); err != nil {
		t.Fatal(err)
	}
	mustExec(t, repo.db, "UPDATE workspace_retention_policies SET effective_at=now()-interval '1 hour' WHERE workspace_id=?", w)
	mustExec(t, repo.db, "UPDATE workspace_feature_policies SET retention_enabled=false WHERE workspace_id=?", w)
	result, err := repo.PurgeBatch(context.Background(), time.Now().UTC(), 100)
	if err != nil || result.Scanned != 0 {
		t.Fatalf("disabled result=%+v err=%v", result, err)
	}
}

func TestRetentionRedactsCommentsAndPrunesOnlyAutomaticVersions(t *testing.T) {
	repo, service, owner, w := retentionFixture(t)
	db := repo.db
	scene, thread, comment := uuid.New(), uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Retention content')", scene, w)
	mustExec(t, db, `INSERT INTO comment_threads(id,workspace_id,scene_id,creator_kind,creator_id,creator_name,anchor_x,anchor_y,created_at,updated_at) VALUES (?,?,?,'user',?,'Owner',0,0,?,?)`, thread, w, scene, owner.String(), time.Now().UTC().AddDate(-1, 0, 0), time.Now().UTC().AddDate(-1, 0, 0))
	mustExec(t, db, `INSERT INTO comments(id,workspace_id,scene_id,thread_id,author_kind,author_id,author_name,body,created_at,updated_at) VALUES (?,?,?,?,'user',?,'Owner','sensitive old text',?,?)`, comment, w, scene, thread, owner.String(), time.Now().UTC().AddDate(-1, 0, 0), time.Now().UTC().AddDate(-1, 0, 0))
	mustExec(t, db, `INSERT INTO scene_versions(id,workspace_id,scene_id,document_id,source_sequence,content,content_bytes,content_hash,byte_size,kind,creator_kind,creator_id,request_id,payload_hash,created_at)
	SELECT gen_random_uuid(),?,?,gen_random_uuid(),g,'{}'::jsonb,convert_to('{}','UTF8'),repeat('a',64),2,'automatic','system','retention','auto-'||g,repeat('b',64),clock_timestamp()-(g||' minutes')::interval FROM generate_series(1,102) g`, w, scene)
	mustExec(t, db, `INSERT INTO scene_versions(id,workspace_id,scene_id,document_id,source_sequence,content,content_bytes,content_hash,byte_size,pinned,kind,creator_kind,creator_id,request_id,payload_hash) VALUES (?,?,?,gen_random_uuid(),200,'{}',convert_to('{}','UTF8'),repeat('c',64),2,true,'automatic','system','retention','pinned',repeat('d',64)),(?,?,?,gen_random_uuid(),201,'{}',convert_to('{}','UTF8'),repeat('e',64),2,false,'manual','user',?,'manual',repeat('f',64))`, uuid.New(), w, scene, uuid.New(), w, scene, owner.String())
	commentsDays := 30
	input := domain.DefaultRetentionPolicyInput()
	input.CommentsDays = &commentsDays
	preview, err := service.PreviewPolicy(context.Background(), domain.UserPrincipal(owner), w, input)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Counts.Versions != 2 || preview.Counts.Comments != 1 {
		t.Fatalf("counts=%+v", preview.Counts)
	}
	if _, err = service.Schedule(context.Background(), domain.UserPrincipal(owner), w, preview.ID, 0, uuid.New()); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, "UPDATE workspace_retention_policies SET effective_at=now()-interval '1 hour' WHERE workspace_id=?", w)
	result, err := repo.PurgeBatch(context.Background(), time.Now().UTC(), 1000)
	if err != nil || result.Purged.Versions != 2 || result.Purged.Comments != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	var automatic, pinned, manual int64
	db.Table("scene_versions").Where("workspace_id=? AND scene_id=? AND kind='automatic' AND pinned=false", w, scene).Count(&automatic)
	db.Table("scene_versions").Where("workspace_id=? AND scene_id=? AND pinned=true", w, scene).Count(&pinned)
	db.Table("scene_versions").Where("workspace_id=? AND scene_id=? AND kind='manual'", w, scene).Count(&manual)
	if automatic != 100 || pinned != 1 || manual != 1 {
		t.Fatalf("automatic=%d pinned=%d manual=%d", automatic, pinned, manual)
	}
	var redacted struct {
		Body      string
		DeletedAt *time.Time
	}
	if err = db.Table("comments").Select("body,deleted_at").Where("id=?", comment).Scan(&redacted).Error; err != nil || redacted.Body != "" || redacted.DeletedAt == nil {
		t.Fatalf("redacted=%+v err=%v", redacted, err)
	}
	again, err := repo.PurgeBatch(context.Background(), time.Now().UTC(), 1000)
	if err != nil || len(again.Items) != 0 {
		t.Fatalf("idempotent rerun=%+v err=%v", again, err)
	}
}
