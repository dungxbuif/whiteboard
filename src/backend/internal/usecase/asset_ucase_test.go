package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type mockSceneAssetRepo struct {
	domain.SceneAssetRepository
	assets         map[uuid.UUID]*domain.SceneAsset
	updatedURL     string
	updatedExpires *time.Time
}

func (m *mockSceneAssetRepo) Create(_ domain.Context, asset *domain.SceneAsset) error {
	m.assets[asset.ID] = asset
	return nil
}

func (m *mockSceneAssetRepo) GetByFileIDs(_ domain.Context, workspaceID, sceneID uuid.UUID, fileIDs []string) ([]domain.SceneAsset, error) {
	var result []domain.SceneAsset
	for _, asset := range m.assets {
		if asset.WorkspaceID == workspaceID && asset.FileID != nil {
			for _, fid := range fileIDs {
				if *asset.FileID == fid {
					result = append(result, *asset)
					break
				}
			}
		}
	}
	return result, nil
}

func (m *mockSceneAssetRepo) UpdatePresignedURL(_ domain.Context, id uuid.UUID, url string, expiresAt time.Time) error {
	if asset, ok := m.assets[id]; ok {
		asset.PresignedURL = &url
		asset.URLExpiresAt = &expiresAt
		m.updatedURL = url
		m.updatedExpires = &expiresAt
	}
	return nil
}

type mockS3Repo struct {
	domain.S3Repository
}

func (m *mockS3Repo) PresignUpload(_ domain.Context, bucket, key string, expires time.Duration) (string, error) {
	return fmt.Sprintf("https://s3.local/%s/%s/upload", bucket, key), nil
}

func (m *mockS3Repo) PresignDownload(_ domain.Context, bucket, key string, expires time.Duration) (string, error) {
	return fmt.Sprintf("https://s3.local/%s/%s/download", bucket, key), nil
}

func TestAssetUsecase_PresignUpload(t *testing.T) {
	workspaceID := uuid.New()
	userID := uuid.New()
	sceneID := uuid.New()

	sceneRepo := &mockSceneRepo{
		scene: &domain.Scene{ID: sceneID, WorkspaceID: workspaceID},
	}
	assetRepo := &mockSceneAssetRepo{assets: make(map[uuid.UUID]*domain.SceneAsset)}
	s3Repo := &mockS3Repo{}

	uc := NewAssetUsecase(sceneRepo, assetRepo, s3Repo, "test-bucket", time.Hour)

	reqs := []domain.AssetUploadRequest{
		{FileID: "file-1", MimeType: "image/png", ByteSize: 1024},
		{FileID: "file-2", MimeType: "image/jpeg", ByteSize: 2048},
	}

	res, err := uc.PresignUpload(context.Background(), workspaceID, userID, sceneID, reqs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(res))
	}

	for _, req := range reqs {
		resp, ok := res[req.FileID]
		if !ok {
			t.Errorf("missing response for file %s", req.FileID)
		}
		if resp.UploadUrl == "" {
			t.Errorf("empty upload URL for file %s", req.FileID)
		}
	}
}

func TestAssetUsecase_PresignView_CacheMissAndHit(t *testing.T) {
	workspaceID := uuid.New()
	sceneID := uuid.New()
	fileID := "file-1"
	assetID := uuid.New()

	sceneRepo := &mockSceneRepo{
		scene: &domain.Scene{ID: sceneID, WorkspaceID: workspaceID},
	}

	// An asset with expired or nil URL
	asset := &domain.SceneAsset{
		ID:            assetID,
		WorkspaceID:   workspaceID,
		SceneID:       &sceneID,
		FileID:        &fileID,
		StorageBucket: "test-bucket",
		StorageKey:    "test-key",
		MimeType:      "image/png",
	}

	assetRepo := &mockSceneAssetRepo{
		assets: map[uuid.UUID]*domain.SceneAsset{
			assetID: asset,
		},
	}
	s3Repo := &mockS3Repo{}

	uc := NewAssetUsecase(sceneRepo, assetRepo, s3Repo, "test-bucket", time.Hour)

	// Cache Miss
	res, err := uc.PresignView(context.Background(), workspaceID, sceneID, []string{fileID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, ok := res[fileID]
	if !ok {
		t.Fatalf("expected response for file %s", fileID)
	}
	expectedUrl := "https://s3.local/test-bucket/test-key/download"
	if resp.ViewUrl != expectedUrl {
		t.Errorf("expected url %s, got %s", expectedUrl, resp.ViewUrl)
	}
	if assetRepo.updatedURL != expectedUrl {
		t.Errorf("expected repo to update URL to %s, got %s", expectedUrl, assetRepo.updatedURL)
	}

	// Now that it's cached, test Cache Hit
	// mockSceneAssetRepo UpdatePresignedURL sets PresignedURL and URLExpiresAt directly
	if asset.PresignedURL == nil || *asset.PresignedURL != expectedUrl {
		t.Fatalf("asset PresignedURL not set correctly")
	}

	res2, err := uc.PresignView(context.Background(), workspaceID, sceneID, []string{fileID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp2, ok := res2[fileID]
	if !ok {
		t.Fatalf("expected response for file %s on hit", fileID)
	}
	if resp2.ViewUrl != expectedUrl {
		t.Errorf("expected hit url %s, got %s", expectedUrl, resp2.ViewUrl)
	}
}

func TestAssetAccessCannotReadOtherSceneFiles(t *testing.T) {
	workspace, scene, other, assetID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	fileID := "foreign-file"
	sceneRepo := &mockSceneRepo{scene: &domain.Scene{ID: scene, WorkspaceID: workspace}}
	assetRepo := &mockSceneAssetRepo{assets: map[uuid.UUID]*domain.SceneAsset{assetID: {ID: assetID, WorkspaceID: workspace, SceneID: &other, FileID: &fileID, StorageKey: "foreign"}}}
	service := NewAssetUsecase(sceneRepo, assetRepo, &mockS3Repo{}, "bucket", time.Hour)
	result, err := service.PresignView(context.Background(), workspace, scene, []string{fileID})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("other scene file leaked: %+v", result)
	}
}

func TestPublicUploadCapabilityValidation(t *testing.T) {
	for _, r := range []domain.AssetUploadRequest{{FileID: "x", MimeType: "text/html", ByteSize: 1}, {FileID: "x", MimeType: "image/png", ByteSize: 0}, {FileID: "x", MimeType: "image/png", ByteSize: 10485761}, {FileID: "", MimeType: "image/png", ByteSize: 1}} {
		if e := validateAssets([]domain.AssetUploadRequest{r}); e == nil {
			t.Errorf("accepted invalid upload: %+v", r)
		}
	}
	if e := validateAssets(nil); e == nil {
		t.Error("accepted empty upload batch")
	}
	if e := validateAssets([]domain.AssetUploadRequest{{FileID: "same", MimeType: "image/png", ByteSize: 1}, {FileID: "same", MimeType: "image/png", ByteSize: 1}}); e == nil {
		t.Error("accepted duplicate file IDs")
	}
}

func (m *mockS3Repo) PresignUploadConstrained(c domain.Context, b, k, mime string, n int64, deadline time.Time) (string, map[string]string, error) {
	u, e := m.PresignUpload(c, b, k, time.Until(deadline))
	return u, map[string]string{"Content-Type": mime}, e
}

type delayedAssetScene struct {
	domain.SceneRepository
	scene *domain.Scene
	delay time.Duration
}

func (r delayedAssetScene) FindByID(domain.Context, uuid.UUID, uuid.UUID) (*domain.Scene, error) {
	time.Sleep(r.delay)
	return r.scene, nil
}

type delayedAssetRows struct {
	*mockSceneAssetRepo
	delay time.Duration
}

func (r delayedAssetRows) GetByFileIDs(c domain.Context, w, s uuid.UUID, ids []string) ([]domain.SceneAsset, error) {
	time.Sleep(r.delay)
	return r.mockSceneAssetRepo.GetByFileIDs(c, w, s, ids)
}

type deadlineAssetSigner struct {
	mockS3Repo
	observed time.Time
	calls    int
}

func (s *deadlineAssetSigner) PresignUploadConstrained(_ domain.Context, _, _, _ string, _ int64, deadline time.Time) (string, map[string]string, error) {
	s.observed = deadline
	s.calls++
	return "upload", nil, nil
}
func (s *deadlineAssetSigner) PresignDownloadUntil(_ domain.Context, _, _ string, deadline time.Time) (string, error) {
	s.observed = deadline
	s.calls++
	return "view", nil
}
func TestAssetAbsoluteDeadline(t *testing.T) {
	for _, mode := range []string{"upload", "view", "cache"} {
		t.Run(mode, func(t *testing.T) {
			ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
			file := "deadline"
			id := uuid.New()
			deadline := time.Now().Add(8 * time.Second).Truncate(time.Second)
			row := &domain.SceneAsset{ID: id, WorkspaceID: ref.WorkspaceID, SceneID: &ref.SceneID, FileID: &file, MimeType: "image/png", ByteSize: 7, StorageBucket: "assets", StorageKey: "existing"}
			if mode == "cache" {
				cached := "cached"
				expires := deadline.Add(time.Second)
				row.PresignedURL = &cached
				row.URLExpiresAt = &expires
			}
			rows := &mockSceneAssetRepo{assets: map[uuid.UUID]*domain.SceneAsset{id: row}}
			signer := &deadlineAssetSigner{}
			scene := delayedAssetScene{scene: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID}}
			var repo domain.SceneAssetRepository = rows
			if mode == "upload" {
				scene.delay = 1200 * time.Millisecond
			} else {
				repo = delayedAssetRows{rows, 1200 * time.Millisecond}
			}
			u := NewAssetUsecase(scene, repo, signer, "assets", time.Minute)
			if mode == "upload" {
				result, e := u.PresignPrincipalUpload(context.Background(), domain.UserPrincipal(uuid.New()), ref, []domain.AssetUploadRequest{{FileID: "new", MimeType: "image/png", ByteSize: 7}}, deadline)
				if e != nil {
					t.Fatal(e)
				}
				if result["new"].ExpiresAt.After(deadline) {
					t.Errorf("upload expiry %s exceeds authorization %s", result["new"].ExpiresAt, deadline)
				}
			} else {
				result, e := u.PresignScopedView(context.Background(), ref, []string{file}, deadline)
				if e != nil {
					t.Fatal(e)
				}
				if result[file].ExpiresAt.After(deadline) {
					t.Errorf("view expiry %s exceeds authorization %s", result[file].ExpiresAt, deadline)
				}
			}
			if !signer.observed.IsZero() && signer.observed.After(deadline) {
				t.Errorf("signer lifetime %s exceeds authorization %s", signer.observed, deadline)
			}
		})
	}
}

func (m *mockS3Repo) PresignDownloadUntil(c domain.Context, b, k string, deadline time.Time) (string, error) {
	return m.PresignDownload(c, b, k, time.Until(deadline))
}

func TestAssetAbsoluteDeadlineExpiredDuringRead(t *testing.T) {
	for _, mode := range []string{"upload", "view"} {
		t.Run(mode, func(t *testing.T) {
			ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
			file := "expired"
			id := uuid.New()
			deadline := time.Now().Add(2 * time.Second)
			rows := &mockSceneAssetRepo{assets: map[uuid.UUID]*domain.SceneAsset{id: {ID: id, WorkspaceID: ref.WorkspaceID, SceneID: &ref.SceneID, FileID: &file}}}
			signer := &deadlineAssetSigner{}
			scene := delayedAssetScene{scene: &domain.Scene{ID: ref.SceneID, WorkspaceID: ref.WorkspaceID}, delay: 2100 * time.Millisecond}
			u := NewAssetUsecase(scene, rows, signer, "assets", time.Minute)
			if mode == "upload" {
				g, e := u.PresignPrincipalUpload(context.Background(), domain.UserPrincipal(uuid.New()), ref, []domain.AssetUploadRequest{{FileID: "new", MimeType: "image/png", ByteSize: 7}}, deadline)
				if e == nil || len(g) != 0 {
					t.Fatal("read returned expired upload capability")
				}
			} else {
				g, e := u.PresignScopedView(context.Background(), ref, []string{file}, deadline)
				if e == nil || len(g) != 0 {
					t.Fatal("read returned expired view capability")
				}
			}
			if signer.calls != 0 {
				t.Fatal("expired authority reached signing")
			}
		})
	}
}
