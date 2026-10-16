package postgres

import (
	"context"
	"devtool/internal/domain"
	"encoding/json"
	"github.com/google/uuid"
	"os"
	"testing"
)

func TestV2RepositoryRejectsStaleLegacyWriter(t *testing.T) {
	db := accessTestDB(t, 8)
	migration, e := os.ReadFile("../../../migrations/000009_collab_documents.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, db, string(migration))
	owner, _, wid := accessFixture(t, db)
	_ = owner
	sid := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title,content) VALUES(?,?,?,?)", sid, wid, "Existing", `{}`)
	repo := NewSceneRepository(db)
	scene, e := repo.FindByID(context.Background(), wid, sid)
	if e != nil {
		t.Fatal(e)
	}
	scene.ContentPatch = true
	scene.Content = json.RawMessage(`{"legacy":true}`)
	mustExec(t, db, "UPDATE scenes SET writer_mode='v2' WHERE id=?", sid)
	if e = repo.Update(context.Background(), scene); e == nil || e.Error() != "COLLAB_REQUIRED" {
		t.Fatalf("stale writer accepted: %v", e)
	}
	scene.ContentPatch = false
	scene.Title = "Metadata updated"
	if e = repo.Update(context.Background(), scene); e != nil {
		t.Fatal(e)
	}
	var persisted domain.Scene
	if e = db.First(&persisted, "id = ?", sid).Error; e != nil {
		t.Fatal(e)
	}
	if string(persisted.Content) == string(scene.Content) || persisted.WriterMode != "v2" {
		t.Fatal("metadata changed durable content/mode")
	}
	// New scenes use the database's legacy mode default.
	fresh := &domain.Scene{ID: uuid.New(), WorkspaceID: wid, Title: "Fresh", Content: json.RawMessage(`{}`)}
	if e = repo.Create(context.Background(), fresh); e != nil {
		t.Fatal(e)
	}
}
