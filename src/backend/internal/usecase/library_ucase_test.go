package usecase

import (
	"context"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type libraryAccessStub struct {
	workspace domain.AccessDecision
	scenes    map[uuid.UUID]domain.AccessDecision
	err       map[uuid.UUID]error
}

func (s libraryAccessStub) Resolve(_ domain.Context, _ domain.Principal, ref domain.SceneRef) (domain.AccessDecision, error) {
	if err := s.err[ref.SceneID]; err != nil {
		return domain.AccessDecision{}, err
	}
	return s.scenes[ref.SceneID], nil
}
func (s libraryAccessStub) ResolveWorkspace(domain.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error) {
	return s.workspace, nil
}
func (libraryAccessStub) WithWorkspaceLock(ctx domain.Context, _ uuid.UUID, _ uuid.UUID, fn func(domain.Context) error) error {
	return fn(ctx)
}

type libraryFeatureStub struct{ enabled bool }

func (s libraryFeatureStub) Policy(domain.Context, domain.Principal, uuid.UUID) (domain.WorkspaceFeaturePolicy, error) {
	return domain.WorkspaceFeaturePolicy{LibrariesEnabled: s.enabled}, nil
}

type libraryRepoStub struct {
	value    domain.LibraryItem
	revision domain.LibraryRevision
	saved    bool
	pins     []domain.LibraryAssetPin
}

func (s *libraryRepoStub) CreateRevision(_ domain.Context, item domain.LibraryItem, revision domain.LibraryRevision, pins []domain.LibraryAssetPin) (domain.LibraryItem, domain.LibraryRevision, error) {
	s.saved = true
	s.value = item
	s.revision = revision
	s.pins = pins
	return item, revision, nil
}
func (s *libraryRepoStub) Item(domain.Context, uuid.UUID, uuid.UUID) (domain.LibraryItem, error) {
	return s.value, nil
}
func (s *libraryRepoStub) Revision(domain.Context, uuid.UUID, uuid.UUID, int64) (domain.LibraryRevision, error) {
	return s.revision, nil
}
func (s *libraryRepoStub) List(domain.Context, uuid.UUID, string, int) ([]domain.LibraryItem, error) {
	return []domain.LibraryItem{s.value}, nil
}
func (s *libraryRepoStub) RevisionAssets(domain.Context, uuid.UUID, uuid.UUID, int64) ([]domain.LibraryAssetPin, error) {
	return s.pins, nil
}

type libraryStagerStub struct {
	called      int
	replacement map[string]uuid.UUID
}

func (s *libraryStagerStub) Stage(_ domain.Context, _ domain.Principal, _ domain.SceneRef, _ uuid.UUID, _ []domain.LibraryAssetPin) (map[string]uuid.UUID, error) {
	s.called++
	return s.replacement, nil
}

type libraryCapsStub struct{ action string }

func (s *libraryCapsStub) AuthorizeScene(domain.Context, domain.Principal, domain.SceneRef, time.Duration) (time.Time, error) {
	panic("unused")
}
func (s *libraryCapsStub) AuthorizeCommand(_ domain.Context, _ domain.Principal, m domain.SceneMutation) (domain.SignedGrant, error) {
	s.action = m.Action
	return domain.SignedGrant{Token: "grant"}, nil
}
func (s *libraryCapsStub) CreateAssetCapability(domain.Context, domain.Principal, domain.SceneRef, []domain.AssetUploadRequest) (domain.AssetGrant, error) {
	panic("unused")
}

type libraryRemoteStub struct{ called bool }

func (s *libraryRemoteStub) Call(_ domain.Context, operation string, _ domain.SceneRef, _ map[string]any, _ int64) (json.RawMessage, error) {
	s.called = operation == "library-insertions"
	return json.RawMessage(`{"requestId":"11111111-1111-4111-8111-111111111111","sequence":"8","stateVector":"vector","elementIds":["new"]}`), nil
}

func validLibraryJSON() json.RawMessage {
	return json.RawMessage(`{"type":"excalidrawlib","version":2,"libraryItems":[{"id":"item","status":"published","created":1,"elements":[],"files":{}}]}`)
}

func libraryJSONWithAsset(asset uuid.UUID) json.RawMessage {
	return json.RawMessage(`{"type":"excalidrawlib","version":2,"libraryItems":[{"id":"item","status":"published","created":1,"elements":[],"files":{"file":{"id":"file","mimeType":"image/png","created":1,"assetId":"` + asset.String() + `"}}}]}`)
}

func TestLibraryOwnerOnlyWritesAndMembershipReads(t *testing.T) {
	w, u := uuid.New(), uuid.New()
	repo := &libraryRepoStub{}
	member := NewLibraryService(repo, libraryFeatureStub{true}, libraryAccessStub{workspace: domain.AccessDecision{Access: domain.AccessEdit}}, nil, nil)
	_, _, err := member.Import(context.Background(), domain.UserPrincipal(u), w, domain.LibraryImportInput{RequestID: uuid.New(), Name: "Flow", Content: validLibraryJSON()})
	if !errors.Is(err, domain.ErrAccessDenied) || repo.saved {
		t.Fatalf("member publish err=%v saved=%v", err, repo.saved)
	}
	repo.value = domain.LibraryItem{ID: uuid.New(), WorkspaceID: w, Name: "Flow", CurrentRevision: 1}
	if _, err = member.List(context.Background(), domain.UserPrincipal(u), w, "", 50); err != nil {
		t.Fatalf("member list: %v", err)
	}
	owner := NewLibraryService(repo, libraryFeatureStub{true}, libraryAccessStub{workspace: domain.AccessDecision{Access: domain.AccessManage, Owner: true}}, nil, nil)
	if _, _, err = owner.Import(context.Background(), domain.UserPrincipal(u), w, domain.LibraryImportInput{RequestID: uuid.New(), Name: "Flow", Content: validLibraryJSON()}); err != nil || !repo.saved {
		t.Fatalf("owner import err=%v saved=%v", err, repo.saved)
	}
}

func TestLibraryForeignAssetDenied(t *testing.T) {
	w, u, source := uuid.New(), uuid.New(), uuid.New()
	repo := &libraryRepoStub{}
	access := libraryAccessStub{workspace: domain.AccessDecision{Access: domain.AccessManage, Owner: true}, err: map[uuid.UUID]error{source: domain.ErrInaccessible}}
	service := NewLibraryService(repo, libraryFeatureStub{true}, access, nil, nil)
	asset := uuid.New()
	_, _, err := service.Import(context.Background(), domain.UserPrincipal(u), w, domain.LibraryImportInput{RequestID: uuid.New(), Name: "Flow", Content: libraryJSONWithAsset(asset), Assets: []domain.LibraryAssetPin{{SourceSceneID: source, AssetID: asset, FileID: "file"}}})
	if !errors.Is(err, domain.ErrInaccessible) || repo.saved {
		t.Fatalf("foreign asset err=%v saved=%v", err, repo.saved)
	}
}

func TestLibraryInsertRequiresCurrentTargetEditAndDurableReceipt(t *testing.T) {
	w, u, scene, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repo := &libraryRepoStub{value: domain.LibraryItem{ID: itemID, WorkspaceID: w, Name: "Flow", CurrentRevision: 1}, revision: domain.LibraryRevision{LibraryItemID: itemID, WorkspaceID: w, Revision: 1, Content: json.RawMessage(`{"id":"item","status":"published","created":1,"elements":[],"files":{}}`)}}
	caps, remote := &libraryCapsStub{}, &libraryRemoteStub{}
	access := libraryAccessStub{workspace: domain.AccessDecision{Access: domain.AccessView}, scenes: map[uuid.UUID]domain.AccessDecision{scene: {Access: domain.AccessEdit}}}
	service := NewLibraryService(repo, libraryFeatureStub{true}, access, caps, remote)
	result, err := service.Insert(context.Background(), domain.UserPrincipal(u), domain.SceneRef{WorkspaceID: w, SceneID: scene}, domain.LibraryInsertInput{LibraryItemID: itemID, Revision: 1, ExpectedSequence: "7", RequestID: uuid.MustParse("11111111-1111-4111-8111-111111111111")})
	if err != nil || result.Sequence != "8" || caps.action != domain.ActionLibraryInsert || !remote.called {
		t.Fatalf("insert=%+v err=%v action=%s called=%v", result, err, caps.action, remote.called)
	}
	access.scenes[scene] = domain.AccessDecision{Access: domain.AccessView}
	service = NewLibraryService(repo, libraryFeatureStub{true}, access, caps, remote)
	if _, err = service.Insert(context.Background(), domain.UserPrincipal(u), domain.SceneRef{WorkspaceID: w, SceneID: scene}, domain.LibraryInsertInput{LibraryItemID: itemID, Revision: 1, ExpectedSequence: "8", RequestID: uuid.New()}); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("view insert err=%v", err)
	}
}

func TestLibraryAssetPinRaceStagesIdempotentlyBeforeWriter(t *testing.T) {
	w, u, scene, itemID, source, asset := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	content := json.RawMessage(`{"type":"excalidrawlib","version":2,"libraryItems":[{"id":"item","status":"published","created":1,"elements":[],"files":{"old":{"id":"old","mimeType":"image/png","created":1,"assetId":"` + asset.String() + `"}}}]}`)
	repo := &libraryRepoStub{revision: domain.LibraryRevision{LibraryItemID: itemID, WorkspaceID: w, Revision: 1, Content: content}, pins: []domain.LibraryAssetPin{{SourceSceneID: source, AssetID: asset, FileID: "old"}}}
	stager := &libraryStagerStub{replacement: map[string]uuid.UUID{"old": uuid.New()}}
	service := NewLibraryService(repo, libraryFeatureStub{true}, libraryAccessStub{workspace: domain.AccessDecision{Access: domain.AccessView}, scenes: map[uuid.UUID]domain.AccessDecision{scene: {Access: domain.AccessEdit}}}, &libraryCapsStub{}, &libraryRemoteStub{})
	service.SetAssetStager(stager)
	for range 2 {
		if _, err := service.Insert(context.Background(), domain.UserPrincipal(u), domain.SceneRef{WorkspaceID: w, SceneID: scene}, domain.LibraryInsertInput{LibraryItemID: itemID, Revision: 1, ExpectedSequence: "7", RequestID: uuid.MustParse("11111111-1111-4111-8111-111111111111")}); err != nil {
			t.Fatal(err)
		}
	}
	if stager.called != 2 {
		t.Fatalf("stage retries=%d", stager.called)
	}
}

func TestLibraryRejectsUnsupportedNativePayloadBeforeRepository(t *testing.T) {
	w, u := uuid.New(), uuid.New()
	repo := &libraryRepoStub{}
	service := NewLibraryService(repo, libraryFeatureStub{true}, libraryAccessStub{workspace: domain.AccessDecision{Access: domain.AccessManage, Owner: true}}, nil, nil)
	unsafe := json.RawMessage(`{"type":"excalidrawlib","version":2,"source":"https://user:secret@example.test/library","libraryItems":[{"id":"item","status":"published","created":1,"elements":[{"id":"embed","type":"iframe"}],"files":{}}]}`)
	if _, _, err := service.Import(context.Background(), domain.UserPrincipal(u), w, domain.LibraryImportInput{RequestID: uuid.New(), Name: "Unsafe", Content: unsafe}); !errors.Is(err, domain.ErrInvalidLibrary) || repo.saved {
		t.Fatalf("unsafe import err=%v saved=%v", err, repo.saved)
	}
}
