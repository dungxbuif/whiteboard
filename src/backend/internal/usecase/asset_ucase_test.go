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

func (m *mockSceneAssetRepo) GetByFileIDs(_ domain.Context, workspaceID uuid.UUID, fileIDs []string) ([]domain.SceneAsset, error) {
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
