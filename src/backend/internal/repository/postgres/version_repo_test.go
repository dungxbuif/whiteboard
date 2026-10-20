package postgres

import (
	"context"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestVersionRepositoryReadsContentAndGuardsMetadataRevision(t *testing.T) {
	db := accessTestDB(t, 18)
	owner, _, workspace := accessFixture(t, db)
	scene, document, version := uuid.New(), uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title,content) VALUES (?,?,'Scene','{}')", scene, workspace)
	content := `{"elements":[],"appState":{},"files":{}}`
	mustExec(t, db, "INSERT INTO scene_versions(id,workspace_id,scene_id,document_id,source_sequence,content,content_bytes,content_hash,byte_size,label,pinned,kind,creator_kind,creator_id,request_id,payload_hash) VALUES (?,?,?,?,7,?::jsonb,?,repeat('a',64),?,'Before',false,'manual','user',?,?,repeat('b',64))", version, workspace, scene, document, content, []byte(content), len(content), owner.String(), uuid.NewString())
	repo := NewVersionRepository(db)
	ref := domain.SceneRef{WorkspaceID: workspace, SceneID: scene}
	got, err := repo.Version(context.Background(), ref, version)
	if err != nil || got.DocumentID != document || got.Sequence != "7" || !json.Valid(got.Content) {
		t.Fatalf("version=%+v err=%v", got, err)
	}
	access, err := usecase.NewAccessService(NewAccessRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	label := "Launch"
	pinned := true
	err = access.WithWorkspaceLock(context.Background(), workspace, owner, func(ctx domain.Context) error {
		updated, e := repo.SetVersionMetadata(ctx, ref, version, 1, &label, &pinned)
		if e != nil {
			return e
		}
		if updated.MetadataRevision != 2 || updated.Label == nil || *updated.Label != label || !updated.Pinned {
			t.Fatalf("updated=%+v", updated)
		}
		_, e = repo.SetVersionMetadata(ctx, ref, version, 1, &label, &pinned)
		return e
	})
	if !errors.Is(err, domain.ErrRevisionConflict) {
		t.Fatalf("stale metadata err=%v", err)
	}
	var sequence string
	db.Raw("SELECT source_sequence::text FROM scene_versions WHERE id=?", version).Scan(&sequence)
	if sequence != "7" {
		t.Fatalf("immutable sequence changed: %s", sequence)
	}
	if err := db.Exec("UPDATE scene_versions SET content='{}'::jsonb WHERE id=?", version).Error; err == nil {
		t.Fatal("immutable version content accepted an update")
	}
}
