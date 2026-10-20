package postgres

import (
	"context"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type commentFixtureState struct {
	repo    *CommentRepository
	service *usecase.CommentService
	links   *usecase.ShareLinkService
	rooms   *usecase.RoomService
	owner   uuid.UUID
	editor  uuid.UUID
	viewer  uuid.UUID
	ref     domain.SceneRef
}

func commentFixture(t *testing.T, events interface {
	AppendEvent(domain.Context, domain.EventInput) (domain.WorkspaceEvent, error)
}) (*commentFixtureState, *gorm.DB) {
	t.Helper()
	db := accessTestDB(t, 17)
	owner, viewer, workspace := accessFixture(t, db)
	editor, scene := uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO users(id,email,name) VALUES (?,?,'Editor')", editor, editor.String()+"@example.test")
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?, 'viewer'),(?,?, 'editor')", workspace, viewer, workspace, editor)
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Comments')", scene, workspace)
	mustExec(t, db, "UPDATE workspace_feature_policies SET comments_enabled=true WHERE workspace_id=?", workspace)
	access, err := usecase.NewAccessService(NewAccessRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	features := usecase.NewFeatureService(NewFeatureRepository(db), access)
	if events == nil {
		events = usecase.NewEventService(NewEventRepository(db), access)
	}
	repo := NewCommentRepository(db)
	service := usecase.NewCommentService(repo, features, access, events)
	shareRepo := NewShareLinkRepository(db)
	rooms, err := usecase.NewRoomService(shareRepo, access, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	return &commentFixtureState{repo: repo, service: service, links: usecase.NewShareLinkService(shareRepo, access), rooms: rooms, owner: owner, editor: editor, viewer: viewer, ref: domain.SceneRef{WorkspaceID: workspace, SceneID: scene}}, db
}

func TestGuestViewCommentWithoutEdit(t *testing.T) {
	f, _ := commentFixture(t, nil)
	link, err := f.links.Create(context.Background(), f.owner, f.ref, domain.CreateShareLink{Access: domain.AccessView, AllowComments: true})
	if err != nil {
		t.Fatal(err)
	}
	session, err := f.rooms.ExchangePublicSession(context.Background(), link.Token, "", "Guest")
	if err != nil {
		t.Fatal(err)
	}
	guest, err := f.rooms.ParseSceneSession(session.Token)
	if err != nil {
		t.Fatal(err)
	}
	thread, err := f.service.CreateThread(context.Background(), guest.Principal, f.ref, domain.CreateThreadInput{RequestID: uuid.New(), Anchor: domain.CommentAnchor{X: 10, Y: 20}, Body: "Guest note"})
	if err != nil || thread.Messages[0].Body != "Guest note" {
		t.Fatalf("thread=%+v err=%v", thread, err)
	}

	blocked, err := f.links.Create(context.Background(), f.owner, f.ref, domain.CreateShareLink{Access: domain.AccessEdit})
	if err != nil {
		t.Fatal(err)
	}
	blockedSession, _ := f.rooms.ExchangePublicSession(context.Background(), blocked.Token, "", "No comments")
	blockedGuest, _ := f.rooms.ParseSceneSession(blockedSession.Token)
	_, err = f.service.Reply(context.Background(), blockedGuest.Principal, f.ref, thread.ID, domain.ReplyInput{RequestID: uuid.New(), Body: "must fail"})
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("comment-disabled guest accepted: %v", err)
	}
}

func TestThreadScopeAndOwnerModeration(t *testing.T) {
	f, db := commentFixture(t, nil)
	thread, err := f.service.CreateThread(context.Background(), domain.UserPrincipal(f.viewer), f.ref, domain.CreateThreadInput{RequestID: uuid.New(), Anchor: domain.CommentAnchor{X: 1, Y: 2}, Body: "viewer"})
	if err != nil {
		t.Fatal(err)
	}
	reply, err := f.service.Reply(context.Background(), domain.UserPrincipal(f.editor), f.ref, thread.ID, domain.ReplyInput{RequestID: uuid.New(), Body: "editor"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.UpdateMessage(context.Background(), domain.UserPrincipal(f.viewer), f.ref, thread.ID, reply.ID, domain.CommentEditInput{RequestID: uuid.New(), ExpectedRevision: 1, Body: "hijack"})
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("foreign edit accepted: %v", err)
	}
	if err = f.service.DeleteMessage(context.Background(), domain.UserPrincipal(f.owner), f.ref, thread.ID, reply.ID, domain.RevisionInput{RequestID: uuid.New(), ExpectedRevision: 1}); err != nil {
		t.Fatal(err)
	}
	resolved := true
	updated, err := f.service.UpdateThread(context.Background(), domain.UserPrincipal(f.editor), f.ref, thread.ID, domain.ThreadEditInput{RequestID: uuid.New(), ExpectedRevision: 1, Resolved: &resolved})
	if err != nil || updated.Status != domain.CommentStatusResolved {
		t.Fatalf("thread=%+v err=%v", updated, err)
	}
	foreignScene := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Foreign')", foreignScene, f.ref.WorkspaceID)
	_, err = f.service.GetThread(context.Background(), domain.UserPrincipal(f.owner), domain.SceneRef{WorkspaceID: f.ref.WorkspaceID, SceneID: foreignScene}, thread.ID, "", 50)
	if !errors.Is(err, domain.ErrInaccessible) {
		t.Fatalf("foreign thread leaked: %v", err)
	}
}

func TestCommentRevokeRace(t *testing.T) {
	f, db := commentFixture(t, nil)
	link, _ := f.links.Create(context.Background(), f.owner, f.ref, domain.CreateShareLink{Access: domain.AccessView, AllowComments: true})
	session, _ := f.rooms.ExchangePublicSession(context.Background(), link.Token, "", "Guest")
	guest, _ := f.rooms.ParseSceneSession(session.Token)
	if err := f.links.Revoke(context.Background(), f.owner, f.ref, link.ID); err != nil {
		t.Fatal(err)
	}
	_, err := f.service.CreateThread(context.Background(), guest.Principal, f.ref, domain.CreateThreadInput{RequestID: uuid.New(), Anchor: domain.CommentAnchor{X: 1, Y: 2}, Body: "late"})
	if !errors.Is(err, domain.ErrInaccessible) && !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("revoked guest accepted: %v", err)
	}
	var count int64
	if err := db.Table("comments").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("messages=%d err=%v", count, err)
	}
}

func TestCommentIdempotency(t *testing.T) {
	f, db := commentFixture(t, nil)
	requestID := uuid.New()
	input := domain.CreateThreadInput{RequestID: requestID, Anchor: domain.CommentAnchor{X: 1, Y: 2}, Body: "once"}
	first, err := f.service.CreateThread(context.Background(), domain.UserPrincipal(f.owner), f.ref, input)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := f.service.CreateThread(context.Background(), domain.UserPrincipal(f.owner), f.ref, input)
	if err != nil || retry.ID != first.ID {
		t.Fatalf("retry=%+v err=%v", retry, err)
	}
	input.Body = "changed"
	if _, err = f.service.CreateThread(context.Background(), domain.UserPrincipal(f.owner), f.ref, input); !errors.Is(err, domain.ErrRequestIDReused) {
		t.Fatalf("changed retry accepted: %v", err)
	}
	var messages, events, receipts int64
	db.Table("comments").Count(&messages)
	db.Table("workspace_events").Where("source='comment'").Count(&events)
	db.Table("feature_mutation_receipts").Where("request_id=?", requestID).Count(&receipts)
	if messages != 1 || events != 1 || receipts != 1 {
		t.Fatalf("messages=%d events=%d receipts=%d", messages, events, receipts)
	}
	var metadata string
	if err := db.Raw("SELECT safe_metadata::text FROM workspace_events WHERE source='comment'").Scan(&metadata).Error; err != nil || strings.Contains(metadata, "once") || strings.Contains(metadata, "body") {
		t.Fatalf("event leaked comment body: %q err=%v", metadata, err)
	}
}

func TestCommentOwnEditDeleteAndRevisionConflict(t *testing.T) {
	f, _ := commentFixture(t, nil)
	principal := domain.UserPrincipal(f.viewer)
	thread, err := f.service.CreateThread(context.Background(), principal, f.ref, domain.CreateThreadInput{RequestID: uuid.New(), Anchor: domain.CommentAnchor{X: 1, Y: 2}, Body: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	message := thread.Messages[0]
	updated, err := f.service.UpdateMessage(context.Background(), principal, f.ref, thread.ID, message.ID, domain.CommentEditInput{RequestID: uuid.New(), ExpectedRevision: 1, Body: "edited"})
	if err != nil || updated.Revision != 2 || updated.Body != "edited" {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	_, err = f.service.UpdateMessage(context.Background(), principal, f.ref, thread.ID, message.ID, domain.CommentEditInput{RequestID: uuid.New(), ExpectedRevision: 1, Body: "stale"})
	if !errors.Is(err, domain.ErrRevisionConflict) {
		t.Fatalf("stale edit accepted: %v", err)
	}
	if err = f.service.DeleteMessage(context.Background(), principal, f.ref, thread.ID, message.ID, domain.RevisionInput{RequestID: uuid.New(), ExpectedRevision: 2}); err != nil {
		t.Fatal(err)
	}
	redacted, err := f.service.GetThread(context.Background(), principal, f.ref, thread.ID, "", 50)
	if err != nil || redacted.Messages[0].DeletedAt == nil || redacted.Messages[0].Body != "" {
		t.Fatalf("thread=%+v err=%v", redacted, err)
	}
}

func TestCommentThreadPagesEveryReplyAfterFifty(t *testing.T) {
	f, db := commentFixture(t, nil)
	principal := domain.UserPrincipal(f.owner)
	thread, err := f.service.CreateThread(context.Background(), principal, f.ref, domain.CreateThreadInput{RequestID: uuid.New(), Anchor: domain.CommentAnchor{X: 1, Y: 2}, Body: "message-00"})
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index < 55; index++ {
		mustExec(t, db, `INSERT INTO comments(id,workspace_id,scene_id,thread_id,author_kind,author_id,author_name,body)
			VALUES (?,?,?,?, 'user', ?, 'Test', ?)`, uuid.New(), f.ref.WorkspaceID, f.ref.SceneID, thread.ID, f.owner.String(), fmt.Sprintf("message-%02d", index))
	}
	first, err := f.service.GetThread(context.Background(), principal, f.ref, thread.ID, "", 50)
	if err != nil || len(first.Messages) != 50 || first.NextMessageCursor == "" || first.Messages[49].Body != "message-49" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := f.service.GetThread(context.Background(), principal, f.ref, thread.ID, first.NextMessageCursor, 50)
	if err != nil || len(second.Messages) != 5 || second.NextMessageCursor != "" || second.Messages[0].Body != "message-50" || second.Messages[4].Body != "message-54" {
		t.Fatalf("second=%+v err=%v", second, err)
	}

	foreign, err := f.service.CreateThread(context.Background(), principal, f.ref, domain.CreateThreadInput{RequestID: uuid.New(), Anchor: domain.CommentAnchor{X: 3, Y: 4}, Body: "foreign"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.GetThread(context.Background(), principal, f.ref, thread.ID, foreign.Messages[0].ID.String(), 50)
	if !errors.Is(err, domain.ErrInaccessible) {
		t.Fatalf("foreign cursor accepted: %v", err)
	}
}

type failingCommentEvents struct{}

func (failingCommentEvents) AppendEvent(domain.Context, domain.EventInput) (domain.WorkspaceEvent, error) {
	return domain.WorkspaceEvent{}, errors.New("event failed")
}

func TestCommentRollbackLeavesNoMessageEventOrReceipt(t *testing.T) {
	f, db := commentFixture(t, failingCommentEvents{})
	requestID := uuid.New()
	_, err := f.service.CreateThread(context.Background(), domain.UserPrincipal(f.owner), f.ref, domain.CreateThreadInput{RequestID: requestID, Anchor: domain.CommentAnchor{X: 1, Y: 2}, Body: "rollback"})
	if err == nil {
		t.Fatal("event failure accepted")
	}
	for _, table := range []string{"comment_threads", "comments", "workspace_events", "feature_mutation_receipts"} {
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
}

func TestCommentMigrationUpDown(t *testing.T) {
	db := accessTestDB(t, 16)
	up, err := os.ReadFile("../../../migrations/000017_comments.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(up)).Error; err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"comment_threads", "comments", "comment_subscriptions"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("missing %s", table)
		}
	}
	down, err := os.ReadFile("../../../migrations/000017_comments.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(down)).Error; err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable("comment_threads") {
		t.Fatal("comment tables survived down")
	}
}
