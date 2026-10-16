package usecase

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"devtool/internal/domain"
)

type assetUsecase struct {
	sceneRepo domain.SceneRepository
	assetRepo domain.SceneAssetRepository
	s3Repo    domain.S3Repository
	bucket    string
	urlExpiry time.Duration
}

func NewAssetUsecase(
	sceneRepo domain.SceneRepository,
	assetRepo domain.SceneAssetRepository,
	s3Repo domain.S3Repository,
	bucket string,
	urlExpiry time.Duration,
) domain.AssetUsecase {
	return &assetUsecase{
		sceneRepo: sceneRepo,
		assetRepo: assetRepo,
		s3Repo:    s3Repo,
		bucket:    bucket,
		urlExpiry: urlExpiry,
	}
}

func (u *assetUsecase) PresignUpload(ctx domain.Context, workspaceID, userID, sceneID uuid.UUID, assets []domain.AssetUploadRequest) (map[string]domain.AssetUploadResponse, error) {
	// Verify scene ownership
	_, err := u.sceneRepo.FindByID(ctx, workspaceID, sceneID)
	if err != nil {
		return nil, fmt.Errorf("scene not found or unauthorized: %w", err)
	}

	result := make(map[string]domain.AssetUploadResponse)

	for _, req := range assets {
		assetID := uuid.New()
		storageKey := fmt.Sprintf("workspaces/%s/scenes/%s/assets/%s", workspaceID, sceneID, assetID)

		uploadUrl, err := u.s3Repo.PresignUpload(ctx, u.bucket, storageKey, u.urlExpiry)
		if err != nil {
			return nil, fmt.Errorf("failed to presign upload for file %s: %w", req.FileID, err)
		}

		fileID := req.FileID // Avoid taking address of loop variable directly in older Go, though Go 1.22+ handles it.
		asset := &domain.SceneAsset{
			ID:            assetID,
			WorkspaceID:   workspaceID,
			SceneID:       &sceneID,
			UploadedBy:    &userID,
			AssetKind:     domain.SceneAssetKindExcalidrawFile,
			FileID:        &fileID,
			StorageBucket: u.bucket,
			StorageKey:    storageKey,
			MimeType:      req.MimeType,
			ByteSize:      req.ByteSize,
			SHA256:        req.SHA256,
			Width:         req.Width,
			Height:        req.Height,
		}

		if err := u.assetRepo.Create(ctx, asset); err != nil {
			return nil, fmt.Errorf("failed to create asset metadata: %w", err)
		}

		result[fileID] = domain.AssetUploadResponse{
			AssetID:    assetID,
			UploadUrl:  uploadUrl,
			StorageKey: storageKey,
			ExpiresAt:  time.Now().Add(u.urlExpiry),
		}
	}

	return result, nil
}

func (u *assetUsecase) PresignView(ctx domain.Context, workspaceID, sceneID uuid.UUID, fileIDs []string) (map[string]domain.AssetViewResponse, error) {
	// Verify scene ownership
	_, err := u.sceneRepo.FindByID(ctx, workspaceID, sceneID)
	if err != nil {
		return nil, fmt.Errorf("scene not found or unauthorized: %w", err)
	}

	assets, err := u.assetRepo.GetByFileIDs(ctx, workspaceID, fileIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assets: %w", err)
	}

	result := make(map[string]domain.AssetViewResponse)
	now := time.Now()

	for _, asset := range assets {
		if asset.FileID == nil || asset.SceneID == nil || *asset.SceneID != sceneID || asset.WorkspaceID != workspaceID || asset.DeletedAt != nil {
			continue
		}

		var viewUrl string
		var expiresAt time.Time

		// Check if we have a valid cached presigned URL
		if asset.PresignedURL != nil && asset.URLExpiresAt != nil && asset.URLExpiresAt.After(now.Add(5*time.Minute)) {
			viewUrl = *asset.PresignedURL
			expiresAt = *asset.URLExpiresAt
		} else {
			// Generate new URL
			newUrl, err := u.s3Repo.PresignDownload(ctx, asset.StorageBucket, asset.StorageKey, u.urlExpiry)
			if err != nil {
				return nil, fmt.Errorf("failed to presign download for asset %s: %w", asset.ID, err)
			}
			viewUrl = newUrl
			expiresAt = now.Add(u.urlExpiry)

			_ = u.assetRepo.UpdatePresignedURL(ctx, asset.ID, viewUrl, expiresAt)
		}

		result[*asset.FileID] = domain.AssetViewResponse{
			AssetID:   asset.ID,
			ViewUrl:   viewUrl,
			MimeType:  asset.MimeType,
			ExpiresAt: expiresAt,
		}
	}

	return result, nil
}
