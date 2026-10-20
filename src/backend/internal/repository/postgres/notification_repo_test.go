package postgres

import (
	"context"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
)

func TestNotificationRetryAndRevocation(t *testing.T) {
	db := accessTestDB(t, 19)
	owner, recipient, workspace := accessFixture(t, db)
	scene, thread := uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer')", workspace, recipient)
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Inbox')", scene, workspace)
	mustExec(t, db, "UPDATE workspace_feature_policies SET comments_enabled=true WHERE workspace_id=?", workspace)
	mustExec(t, db, `INSERT INTO comment_threads(id,workspace_id,scene_id,creator_kind,creator_id,creator_name,anchor_x,anchor_y,status)
		VALUES (?,?,?,'user',?,'Owner',1,2,'open')`, thread, workspace, scene, owner.String())
	mustExec(t, db, "INSERT INTO comment_subscriptions(workspace_id,scene_id,thread_id,user_id) VALUES (?,?,?,?)", workspace, scene, thread, recipient)
	mustExec(t, db, `INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,subject_id,safe_metadata,source,source_key)
		VALUES (?,?, 'user', ?, 'comment.replied', ?, jsonb_build_object('threadId',?::text,'commentId',?::text), 'comment', ?)`, workspace, scene, owner.String(), uuid.NewString(), thread, uuid.New(), uuid.NewString())
	var eventID int64
	if err := db.Raw("SELECT max(id) FROM workspace_events").Scan(&eventID).Error; err != nil {
		t.Fatal(err)
	}
	var queued int64
	if err := db.Table("feature_jobs").Where("event_id=? AND kind='notification.deliver'", eventID).Count(&queued).Error; err != nil || queued != 1 {
		t.Fatalf("queued=%d err=%v", queued, err)
	}

	access, err := usecase.NewAccessService(NewAccessRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	features := usecase.NewFeatureService(NewFeatureRepository(db), access)
	repo := NewNotificationRepository(db)
	subscribers, subErr := repo.Subscribers(context.Background(), domain.SceneRef{WorkspaceID: workspace, SceneID: scene}, thread)
	if subErr != nil || len(subscribers) != 1 {
		t.Fatalf("subscribers=%v err=%v", subscribers, subErr)
	}
	if authErr := features.Require(context.Background(), domain.UserPrincipal(recipient), domain.SceneRef{WorkspaceID: workspace, SceneID: scene}, domain.FeatureCommentRead); authErr != nil {
		t.Fatalf("recipient authority: %v", authErr)
	}
	service := usecase.NewNotificationService(repo, features, access)
	if err = service.DeliverEvent(context.Background(), eventID); err != nil {
		t.Fatal(err)
	}
	if err = service.DeliverEvent(context.Background(), eventID); err != nil {
		t.Fatal(err)
	}
	var delivered int64
	if err = db.Table("user_notifications").Where("user_id=? AND event_id=?", recipient, eventID).Count(&delivered).Error; err != nil || delivered != 1 {
		t.Fatalf("delivered=%d err=%v", delivered, err)
	}
	page, err := service.List(context.Background(), recipient, "", 50)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	wantLink := "/w/" + workspace.String() + "/scenes/" + scene.String() + "?thread=" + thread.String()
	if page.Items[0].DeepLink != wantLink {
		t.Fatalf("deepLink=%q want=%q", page.Items[0].DeepLink, wantLink)
	}
	notificationID := page.Items[0].ID
	readRequest := uuid.New()
	if err = service.MarkRead(context.Background(), recipient, notificationID, readRequest); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, `INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,subject_id,safe_metadata,source,source_key)
		VALUES (?,?, 'user', ?, 'comment.replied', ?, jsonb_build_object('threadId',?::text,'commentId',?::text), 'comment', ?)`, workspace, scene, owner.String(), uuid.NewString(), thread, uuid.New(), uuid.NewString())
	var secondEventID int64
	if err = db.Raw("SELECT max(id) FROM workspace_events").Scan(&secondEventID).Error; err != nil {
		t.Fatal(err)
	}
	if err = service.DeliverEvent(context.Background(), secondEventID); err != nil {
		t.Fatal(err)
	}
	page, err = service.List(context.Background(), recipient, "", 50)
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("second page=%+v err=%v", page, err)
	}
	if err = service.MarkRead(context.Background(), recipient, page.Items[0].ID, readRequest); !errors.Is(err, domain.ErrRequestIDReused) {
		t.Fatalf("read request reuse=%v", err)
	}

	mustExec(t, db, "DELETE FROM workspace_members WHERE workspace_id=? AND user_id=?", workspace, recipient)
	page, err = service.List(context.Background(), recipient, "", 50)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("revoked page=%+v err=%v", page, err)
	}
	if err = service.MarkRead(context.Background(), recipient, notificationID, uuid.New()); err == nil {
		t.Fatal("revoked recipient marked notification read")
	}
	mustExec(t, db, `INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,subject_id,safe_metadata,source,source_key)
		VALUES (?,?, 'user', ?, 'comment.replied', ?, jsonb_build_object('threadId',?::text,'commentId',?::text), 'comment', ?)`, workspace, scene, owner.String(), uuid.NewString(), thread, uuid.New(), uuid.NewString())
	var revokedEventID int64
	if err = db.Raw("SELECT max(id) FROM workspace_events").Scan(&revokedEventID).Error; err != nil {
		t.Fatal(err)
	}
	if err = service.DeliverEvent(context.Background(), revokedEventID); err != nil {
		t.Fatal(err)
	}
	if err = db.Table("user_notifications").Where("user_id=? AND event_id=?", recipient, revokedEventID).Count(&delivered).Error; err != nil || delivered != 0 {
		t.Fatalf("revoked delivery=%d err=%v", delivered, err)
	}
	_ = domain.Notification{}
}

func TestNotificationPaginationScansPastRevokedRows(t *testing.T) {
	db := accessTestDB(t, 19)
	owner, recipient, revokedWorkspace := accessFixture(t, db)
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer')", revokedWorkspace, recipient)
	authorizedOwner, authorizedWorkspace := uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO users(id,email,name) VALUES (?,?,'Owner 2')", authorizedOwner, authorizedOwner.String()+"@example.test")
	mustExec(t, db, "INSERT INTO workspaces(id,name,owner_id) VALUES (?,'Authorized',?)", authorizedWorkspace, authorizedOwner)
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer')", authorizedWorkspace, recipient)
	revokedScene, authorizedScene := uuid.New(), uuid.New()
	revokedThread, authorizedThread := uuid.New(), uuid.New()
	for _, row := range []struct{ workspace, scene, thread, creator uuid.UUID }{{revokedWorkspace, revokedScene, revokedThread, owner}, {authorizedWorkspace, authorizedScene, authorizedThread, authorizedOwner}} {
		mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Inbox')", row.scene, row.workspace)
		mustExec(t, db, "UPDATE workspace_feature_policies SET comments_enabled=true WHERE workspace_id=?", row.workspace)
		mustExec(t, db, `INSERT INTO comment_threads(id,workspace_id,scene_id,creator_kind,creator_id,creator_name,anchor_x,anchor_y,status) VALUES (?,?,?,'user',?,'Owner',1,2,'open')`, row.thread, row.workspace, row.scene, row.creator.String())
	}
	for index := 0; index < 55; index++ {
		mustExec(t, db, `INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,subject_id,safe_metadata,source,source_key,occurred_at) VALUES (?,?, 'user', ?, 'comment.replied', ?, jsonb_build_object('threadId',?::text), 'comment', ?, clock_timestamp() + (? * interval '1 second'))`, revokedWorkspace, revokedScene, owner.String(), uuid.NewString(), revokedThread, uuid.NewString(), index+10)
		var eventID int64
		db.Raw("SELECT max(id) FROM workspace_events").Scan(&eventID)
		mustExec(t, db, "INSERT INTO user_notifications(user_id,workspace_id,scene_id,event_id,kind,actor_name,thread_id,created_at) VALUES (?,?,?,?, 'comment.replied','Owner',?,clock_timestamp() + (? * interval '1 second'))", recipient, revokedWorkspace, revokedScene, eventID, revokedThread, index+10)
	}
	for index := 0; index < 3; index++ {
		mustExec(t, db, `INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,subject_id,safe_metadata,source,source_key) VALUES (?,?, 'user', ?, 'comment.replied', ?, jsonb_build_object('threadId',?::text), 'comment', ?)`, authorizedWorkspace, authorizedScene, authorizedOwner.String(), uuid.NewString(), authorizedThread, uuid.NewString())
		var eventID int64
		db.Raw("SELECT max(id) FROM workspace_events").Scan(&eventID)
		mustExec(t, db, "INSERT INTO user_notifications(user_id,workspace_id,scene_id,event_id,kind,actor_name,thread_id,created_at) VALUES (?,?,?,?, 'comment.replied','Owner 2',?,clock_timestamp() - (? * interval '1 second'))", recipient, authorizedWorkspace, authorizedScene, eventID, authorizedThread, index+10)
	}
	mustExec(t, db, "DELETE FROM workspace_members WHERE workspace_id=? AND user_id=?", revokedWorkspace, recipient)
	access, _ := usecase.NewAccessService(NewAccessRepository(db))
	features := usecase.NewFeatureService(NewFeatureRepository(db), access)
	service := usecase.NewNotificationService(NewNotificationRepository(db), features, access)
	first, err := service.List(context.Background(), recipient, "", 2)
	if err != nil || len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := service.List(context.Background(), recipient, first.NextCursor, 2)
	if err != nil || len(second.Items) != 1 || second.NextCursor != "" || second.Items[0].ID == first.Items[0].ID || second.Items[0].ID == first.Items[1].ID {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	mustExec(t, db, "DROP TABLE workspace_feature_policies CASCADE")
	if _, err = service.List(context.Background(), recipient, "", 2); err == nil {
		t.Fatal("feature-policy infrastructure failure was treated as an authorization filter")
	}
}

func TestNotificationMigrationUpDown(t *testing.T) {
	db := accessTestDB(t, 18)
	up, err := os.ReadFile("../../../migrations/000019_user_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(up)).Error; err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable("user_notifications") {
		t.Fatal("user_notifications missing after up")
	}
	down, err := os.ReadFile("../../../migrations/000019_user_notifications.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(down)).Error; err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable("user_notifications") {
		t.Fatal("user_notifications survived down")
	}
}
