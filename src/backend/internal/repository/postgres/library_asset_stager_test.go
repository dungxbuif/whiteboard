package postgres

import (
	"context"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type libraryCopyStub struct {
	calls, deletes int
	deleteErr      error
}

func (s *libraryCopyStub) CopyObject(domain.Context, string, string, string, string) error {
	s.calls++
	return nil
}
func (s *libraryCopyStub) Delete(domain.Context, string, string) error {
	s.deletes++
	return s.deleteErr
}

func TestSceneCopyStagesImageAndRewritesReferences(t *testing.T) {
	db := accessTestDB(t, 999)
	owner, _, workspace := accessFixture(t, db)
	source, target, asset, requestID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for index, scene := range []uuid.UUID{source, target} {
		if err := db.Exec(`INSERT INTO scenes(id,workspace_id,title,content,created_by) VALUES(?,?,?,?::jsonb,?)`, scene, workspace, "Scene "+string(rune('A'+index)), `{"elements":[],"appState":{},"files":{}}`, owner).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec("UPDATE scenes SET pending_duplicate=TRUE, deleted_at=NOW() WHERE id=?", target).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO scene_assets(id,workspace_id,scene_id,uploaded_by,uploaded_principal_id,uploaded_principal_type,asset_kind,file_id,storage_bucket,storage_key,mime_type,byte_size) VALUES(?,?,?,?,?,'user','excalidraw_file','image','bucket','source','image/png',7)`, asset, workspace, source, owner, owner.String()).Error; err != nil {
		t.Fatal(err)
	}
	// Older descriptors may omit assetId even though a scene-scoped row exists.
	content := json.RawMessage(`{"elements":[{"id":"shape","type":"image","fileId":"image"}],"appState":{},"files":{"image":{"id":"image","mimeType":"image/png"}}}`)
	copier := &libraryCopyStub{}
	stager := NewLibraryAssetStager(db, copier, time.Hour)
	updated, err := stager.StageSceneCopy(context.Background(), domain.UserPrincipal(owner), domain.SceneRef{WorkspaceID: workspace, SceneID: source}, domain.SceneRef{WorkspaceID: workspace, SceneID: target}, requestID, content)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Elements []struct {
			FileID string `json:"fileId"`
		} `json:"elements"`
		Files map[string]struct {
			ID      string    `json:"id"`
			AssetID uuid.UUID `json:"assetId"`
		} `json:"files"`
	}
	if err := json.Unmarshal(updated, &result); err != nil {
		t.Fatal(err)
	}
	newFile := result.Elements[0].FileID
	if newFile == "image" || result.Files[newFile].ID != newFile || result.Files[newFile].AssetID == asset || copier.calls != 1 {
		t.Fatalf("file=%s content=%s copies=%d", newFile, updated, copier.calls)
	}
	var count int64
	db.Table("scene_assets").Where("workspace_id=? AND scene_id=? AND id=? AND file_id=?", workspace, target, result.Files[newFile].AssetID, newFile).Count(&count)
	if count != 1 {
		t.Fatalf("target asset rows=%d", count)
	}
	copier.deleteErr = errors.New("storage unavailable")
	claimed, err := NewSceneRepository(db).ClaimDuplicateCleanup(context.Background(), workspace, target)
	if err != nil || !claimed {
		t.Fatalf("cleanup claim=%v err=%v", claimed, err)
	}
	if err := stager.CleanupSceneCopy(context.Background(), domain.SceneRef{WorkspaceID: workspace, SceneID: target}, requestID); err == nil {
		t.Fatal("expected object cleanup failure")
	}
	db.Table("scene_assets").Where("workspace_id=? AND scene_id=?", workspace, target).Count(&count)
	if count != 1 {
		t.Fatalf("copied row removed before object cleanup: %d", count)
	}
	copier.deleteErr = nil
	if err := stager.CleanupSceneCopy(context.Background(), domain.SceneRef{WorkspaceID: workspace, SceneID: target}, requestID); err != nil {
		t.Fatal(err)
	}
	db.Table("scene_assets").Where("workspace_id=? AND scene_id=?", workspace, target).Count(&count)
	if count != 0 || copier.deletes != 2 {
		t.Fatalf("target rows=%d object deletions=%d", count, copier.deletes)
	}
}

func TestSceneCopyRejectsInlineFileWithoutSceneAsset(t *testing.T) {
	db := accessTestDB(t, 999)
	owner, _, workspace := accessFixture(t, db)
	source, target := uuid.New(), uuid.New()
	for index, scene := range []uuid.UUID{source, target} {
		mustExec(t, db, `INSERT INTO scenes(id,workspace_id,title,content,created_by) VALUES(?,?,?,?::jsonb,?)`, scene, workspace, "Inline "+string(rune('A'+index)), `{"elements":[],"appState":{},"files":{}}`, owner)
	}
	content := json.RawMessage(`{"elements":[{"id":"image","type":"image","fileId":"file"}],"appState":{},"files":{"file":{"id":"file","mimeType":"image/png","dataURL":"data:image/png;base64,AAAA"}}}`)
	stager := NewLibraryAssetStager(db, &libraryCopyStub{}, time.Hour)
	if _, err := stager.StageSceneCopy(context.Background(), domain.UserPrincipal(owner), domain.SceneRef{WorkspaceID: workspace, SceneID: source}, domain.SceneRef{WorkspaceID: workspace, SceneID: target}, uuid.New(), content); !errors.Is(err, domain.ErrInvalidLibrary) {
		t.Fatalf("copy must reject image without scene-scoped asset: %v", err)
	}
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
