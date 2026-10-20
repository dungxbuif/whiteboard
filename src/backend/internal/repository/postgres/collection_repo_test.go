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

func TestCollectionRepositoryPrivacyReferencesAndOrder(t *testing.T) {
	db := accessTestDB(t, 22)
	owner, member, workspaceID := accessFixture(t, db)
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'editor')", workspaceID, member)
	mustExec(t, db, "UPDATE workspace_feature_policies SET collections_enabled=true WHERE workspace_id=?", workspaceID)
	sceneA, sceneB := uuid.New(), uuid.New()
	for index, sceneID := range []uuid.UUID{sceneA, sceneB} {
		mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,?)", sceneID, workspaceID, "Scene "+string(rune('A'+index)))
	}
	accessService, err := usecase.NewAccessService(NewAccessRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewCollectionRepository(db)
	features := NewFeatureRepository(db)
	service := usecase.NewCollectionService(repo, usecase.NewFeatureService(features, accessService), accessService)
	private, err := service.Create(context.Background(), domain.UserPrincipal(member), workspaceID, domain.CollectionCreateInput{RequestID: uuid.New(), Name: "Private", Visibility: domain.CollectionPrivate})
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := service.Create(context.Background(), domain.UserPrincipal(owner), workspaceID, domain.CollectionCreateInput{RequestID: uuid.New(), Name: "Workspace", Visibility: domain.CollectionWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.List(context.Background(), domain.UserPrincipal(owner), workspaceID, "", 50)
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != workspace.ID {
		t.Fatalf("owner private leak page=%+v err=%v", page, err)
	}
	if _, err = service.Get(context.Background(), domain.UserPrincipal(owner), workspaceID, private.ID); !errors.Is(err, domain.ErrInaccessible) {
		t.Fatalf("owner read private collection err=%v", err)
	}
	if err = service.Delete(context.Background(), domain.UserPrincipal(owner), workspaceID, private.ID, domain.CollectionRevisionInput{RequestID: uuid.New(), ExpectedRevision: private.Revision}); err != nil {
		t.Fatalf("owner moderation delete: %v", err)
	}
	workspace, err = service.AddScene(context.Background(), domain.UserPrincipal(owner), workspaceID, workspace.ID, domain.CollectionSceneInput{RequestID: uuid.New(), ExpectedRevision: workspace.Revision, SceneID: sceneA})
	if err != nil {
		t.Fatal(err)
	}
	workspace, err = service.AddScene(context.Background(), domain.UserPrincipal(owner), workspaceID, workspace.ID, domain.CollectionSceneInput{RequestID: uuid.New(), ExpectedRevision: workspace.Revision, SceneID: sceneB})
	if err != nil {
		t.Fatal(err)
	}
	workspace, err = service.Reorder(context.Background(), domain.UserPrincipal(owner), workspaceID, workspace.ID, domain.CollectionReorderInput{RequestID: uuid.New(), ExpectedRevision: workspace.Revision, SceneIDs: []uuid.UUID{sceneB, sceneA}})
	if err != nil || len(workspace.Scenes) != 2 || workspace.Scenes[0].ID != sceneB || workspace.Scenes[1].ID != sceneA {
		t.Fatalf("reorder=%+v err=%v", workspace.Scenes, err)
	}
	if _, err = service.Reorder(context.Background(), domain.UserPrincipal(owner), workspaceID, workspace.ID, domain.CollectionReorderInput{RequestID: uuid.New(), ExpectedRevision: workspace.Revision - 1, SceneIDs: []uuid.UUID{sceneA, sceneB}}); !errors.Is(err, domain.ErrRevisionConflict) {
		t.Fatalf("stale reorder err=%v", err)
	}
	if err = service.Delete(context.Background(), domain.UserPrincipal(owner), workspaceID, workspace.ID, domain.CollectionRevisionInput{RequestID: uuid.New(), ExpectedRevision: workspace.Revision}); err != nil {
		t.Fatal(err)
	}
	var scenes int64
	if err = db.Table("scenes").Where("id IN ?", []uuid.UUID{sceneA, sceneB}).Count(&scenes).Error; err != nil || scenes != 2 {
		t.Fatalf("collection deletion removed scenes count=%d err=%v", scenes, err)
	}
}

func TestCollectionMutationIdempotency(t *testing.T) {
	db := accessTestDB(t, 22)
	owner, _, workspaceID := accessFixture(t, db)
	mustExec(t, db, "UPDATE workspace_feature_policies SET collections_enabled=true WHERE workspace_id=?", workspaceID)
	accessService, err := usecase.NewAccessService(NewAccessRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	service := usecase.NewCollectionService(NewCollectionRepository(db), usecase.NewFeatureService(NewFeatureRepository(db), accessService), accessService)
	requestID := uuid.New()
	input := domain.CollectionCreateInput{RequestID: requestID, Name: "Retry", Visibility: domain.CollectionWorkspace}
	first, err := service.Create(context.Background(), domain.UserPrincipal(owner), workspaceID, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(context.Background(), domain.UserPrincipal(owner), workspaceID, input)
	if err != nil || second.ID != first.ID || second.Revision != first.Revision {
		t.Fatalf("retry first=%+v second=%+v err=%v", first, second, err)
	}
	input.Name = "Changed"
	if _, err = service.Create(context.Background(), domain.UserPrincipal(owner), workspaceID, input); !errors.Is(err, domain.ErrRequestIDReused) {
		t.Fatalf("changed retry err=%v", err)
	}
	var count int64
	if err = db.Table("collections").Where("workspace_id=?", workspaceID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("idempotent collection count=%d err=%v", count, err)
	}
}

func TestCollectionMigrationUpDown(t *testing.T) {
	db := accessTestDB(t, 21)
	up, err := os.ReadFile("../../../migrations/000022_collections.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(up)).Error; err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable("collections") || !db.Migrator().HasTable("collection_scenes") {
		t.Fatal("collection tables missing")
	}
	down, err := os.ReadFile("../../../migrations/000022_collections.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(down)).Error; err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable("collections") || db.Migrator().HasTable("collection_scenes") {
		t.Fatal("collection tables survived down")
	}
}

func TestCollectionForeignWorkspaceRejected(t *testing.T) {
	db := accessTestDB(t, 22)
	owner, _, workspaceID := accessFixture(t, db)
	otherOwner, otherWorkspace := uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO users(id,email,name) VALUES (?,?,'Other')", otherOwner, otherOwner.String()+"@example.test")
	mustExec(t, db, "INSERT INTO workspaces(id,name,owner_id) VALUES (?,'Other',?)", otherWorkspace, otherOwner)
	mustExec(t, db, "UPDATE workspace_feature_policies SET collections_enabled=true WHERE workspace_id IN (?,?)", workspaceID, otherWorkspace)
	foreignScene := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Foreign')", foreignScene, otherWorkspace)
	accessService, err := usecase.NewAccessService(NewAccessRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	service := usecase.NewCollectionService(NewCollectionRepository(db), usecase.NewFeatureService(NewFeatureRepository(db), accessService), accessService)
	collection, err := service.Create(context.Background(), domain.UserPrincipal(owner), workspaceID, domain.CollectionCreateInput{RequestID: uuid.New(), Name: "Workspace", Visibility: domain.CollectionWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.AddScene(context.Background(), domain.UserPrincipal(owner), workspaceID, collection.ID, domain.CollectionSceneInput{RequestID: uuid.New(), ExpectedRevision: collection.Revision, SceneID: foreignScene}); err == nil {
		t.Fatal("foreign scene was added")
	}
}
