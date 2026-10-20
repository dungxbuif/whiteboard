package usecase

import (
	"context"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"encoding/json"
	"errors"
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
}

func (s *versionRemoteStub) Call(_ domain.Context, operation string, _ domain.SceneRef, envelope map[string]any, _ int64) (json.RawMessage, error) {
	s.operation, s.envelope = operation, envelope
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
