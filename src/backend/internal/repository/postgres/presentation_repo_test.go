package postgres

import (
	"context"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPresentationLinkRevokeDoesNotRevokeMembership(t *testing.T) {
	db := accessTestDB(t, 20)
	owner, _, workspace := accessFixture(t, db)
	scene, session, linkID := uuid.New(), uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Present')", scene, workspace)
	raw := "presentation-secret"
	sum := sha256.Sum256([]byte(raw))
	repo := NewPresentationRepository(db)
	link := &domain.PresentationLink{ID: linkID, WorkspaceID: workspace, SceneID: scene, SessionID: session, TokenHash: hex.EncodeToString(sum[:]), CreatedBy: owner, Revision: 1, Active: true, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.CreateLink(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	found, err := repo.FindLinkByHash(context.Background(), hex.EncodeToString(sum[:]))
	if err != nil || found.ID != linkID {
		t.Fatalf("found=%+v err=%v", found, err)
	}
	if err = repo.RevokeLink(context.Background(), domain.SceneRef{WorkspaceID: workspace, SceneID: scene}, session, linkID); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.FindLinkByHash(context.Background(), hex.EncodeToString(sum[:])); err == nil {
		t.Fatal("revoked link remained usable")
	}
	var memberships int64
	if err = db.Table("workspace_members").Where("workspace_id=? AND user_id=?", workspace, owner).Count(&memberships).Error; err != nil || memberships != 1 {
		t.Fatalf("membership=%d err=%v", memberships, err)
	}
}

func TestPresentationMigrationUpDown(t *testing.T) {
	db := accessTestDB(t, 19)
	up, err := os.ReadFile("../../../migrations/000020_presentation_links.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(up)).Error; err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable("presentation_links") {
		t.Fatal("presentation_links missing")
	}
	down, err := os.ReadFile("../../../migrations/000020_presentation_links.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(down)).Error; err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable("presentation_links") {
		t.Fatal("presentation_links survived down")
	}
}
