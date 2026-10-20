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
	publishCommitted       bool
	stateErr               error
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
	if s.publishCommitted {
		scene.PendingDuplicate = false
		scene.DeletedAt.Valid = false
	}
	if s.publishErr != nil {
		return s.publishErr
	}
	s.created = scene
	return nil
}
func (s *duplicateSceneRepoStub) FindDuplicateState(ctx domain.Context, _ uuid.UUID, id uuid.UUID) (*domain.Scene, error) {
	s.cleanupContextCanceled = ctx.Err() != nil
	if s.stateErr != nil {
		return nil, s.stateErr
	}
	if s.created == nil || s.created.ID != id || s.deleted {
		return nil, nil
	}
	return s.created, nil
}

func TestDuplicateUncertainPublishStatePreservesObjects(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: json.RawMessage(`{"elements":[],"appState":{},"files":{}}`)}, publishErr: errors.New("publish connection lost"), stateErr: errors.New("verification unavailable")}
	assets := &duplicateAssetStagerStub{}
	service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
	service.SetDuplicateDependencies(scenes, nil, assets, duplicateAccessStub{})
	_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
	var uncertain *domain.DuplicateStateError
	if !errors.As(err, &uncertain) || scenes.deleted || assets.cleaned || scenes.cleanupContextCanceled {
		t.Fatalf("err=%v scenes=%+v assets=%+v", err, scenes, assets)
	}
}
func (s *duplicateSceneRepoStub) ClaimDuplicateCleanup(ctx domain.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	s.cleanupContextCanceled = ctx.Err() != nil
	if s.created == nil || !s.created.PendingDuplicate {
		return false, nil
	}
	s.created.DuplicateCleanupStarted = true
	return true, nil
}
func (s *duplicateSceneRepoStub) HardDelete(ctx domain.Context, _ uuid.UUID, _ uuid.UUID) error {
	s.cleanupContextCanceled = ctx.Err() != nil
	if s.created == nil || !s.created.PendingDuplicate || !s.created.DuplicateCleanupStarted {
		return domain.ErrInaccessible
	}
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
func (duplicateAccessStub) WithWorkspaceLock(ctx domain.Context, _ uuid.UUID, _ uuid.UUID, fn func(domain.Context) error) error {
	return fn(ctx)
}

type revocableDuplicateAccessStub struct {
	duplicateAccessStub
	revokeScene, revokeWorkspace, locked bool
}

func (s *revocableDuplicateAccessStub) Resolve(_ domain.Context, _ domain.Principal, _ domain.SceneRef) (domain.AccessDecision, error) {
	if s.revokeScene {
		return domain.AccessDecision{Access: domain.AccessView}, nil
	}
	return domain.AccessDecision{Access: domain.AccessEdit}, nil
}
func (s *revocableDuplicateAccessStub) ResolveWorkspace(_ domain.Context, _ domain.Principal, _ uuid.UUID) (domain.AccessDecision, error) {
	if s.revokeWorkspace {
		return domain.AccessDecision{Access: domain.AccessView}, nil
	}
	return domain.AccessDecision{Access: domain.AccessEdit}, nil
}
func (s *revocableDuplicateAccessStub) WithWorkspaceLock(ctx domain.Context, _ uuid.UUID, _ uuid.UUID, fn func(domain.Context) error) error {
	s.locked = true
	defer func() { s.locked = false }()
	return fn(ctx)
}

type duplicateAssetStagerStub struct {
	called, cleaned, cleanupContextCanceled bool
	stageErr, cleanupErr                    error
	onStage                                 func()
	result                                  json.RawMessage
}

func TestDuplicateRejectsRevocationAfterStagingBeforePublish(t *testing.T) {
	for _, revoked := range []string{"scene", "workspace"} {
		t.Run(revoked, func(t *testing.T) {
			ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
			scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: json.RawMessage(`{"elements":[],"appState":{},"files":{}}`)}}
			access := &revocableDuplicateAccessStub{}
			staging := make(chan struct{})
			continueStage := make(chan struct{})
			assets := &duplicateAssetStagerStub{onStage: func() {
				close(staging)
				<-continueStage
			}}
			service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
			service.SetDuplicateDependencies(scenes, nil, assets, access)
			result := make(chan error, 1)
			go func() {
				_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
				result <- err
			}()
			<-staging
			if revoked == "scene" {
				access.revokeScene = true
			} else {
				access.revokeWorkspace = true
			}
			close(continueStage)
			err := <-result
			if !errors.Is(err, domain.ErrAccessDenied) || scenes.published || !assets.cleaned || !scenes.deleted {
				t.Fatalf("revocation escaped publish gate: err=%v published=%v cleaned=%v deleted=%v", err, scenes.published, assets.cleaned, scenes.deleted)
			}
		})
	}
}

func TestDuplicateV1ContentDropsRendererStateAndInlineFiles(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	asset := uuid.NewString()
	legacy := json.RawMessage(`{"elements":[` + validDuplicateImageElement + `],"appState":{"theme":"dark","gridSize":20,"openSidebar":"library","collaborators":{"user":{}}},"files":{"file":{"id":"file","mimeType":"image/png","created":1,"assetId":"` + asset + `","dataURL":"data:image/png;base64,secret","lastRetrieved":2}}}`)
	scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: legacy}}
	service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
	service.SetDuplicateDependencies(scenes, nil, &duplicateAssetStagerStub{}, duplicateAccessStub{})
	copy, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
	if err != nil {
		t.Fatal(err)
	}
	var content struct {
		Elements          []map[string]any          `json:"elements"`
		AppState          map[string]any            `json:"appState"`
		Files             map[string]map[string]any `json:"files"`
		PresentationOrder []string                  `json:"presentationOrder"`
	}
	if err := json.Unmarshal(copy.Content, &content); err != nil {
		t.Fatal(err)
	}
	if len(content.Elements) != 1 || content.Elements[0]["fileId"] != "file" || content.AppState["theme"] != "dark" || content.AppState["gridSize"] != float64(20) || len(content.AppState) != 2 || len(content.PresentationOrder) != 0 {
		t.Fatalf("durable scene fields changed: %s", copy.Content)
	}
	file := content.Files["file"]
	if file["assetId"] != asset || file["mimeType"] != "image/png" || file["lastRetrieved"] != float64(2) || file["dataURL"] != nil {
		t.Fatalf("inline or scene asset fields wrong: %s", copy.Content)
	}
}

const validDuplicateImageElement = `{"id":"image","type":"image","x":0,"y":0,"width":100,"height":100,"angle":0,"strokeColor":"#111111","backgroundColor":"transparent","fillStyle":"solid","strokeWidth":1,"strokeStyle":"solid","roughness":1,"opacity":100,"groupIds":[],"frameId":null,"roundness":null,"seed":1,"version":1,"versionNonce":1,"index":null,"isDeleted":false,"boundElements":null,"updated":1,"link":null,"locked":false,"fileId":"file","status":"saved","scale":[1,1],"crop":null}`
const validDuplicateRectangleElement = `{"id":"target","type":"rectangle","x":0,"y":0,"width":100,"height":100,"angle":0,"strokeColor":"#111111","backgroundColor":"transparent","fillStyle":"solid","strokeWidth":1,"strokeStyle":"solid","roughness":1,"opacity":100,"groupIds":[],"frameId":null,"roundness":null,"seed":1,"version":1,"versionNonce":1,"index":null,"isDeleted":false,"boundElements":null,"updated":1,"link":null,"locked":false}`
const validDuplicateTextElement = `{"id":"child","type":"text","x":0,"y":0,"width":100,"height":30,"angle":0,"strokeColor":"#111111","backgroundColor":"transparent","fillStyle":"solid","strokeWidth":1,"strokeStyle":"solid","roughness":1,"opacity":100,"groupIds":[],"frameId":null,"roundness":null,"seed":1,"version":1,"versionNonce":1,"index":null,"isDeleted":false,"boundElements":null,"updated":1,"link":null,"locked":false,"fontSize":16,"fontFamily":1,"text":"Hi","textAlign":"left","verticalAlign":"top","containerId":"target","originalText":"Hi","autoResize":true,"lineHeight":1.25}`
const validDuplicateLineElement = `{"id":"line","type":"line","x":0,"y":0,"width":100,"height":100,"angle":0,"strokeColor":"#111111","backgroundColor":"transparent","fillStyle":"solid","strokeWidth":1,"strokeStyle":"solid","roughness":1,"opacity":100,"groupIds":[],"frameId":null,"roundness":null,"seed":1,"version":1,"versionNonce":1,"index":null,"isDeleted":false,"boundElements":null,"updated":1,"link":null,"locked":false,"points":[[0,0],[1,1]],"lastCommittedPoint":null,"startBinding":{"elementId":"target","focus":0,"gap":1},"endBinding":null,"startArrowhead":null,"endArrowhead":null}`
const validDuplicateFreedrawElement = `{"id":"free","type":"freedraw","x":0,"y":0,"width":100,"height":100,"angle":0,"strokeColor":"#111111","backgroundColor":"transparent","fillStyle":"solid","strokeWidth":1,"strokeStyle":"solid","roughness":1,"opacity":100,"groupIds":[],"frameId":null,"roundness":null,"seed":1,"version":1,"versionNonce":1,"index":null,"isDeleted":false,"boundElements":null,"updated":1,"link":null,"locked":false,"points":[[0,0]],"pressures":[0.5],"simulatePressure":false,"lastCommittedPoint":null}`

func TestDuplicateV1RejectsUnsafeMetadataBindingsAndPressure(t *testing.T) {
	asset := uuid.NewString()
	imageContent := func(element string) string {
		return `{"elements":[` + element + `],"appState":{},"files":{"file":{"id":"file","mimeType":"image/png","created":1,"assetId":"` + asset + `"}}}`
	}
	bound := strings.Replace(validDuplicateRectangleElement, `"boundElements":null`, `"boundElements":[{"id":"child","type":"text","junk":true}]`, 1)
	invalidBoundType := strings.Replace(validDuplicateRectangleElement, `"boundElements":null`, `"boundElements":[{"id":"child","type":"rectangle"}]`, 1)
	childRectangle := strings.Replace(validDuplicateRectangleElement, `"id":"target"`, `"id":"child"`, 1)
	line := strings.Replace(validDuplicateLineElement, `"gap":1}`, `"gap":1,"junk":true}`, 1)
	for _, tc := range []struct{ name, content string }{
		{name: "null custom data", content: imageContent(strings.Replace(validDuplicateImageElement, `"locked":false`, `"locked":false,"customData":null`, 1))},
		{name: "nested script metadata", content: imageContent(strings.Replace(validDuplicateImageElement, `"locked":false`, `"locked":false,"customData":{"safe":{"script":"alert(1)"}}`, 1))},
		{name: "bound entry extra field", content: `{"elements":[` + bound + `,` + validDuplicateTextElement + `],"appState":{},"files":{}}`},
		{name: "bound entry invalid type", content: `{"elements":[` + invalidBoundType + `,` + childRectangle + `],"appState":{},"files":{}}`},
		{name: "binding extra field", content: `{"elements":[` + validDuplicateRectangleElement + `,` + line + `],"appState":{},"files":{}}`},
		{name: "pressure above one", content: `{"elements":[` + strings.Replace(validDuplicateFreedrawElement, `"pressures":[0.5]`, `"pressures":[1.5]`, 1) + `],"appState":{},"files":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
			scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: json.RawMessage(tc.content)}}
			service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
			service.SetDuplicateDependencies(scenes, nil, &duplicateAssetStagerStub{}, duplicateAccessStub{})
			_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
			if !errors.Is(err, domain.ErrInvalidVersionInput) || scenes.created != nil {
				t.Fatalf("unsafe v1 content copied: err=%v created=%v", err, scenes.created != nil)
			}
		})
	}
}

func TestDuplicateV1RejectsUnsafeStagedMetadata(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	source := json.RawMessage(`{"elements":[` + validDuplicateRectangleElement + `],"appState":{},"files":{}}`)
	staged := json.RawMessage(strings.Replace(string(source), `"locked":false`, `"locked":false,"customData":{"nested":{"accessToken":"secret"}}`, 1))
	scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: source}}
	assets := &duplicateAssetStagerStub{result: staged}
	service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
	service.SetDuplicateDependencies(scenes, nil, assets, duplicateAccessStub{})
	_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
	if !errors.Is(err, domain.ErrInvalidVersionInput) || scenes.published || !scenes.deleted || !assets.cleaned {
		t.Fatalf("unsafe staged metadata escaped: err=%v published=%v deleted=%v cleaned=%v", err, scenes.published, scenes.deleted, assets.cleaned)
	}
}

func duplicateNestedMetadata(depth int) string {
	value := `"ok"`
	for range depth {
		value = `{"safe":` + value + `}`
	}
	return value
}

func duplicateContentWithMetadata(depth int) json.RawMessage {
	element := strings.Replace(validDuplicateRectangleElement, `"locked":false`, `"locked":false,"customData":`+duplicateNestedMetadata(depth), 1)
	return json.RawMessage(`{"elements":[` + element + `],"appState":{},"files":{}}`)
}

func TestDuplicateV1MetadataDepthMatchesSceneRoot(t *testing.T) {
	for _, tc := range []struct {
		name       string
		depth      int
		stage      bool
		wantReject bool
	}{
		{name: "source depth 29 accepted", depth: 29},
		{name: "source depth 30 rejected", depth: 30, wantReject: true},
		{name: "staged depth 29 accepted", depth: 29, stage: true},
		{name: "staged depth 30 rejected", depth: 30, stage: true, wantReject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
			source := duplicateContentWithMetadata(tc.depth)
			if tc.stage {
				source = duplicateContentWithMetadata(1)
			}
			scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: source}}
			assets := &duplicateAssetStagerStub{}
			if tc.stage {
				assets.result = duplicateContentWithMetadata(tc.depth)
			}
			service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
			service.SetDuplicateDependencies(scenes, nil, assets, duplicateAccessStub{})
			_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
			if tc.wantReject {
				if !errors.Is(err, domain.ErrInvalidVersionInput) || scenes.published || (tc.stage && (!scenes.deleted || !assets.cleaned)) || (!tc.stage && scenes.created != nil) {
					t.Fatalf("deep metadata escaped: err=%v created=%v published=%v deleted=%v cleaned=%v", err, scenes.created != nil, scenes.published, scenes.deleted, assets.cleaned)
				}
			} else if err != nil || !scenes.published {
				t.Fatalf("valid metadata rejected: err=%v published=%v", err, scenes.published)
			}
		})
	}
}

func TestDuplicateV1RejectsBrokenImageReferenceAndShape(t *testing.T) {
	for name, element := range map[string]string{
		"missing file":  validDuplicateImageElement,
		"invalid image": strings.Replace(validDuplicateImageElement, `"status":"saved",`, "", 1),
	} {
		t.Run(name, func(t *testing.T) {
			ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
			content := json.RawMessage(`{"elements":[` + element + `],"appState":{},"files":{}}`)
			scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: content}}
			service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
			service.SetDuplicateDependencies(scenes, nil, &duplicateAssetStagerStub{}, duplicateAccessStub{})
			_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
			if !errors.Is(err, domain.ErrInvalidVersionInput) || scenes.created != nil || scenes.published {
				t.Fatalf("invalid v1 image was copied: err=%v created=%v published=%v", err, scenes.created != nil, scenes.published)
			}
		})
	}
}

func TestDuplicateV1RejectsIncompleteFileDescriptor(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	content := json.RawMessage(`{"elements":[` + validDuplicateImageElement + `],"appState":{},"files":{"file":{"id":"file","mimeType":"image/png","assetId":"` + uuid.NewString() + `"}}}`)
	scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: content}}
	service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
	service.SetDuplicateDependencies(scenes, nil, &duplicateAssetStagerStub{}, duplicateAccessStub{})
	_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
	if !errors.Is(err, domain.ErrInvalidVersionInput) || scenes.created != nil {
		t.Fatalf("incomplete asset descriptor was copied: err=%v created=%v", err, scenes.created != nil)
	}
}

func TestDuplicateV1RejectsMalformedDurableFieldTypes(t *testing.T) {
	asset := uuid.NewString()
	valid := `{"elements":[` + validDuplicateImageElement + `],"appState":{"theme":"dark"},"files":{"file":{"id":"file","mimeType":"image/png","created":1,"assetId":"` + asset + `"}}}`
	for _, tc := range []struct{ name, content string }{
		{name: "x string", content: strings.Replace(valid, `"x":0`, `"x":"zero"`, 1)},
		{name: "group IDs string", content: strings.Replace(valid, `"groupIds":[]`, `"groupIds":"group"`, 1)},
		{name: "created null", content: strings.Replace(valid, `"created":1`, `"created":null`, 1)},
		{name: "unsafe link", content: strings.Replace(valid, `"link":null`, `"link":"//evil.example"`, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
			scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: json.RawMessage(tc.content)}}
			service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
			service.SetDuplicateDependencies(scenes, nil, &duplicateAssetStagerStub{}, duplicateAccessStub{})
			_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
			if !errors.Is(err, domain.ErrInvalidVersionInput) || scenes.created != nil {
				t.Fatalf("malformed scene was copied: err=%v created=%v", err, scenes.created != nil)
			}
		})
	}
}

func TestDuplicateV1RejectsStagedFileWithoutAssetID(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	asset := uuid.NewString()
	valid := json.RawMessage(`{"elements":[` + validDuplicateImageElement + `],"appState":{},"files":{"file":{"id":"file","mimeType":"image/png","created":1,"assetId":"` + asset + `"}}}`)
	staged := json.RawMessage(strings.Replace(string(valid), `,"assetId":"`+asset+`"`, "", 1))
	scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: valid}}
	assets := &duplicateAssetStagerStub{result: staged}
	service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
	service.SetDuplicateDependencies(scenes, nil, assets, duplicateAccessStub{})
	_, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
	if !errors.Is(err, domain.ErrInvalidVersionInput) || scenes.published || !scenes.deleted || !assets.cleaned {
		t.Fatalf("staged file without new asset escaped: err=%v published=%v deleted=%v cleaned=%v", err, scenes.published, scenes.deleted, assets.cleaned)
	}
}

func (s *duplicateAssetStagerStub) StageSceneCopy(_ domain.Context, _ domain.Principal, _, _ domain.SceneRef, _ uuid.UUID, content json.RawMessage) (json.RawMessage, error) {
	s.called = true
	if s.onStage != nil {
		s.onStage()
	}
	if s.result != nil {
		return s.result, s.stageErr
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
	var cleanup *domain.DuplicateCleanupError
	if err == nil || !errors.As(err, &cleanup) || cleanup.SceneID != scenes.created.ID || !strings.Contains(err.Error(), "delete object failed") || scenes.deleted || !scenes.created.DeletedAt.Valid || !scenes.created.PendingDuplicate {
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

func TestDuplicatePublishCommitThenErrorPreservesPublishedSceneAndAssets(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	scenes := &duplicateSceneRepoStub{source: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID, WriterMode: "v1", Content: json.RawMessage(`{"elements":[],"appState":{},"files":{}}`)}, publishErr: errors.New("connection lost after commit"), publishCommitted: true}
	assets := &duplicateAssetStagerStub{}
	service := NewVersionService(nil, nil, &versionCapsStub{}, nil)
	service.SetDuplicateDependencies(scenes, nil, assets, duplicateAccessStub{})
	copy, err := service.Duplicate(context.Background(), domain.UserPrincipal(uuid.New()), ref, "Copy", nil)
	if err != nil || copy == nil || copy.PendingDuplicate || copy.DeletedAt.Valid || !scenes.published || scenes.deleted || assets.cleaned {
		t.Fatalf("copy=%+v err=%v scenes=%+v assets=%+v", copy, err, scenes, assets)
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
