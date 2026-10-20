package usecase

import (
	"context"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type versionFeatureStub struct {
	err    error
	action domain.FeatureAction
}

func (s *versionFeatureStub) Require(_ domain.Context, _ domain.Principal, _ domain.SceneRef, a domain.FeatureAction) error {
	s.action = a
	return s.err
}
func (*versionFeatureStub) Policy(domain.Context, domain.Principal, uuid.UUID) (domain.WorkspaceFeaturePolicy, error) {
	return domain.WorkspaceFeaturePolicy{}, nil
}
func (*versionFeatureStub) UpdatePolicy(domain.Context, domain.Principal, uuid.UUID, int64, uuid.UUID, domain.WorkspaceFeaturePolicyPatch) (domain.WorkspaceFeaturePolicy, error) {
	return domain.WorkspaceFeaturePolicy{}, nil
}
func (s *versionFeatureStub) Mutate(ctx domain.Context, _ domain.Principal, _ domain.SceneRef, a domain.FeatureAction, _ uuid.UUID, _ []byte, fn func(domain.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	s.action = a
	if s.err != nil {
		return domain.StoredFeatureResponse{}, s.err
	}
	return fn(ctx)
}

type versionRepoStub struct {
	listed  bool
	version domain.SceneVersion
}

func (s *versionRepoStub) ListVersions(domain.Context, domain.SceneRef, *uuid.UUID, int) ([]domain.SceneVersion, error) {
	s.listed = true
	return nil, nil
}
func (s *versionRepoStub) Version(domain.Context, domain.SceneRef, uuid.UUID) (domain.SceneVersion, error) {
	return s.version, nil
}
func (s *versionRepoStub) SetVersionMetadata(domain.Context, domain.SceneRef, uuid.UUID, int64, *string, *bool) (domain.SceneVersion, error) {
	return s.version, nil
}

type versionCapsStub struct{ mutation domain.SceneMutation }

func (s *versionCapsStub) AuthorizeScene(domain.Context, domain.Principal, domain.SceneRef, time.Duration) (time.Time, error) {
	return time.Time{}, nil
}
func (s *versionCapsStub) AuthorizeCommand(_ domain.Context, _ domain.Principal, m domain.SceneMutation) (domain.SignedGrant, error) {
	s.mutation = m
	return domain.SignedGrant{Token: "signed"}, nil
}
func (*versionCapsStub) CreateAssetCapability(domain.Context, domain.Principal, domain.SceneRef, []domain.AssetUploadRequest) (domain.AssetGrant, error) {
	return domain.AssetGrant{}, nil
}

type versionRemoteStub struct {
	operation string
	envelope  map[string]any
	response  json.RawMessage
	limit     int64
}

type duplicateSceneRepoStub struct {
	domain.SceneRepository
	source                 *domain.Scene
	created                *domain.Scene
	deleted                bool
	published              bool
	cleanupContextCanceled bool
	publishErr             error
}

func (s *duplicateSceneRepoStub) FindByID(_ domain.Context, _ uuid.UUID, id uuid.UUID) (*domain.Scene, error) {
	if s.created != nil && s.created.ID == id {
		return s.created, nil
	}
	return s.source, nil
}
func (s *duplicateSceneRepoStub) Create(_ domain.Context, scene *domain.Scene) error {
	s.created = scene
	return nil
}
func (s *duplicateSceneRepoStub) PublishDuplicate(_ domain.Context, scene *domain.Scene) error {
	s.published = true
	if s.publishErr != nil {
		return s.publishErr
	}
	s.created = scene
	return nil
}
func (s *duplicateSceneRepoStub) HardDelete(ctx domain.Context, _ uuid.UUID, _ uuid.UUID) error {
	s.cleanupContextCanceled = ctx.Err() != nil
	s.deleted = true
	return nil
}

type duplicateAccessStub struct{ domain.AccessService }

func (duplicateAccessStub) Resolve(_ domain.Context, _ domain.Principal, _ domain.SceneRef) (domain.AccessDecision, error) {
	return domain.AccessDecision{Access: domain.AccessEdit}, nil
}
func (duplicateAccessStub) ResolveWorkspace(_ domain.Context, _ domain.Principal, _ uuid.UUID) (domain.AccessDecision, error) {
	return domain.AccessDecision{Access: domain.AccessEdit}, nil
}

type duplicateAssetStagerStub struct {
	called, cleaned, cleanupContextCanceled bool
	stageErr, cleanupErr                    error
	onStage                                 func()
}

func (s *duplicateAssetStagerStub) StageSceneCopy(_ domain.Context, _ domain.Principal, _, _ domain.SceneRef, _ uuid.UUID, content json.RawMessage) (json.RawMessage, error) {
	s.called = true
	if s.onStage != nil {
		s.onStage()
	}
	return content, s.stageErr
}
func (s *duplicateAssetStagerStub) CleanupSceneCopy(ctx domain.Context, _ domain.SceneRef, _ uuid.UUID) error {
	s.cleaned = true
	s.cleanupContextCanceled = ctx.Err() != nil
	return s.cleanupErr
}

func TestDuplicateExportsLargeDurableContentBeforeCreatingScene(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	user := uuid.New()
	large := strings.Repeat("x", 300*1024)
	content := json.RawMessage(`{"elements":[],"appState":{"memo":"` + large + `"},"files":{}}`)
	projection, _ := json.Marshal(map[string]any{"content": content, "sequence": "2"})
	source := &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, Title: "Source", WriterMode: "v2", Content: json.RawMessage(`{"elements":[],"appState":{},"files":{}}`)}
	scenes := &duplicateSceneRepoStub{source: source}
	remote := &versionRemoteStub{response: projection}
	assets := &duplicateAssetStagerStub{}
	service := NewVersionService(nil, nil, &versionCapsStub{}, remote)
	service.SetDuplicateDependencies(scenes, nil, assets, duplicateAccessStub{})
	copy, err := service.Duplicate(context.Background(), domain.UserPrincipal(user), ref, "Copy", nil)
	if err != nil || copy == nil || remote.operation != "export-projection" || remote.limit <= 256*1024 || !assets.called || len(copy.Content) <= 256*1024 {
		t.Fatalf("copy=%v err=%v operation=%s limit=%d staged=%v", copy, err, remote.operation, remote.limit, assets.called)
	}
}

func TestDuplicateFailureCleansHiddenSceneWithIndependentContext(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	user := uuid.New()
	source := &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: json.RawMessage(`{"elements":[],"appState":{},"files":{}}`)}
	scenes := &duplicateSceneRepoStub{source: source}
	stageFailure := errors.New("copy failed")
	assets := &duplicateAssetStagerStub{stageErr: stageFailure}
	service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
	service.SetDuplicateDependencies(scenes, nil, assets, duplicateAccessStub{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Simulate the client disconnecting while external object staging fails.
	assets.onStage = cancel
	_, err := service.Duplicate(ctx, domain.UserPrincipal(user), ref, "Copy", nil)
	if !errors.Is(err, stageFailure) || !assets.cleaned || assets.cleanupContextCanceled || !scenes.deleted || scenes.cleanupContextCanceled || !scenes.created.DeletedAt.Valid {
		t.Fatalf("err=%v assets=%+v scenes=%+v", err, assets, scenes)
	}
}

func TestDuplicateCleanupFailureRetainsHiddenScene(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: json.RawMessage(`{"elements":[],"appState":{},"files":{}}`)}}
	assets := &duplicateAssetStagerStub{stageErr: errors.New("stage failed"), cleanupErr: errors.New("delete object failed")}
	service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
	service.SetDuplicateDependencies(scenes, nil, assets, duplicateAccessStub{})
	_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
	if err == nil || !strings.Contains(err.Error(), "delete object failed") || scenes.deleted || !scenes.created.DeletedAt.Valid {
		t.Fatalf("err=%v hidden=%v deleted=%v", err, scenes.created, scenes.deleted)
	}
}

func TestDuplicatePublishFailureCompensatesStagedAssets(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: json.RawMessage(`{"elements":[],"appState":{},"files":{}}`)}, publishErr: errors.New("publish failed")}
	assets := &duplicateAssetStagerStub{}
	service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
	service.SetDuplicateDependencies(scenes, nil, assets, duplicateAccessStub{})
	_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
	if err == nil || !strings.Contains(err.Error(), "publish failed") || !assets.called || !assets.cleaned || !scenes.deleted || scenes.cleanupContextCanceled {
		t.Fatalf("err=%v assets=%+v scenes=%+v", err, assets, scenes)
	}
}

func TestDuplicateCancellationAfterStagingDoesNotPublish(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: json.RawMessage(`{"elements":[],"appState":{},"files":{}}`)}}
	assets := &duplicateAssetStagerStub{}
	service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
	service.SetDuplicateDependencies(scenes, nil, assets, duplicateAccessStub{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	assets.onStage = cancel
	_, err := service.Duplicate(ctx, domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
	if !errors.Is(err, context.Canceled) || !assets.cleaned || !scenes.deleted || scenes.published || !scenes.created.DeletedAt.Valid {
		t.Fatalf("err=%v assets=%+v scenes=%+v", err, assets, scenes)
	}
}

func TestCurrentProjectionReadsUncompactedDurableContent(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	caps := &versionCapsStub{}
	remote := &versionRemoteStub{response: json.RawMessage(`{"sequence":"2","content":{"elements":[{"id":"recent"}],"appState":{},"files":{}}}`)}
	service := NewVersionService(nil, nil, caps, remote)
	raw, err := service.CurrentProjection(context.Background(), domain.UserPrincipal(uuid.New()), ref)
	if err != nil || remote.operation != "projection" || caps.mutation.Action != domain.ActionSceneRead || !strings.Contains(string(raw), `"id":"recent"`) {
		t.Fatalf("operation=%s action=%s raw=%s err=%v", remote.operation, caps.mutation.Action, raw, err)
	}
}

func (s *versionRemoteStub) Call(_ domain.Context, operation string, _ domain.SceneRef, envelope map[string]any, limit int64) (json.RawMessage, error) {
	s.operation, s.envelope, s.limit = operation, envelope, limit
	return s.response, nil
}

type versionAssetViewerStub struct{ files []string }

func (s *versionAssetViewerStub) PresignScopedView(_ domain.Context, _ domain.SceneRef, files []string, _ time.Time) (map[string]domain.AssetViewResponse, error) {
	s.files = files
	return map[string]domain.AssetViewResponse{"file-1": {AssetID: uuid.New(), ViewUrl: "https://assets.test/file-1", MimeType: "image/png", ExpiresAt: time.Now().Add(time.Minute)}}, nil
}

type versionEventStub struct{ input domain.EventInput }

func (s *versionEventStub) AppendEvent(_ domain.Context, input domain.EventInput) (domain.WorkspaceEvent, error) {
	s.input = input
	return domain.WorkspaceEvent{ID: "1"}, nil
}

func TestGuestHistoryDenied(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	repo := &versionRepoStub{}
	feature := &versionFeatureStub{err: domain.ErrAccessDenied}
	service := NewVersionService(repo, feature, nil, nil)
	guest := domain.Principal{Kind: domain.PrincipalPublic, ID: "guest:test", GrantID: uuid.NewString(), GrantVersion: "1"}
	if _, err := service.List(context.Background(), guest, ref, "", 50); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("guest history err=%v", err)
	}
	if repo.listed {
		t.Fatal("guest denial reached version repository")
	}
}

func TestVersionCreateBindsSequenceAndPayload(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	principal := domain.UserPrincipal(uuid.New())
	requestID := uuid.New()
	versionID, documentID := uuid.New(), uuid.New()
	label := "<Launch> &"
	feature, caps := &versionFeatureStub{}, &versionCapsStub{}
	remote := &versionRemoteStub{response: json.RawMessage(`{"id":"` + versionID.String() + `","documentId":"` + documentID.String() + `","sequence":"42","hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","pinned":true,"kind":"manual","creator":{"kind":"user","id":"` + principal.ID + `"},"createdAt":"2026-10-20T00:00:00Z"}`)}
	service := NewVersionService(&versionRepoStub{}, feature, caps, remote)
	out, err := service.Create(context.Background(), principal, ref, domain.CreateVersionInput{RequestID: requestID, ExpectedSequence: "42", Label: &label, Pinned: true})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != versionID || feature.action != domain.FeatureVersionCreate || caps.mutation.Action != domain.ActionVersionCreate || caps.mutation.ExpectedSequence != "42" || caps.mutation.RequestID != requestID.String() || remote.operation != "versions" {
		t.Fatalf("incorrect bridge binding: out=%+v action=%s mutation=%+v operation=%s", out, feature.action, caps.mutation, remote.operation)
	}
	wantPayload := `{"requestId":"` + requestID.String() + `","expectedSequence":"42","label":"<Launch> &","pinned":true}`
	wantHash := sha256.Sum256([]byte(wantPayload))
	if caps.mutation.PayloadHash != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("payload hash=%s want payload=%s", caps.mutation.PayloadHash, wantPayload)
	}
	inputJSON, _ := json.Marshal(domain.CreateVersionInput{RequestID: requestID, ExpectedSequence: "42", Label: &label, Pinned: true})
	var sent map[string]any
	encoded, _ := json.Marshal(remote.envelope["input"])
	if string(encoded) != string(inputJSON) || json.Unmarshal(encoded, &sent) != nil {
		t.Fatalf("noncanonical version input: %s want %s", encoded, inputJSON)
	}
}

func TestVersionCreateWithoutPrivateBridgeIsRetryableUnavailable(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	principal := domain.UserPrincipal(uuid.New())
	service := NewVersionService(&versionRepoStub{}, &versionFeatureStub{}, &versionCapsStub{}, nil)
	_, err := service.Create(context.Background(), principal, ref, domain.CreateVersionInput{RequestID: uuid.New(), ExpectedSequence: "1", Pinned: false})
	var remote *domain.AgentCollabError
	if !errors.As(err, &remote) || remote.Status != 503 || remote.Code != "UNAVAILABLE" {
		t.Fatalf("nil bridge err=%T %v", err, err)
	}
}

func TestVersionGetMintsFreshImageURLs(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	repo := &versionRepoStub{version: domain.SceneVersion{ID: uuid.New(), Content: json.RawMessage(`{"elements":[],"appState":{},"files":{"file-1":{"assetId":"` + uuid.NewString() + `"}}}`)}}
	features, viewer := &versionFeatureStub{}, &versionAssetViewerStub{}
	service := NewVersionService(repo, features, nil, nil)
	service.SetAssetViewer(viewer)
	got, err := service.Get(context.Background(), domain.UserPrincipal(uuid.New()), ref, repo.version.ID)
	if err != nil {
		t.Fatal(err)
	}
	if features.action != domain.FeatureVersionRead || len(viewer.files) != 1 || viewer.files[0] != "file-1" || got.Assets["file-1"].ViewUrl == "" {
		t.Fatalf("version=%+v files=%v action=%s", got, viewer.files, features.action)
	}
}

func TestVersionMetadataEmitsSafeAtomicEvent(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	versionID, requestID := uuid.New(), uuid.New()
	pinned := true
	repo := &versionRepoStub{version: domain.SceneVersion{ID: versionID, MetadataRevision: 2}}
	features, events := &versionFeatureStub{}, &versionEventStub{}
	service := NewVersionService(repo, features, nil, nil)
	service.SetEventService(events)
	_, err := service.SetMetadata(context.Background(), domain.UserPrincipal(uuid.New()), ref, versionID, domain.VersionMetadataInput{RequestID: requestID, ExpectedRevision: 1, Pinned: &pinned})
	if err != nil {
		t.Fatal(err)
	}
	if events.input.Kind != "version.metadata_updated" || events.input.SourceKey != versionID.String()+":revision:2" {
		t.Fatalf("event=%+v", events.input)
	}
}

func TestVersionRestoreBindsTargetSequenceAndPayload(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	principal := domain.UserPrincipal(uuid.New())
	requestID, versionID, recoveryID := uuid.New(), uuid.New(), uuid.New()
	features, caps := &versionFeatureStub{}, &versionCapsStub{}
	remote := &versionRemoteStub{response: json.RawMessage(`{"requestId":"` + requestID.String() + `","sequence":"43","stateVector":"AA==","preRestoreVersionId":"` + recoveryID.String() + `"}`)}
	service := NewVersionService(&versionRepoStub{}, features, caps, remote)
	result, err := service.Restore(context.Background(), principal, ref, domain.RestoreVersionInput{VersionID: versionID, RequestID: requestID, ExpectedSequence: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if result.PreRestoreVersionID != recoveryID || features.action != domain.FeatureVersionRestore || caps.mutation.Action != domain.ActionVersionRestore || caps.mutation.ExpectedSequence != "42" || remote.operation != "version-restore" {
		t.Fatalf("restore=%+v feature=%s mutation=%+v operation=%s", result, features.action, caps.mutation, remote.operation)
	}
	want := `{"workspaceId":"` + ref.WorkspaceID.String() + `","sceneId":"` + ref.SceneID.String() + `","versionId":"` + versionID.String() + `","requestId":"` + requestID.String() + `","expectedSequence":"42"}`
	digest := sha256.Sum256([]byte(want))
	if caps.mutation.PayloadHash != hex.EncodeToString(digest[:]) {
		t.Fatalf("hash=%s want payload=%s", caps.mutation.PayloadHash, want)
	}
}

func TestVersionDiffUsesCurrentProjectionAtExactSequence(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	versionID := uuid.New()
	repo := &versionRepoStub{version: domain.SceneVersion{ID: versionID, Content: json.RawMessage(`{"elements":[{"id":"old","type":"rectangle","x":0}],"appState":{"theme":"light"},"files":{}}`)}}
	remote := &versionRemoteStub{response: json.RawMessage(`{"sequence":"8","content":{"elements":[{"id":"old","type":"rectangle","x":1},{"id":"new","type":"ellipse"}],"appState":{"theme":"dark"},"files":{}}}`)}
	service := NewVersionService(repo, &versionFeatureStub{}, &versionCapsStub{}, remote)
	diff, err := service.Diff(context.Background(), domain.UserPrincipal(uuid.New()), ref, versionID, "8")
	if err != nil {
		t.Fatal(err)
	}
	if remote.operation != "projection" || len(diff.AddedIDs) != 1 || diff.AddedIDs[0] != "new" || len(diff.ChangedIDs) != 1 || diff.ChangedIDs[0] != "old" || len(diff.ChangedAppStateFields) != 1 || diff.ChangedAppStateFields[0] != "theme" {
		t.Fatalf("operation=%s diff=%+v", remote.operation, diff)
	}
}
