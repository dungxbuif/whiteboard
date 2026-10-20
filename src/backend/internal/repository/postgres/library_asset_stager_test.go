package postgres

import (
	"context"
	"devtool/internal/domain"
	"testing"
	"time"

	"github.com/google/uuid"
)

type libraryCopyStub struct{ calls int }

func (s *libraryCopyStub) CopyObject(domain.Context, string, string, string, string) error {
	s.calls++
	return nil
}

func TestLibraryAssetPinRaceStagesOneIdempotentTarget(t *testing.T) {
	db := accessTestDB(t, 999)
	owner, _, workspace := accessFixture(t, db)
	source, target, asset, requestID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for index, scene := range []uuid.UUID{source, target} {
		if err := db.Exec(`INSERT INTO scenes(id,workspace_id,title,content,created_by) VALUES(?,?,?,?::jsonb,?)`, scene, workspace, "Scene "+string(rune('A'+index)), `{"elements":[],"appState":{},"files":{}}`, owner).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec(`INSERT INTO scene_assets(id,workspace_id,scene_id,uploaded_by,uploaded_principal_id,uploaded_principal_type,asset_kind,file_id,storage_bucket,storage_key,mime_type,byte_size) VALUES(?,?,?,?,?,'user','excalidraw_file','old','bucket','source','image/png',7)`, asset, workspace, source, owner, owner.String()).Error; err != nil {
		t.Fatal(err)
	}
	copier := &libraryCopyStub{}
	stager := NewLibraryAssetStager(db, copier, time.Hour)
	pin := domain.LibraryAssetPin{SourceSceneID: source, AssetID: asset, FileID: "old"}
	ref := domain.SceneRef{WorkspaceID: workspace, SceneID: target}
	principal := domain.UserPrincipal(owner)
	first, err := stager.Stage(context.Background(), principal, ref, requestID, []domain.LibraryAssetPin{pin})
	if err != nil {
		t.Fatal(err)
	}
	second, err := stager.Stage(context.Background(), principal, ref, requestID, []domain.LibraryAssetPin{pin})
	if err != nil {
		t.Fatal(err)
	}
	if first["old"] != second["old"] || copier.calls != 1 {
		t.Fatalf("first=%v second=%v copies=%d", first, second, copier.calls)
	}
	var stages, assets int64
	db.Table("library_asset_stages").Where("request_id=? AND state='ready'", requestID).Count(&stages)
	db.Table("scene_assets").Where("scene_id=? AND id=?", target, first["old"]).Count(&assets)
	if stages != 1 || assets != 1 {
		t.Fatalf("stages=%d assets=%d", stages, assets)
	}
}
