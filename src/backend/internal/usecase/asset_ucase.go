package usecase

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"strings"
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
) *assetUsecase {
	if urlExpiry <= 0 || urlExpiry > domain.AssetCapabilityLifetime {
		urlExpiry = domain.AssetCapabilityLifetime
	}
	return &assetUsecase{
		sceneRepo: sceneRepo,
		assetRepo: assetRepo,
		s3Repo:    s3Repo,
		bucket:    bucket,
		urlExpiry: urlExpiry,
	}
}

func (u *assetUsecase) PresignUpload(ctx domain.Context, workspaceID, userID, sceneID uuid.UUID, assets []domain.AssetUploadRequest) (map[string]domain.AssetUploadResponse, error) {
	return u.PresignPrincipalUpload(ctx, domain.UserPrincipal(userID), domain.SceneRef{WorkspaceID: workspaceID, SceneID: sceneID}, assets, u.urlExpiry)
}
func (u *assetUsecase) PresignPrincipalUpload(ctx domain.Context, p domain.Principal, ref domain.SceneRef, requests []domain.AssetUploadRequest, ttl time.Duration) (map[string]domain.AssetUploadResponse, error) {
	if e := validateAssets(requests); e != nil {
		return nil, e
	}
	if ttl < time.Second {
		return nil, domain.ErrInvalidSceneSession
	}
	if ttl > domain.AssetCapabilityLifetime {
		ttl = domain.AssetCapabilityLifetime
	}
	if _, e := u.sceneRepo.FindByID(ctx, ref.WorkspaceID, ref.SceneID); e != nil {
		return nil, e
	}
	constrained, ok := u.s3Repo.(domain.ScopedUploadRepository)
	if !ok {
		return nil, fmt.Errorf("object storage cannot enforce upload constraints")
	}
	var user *uuid.UUID
	switch p.Kind {
	case domain.PrincipalUser:
		id, e := normalizedUser(p)
		if e != nil {
			return nil, e
		}
		user = &id
	case domain.PrincipalPublic:
		if p.UserID != nil || !strings.HasPrefix(p.ID, "guest:") {
			return nil, domain.ErrAccessDenied
		}
	default:
		return nil, domain.ErrAccessDenied
	}
	result := make(map[string]domain.AssetUploadResponse, len(requests))
	expires := time.Now().Add(ttl).Truncate(time.Second)
	for _, req := range requests {
		if time.Until(expires) < time.Second {
			return nil, domain.ErrInvalidSceneSession
		}
		id := uuid.New()
		key := fmt.Sprintf("workspaces/%s/scenes/%s/assets/%s", ref.WorkspaceID, ref.SceneID, id)
		upload, headers, e := constrained.PresignUploadConstrained(ctx, u.bucket, key, req.MimeType, req.ByteSize, time.Until(expires))
		if e != nil {
			return nil, e
		}
		file := req.FileID
		asset := &domain.SceneAsset{ID: id, WorkspaceID: ref.WorkspaceID, SceneID: &ref.SceneID, UploadedBy: user, UploadedPrincipalID: p.ID, UploadedPrincipalType: string(p.Kind), AssetKind: domain.SceneAssetKindExcalidrawFile, FileID: &file, StorageBucket: u.bucket, StorageKey: key, MimeType: req.MimeType, ByteSize: req.ByteSize, SHA256: req.SHA256, Width: req.Width, Height: req.Height}
		if e = u.assetRepo.Create(ctx, asset); e != nil {
			return nil, e
		}
		result[file] = domain.AssetUploadResponse{AssetID: id, UploadUrl: upload, StorageKey: key, ExpiresAt: expires, Headers: headers}
	}
	if !expires.After(time.Now()) {
		return nil, domain.ErrInvalidSceneSession
	}
	return result, nil
}

func (u *assetUsecase) PresignView(ctx domain.Context, workspaceID, sceneID uuid.UUID, fileIDs []string) (map[string]domain.AssetViewResponse, error) {
	// Verify scene ownership
	_, err := u.sceneRepo.FindByID(ctx, workspaceID, sceneID)
	if err != nil {
		return nil, fmt.Errorf("scene not found or unauthorized: %w", err)
	}

	assets, err := u.assetRepo.GetByFileIDs(ctx, workspaceID, sceneID, fileIDs)
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
		if asset.PresignedURL != nil && asset.URLExpiresAt != nil && asset.URLExpiresAt.After(now.Add(5*time.Second)) && !asset.URLExpiresAt.After(now.Add(u.urlExpiry)) {
			viewUrl = *asset.PresignedURL
			expiresAt = *asset.URLExpiresAt
		} else {
			// Generate new URL
			newUrl, err := u.s3Repo.PresignDownload(ctx, asset.StorageBucket, asset.StorageKey, time.Until(now.Add(u.urlExpiry)))
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

func validateAssets(requests []domain.AssetUploadRequest) error {
	if len(requests) == 0 || len(requests) > domain.MaxAssetBatch {
		return domain.ErrInvalidCapability
	}
	seen := map[string]bool{}
	v := validator.New()
	for _, r := range requests {
		if v.Struct(r) != nil || seen[r.FileID] {
			return domain.ErrInvalidCapability
		}
		seen[r.FileID] = true
		switch r.MimeType {
		case "image/png", "image/jpeg", "image/gif", "image/webp":
		default:
			return domain.ErrInvalidCapability
		}
		if r.SHA256 != nil && v.Var(*r.SHA256, "len=64,hexadecimal,lowercase") != nil {
			return domain.ErrInvalidCapability
		}
		if r.Width != nil && (*r.Width < 1 || *r.Width > 32768) || r.Height != nil && (*r.Height < 1 || *r.Height > 32768) {
			return domain.ErrInvalidCapability
		}
	}
	return nil
}

func (u *assetUsecase) PresignScopedView(ctx domain.Context, ref domain.SceneRef, files []string, ttl time.Duration) (map[string]domain.AssetViewResponse, error) {
	if ttl < time.Second {
		return nil, domain.ErrInvalidSceneSession
	}
	copy := *u
	if ttl < copy.urlExpiry {
		copy.urlExpiry = ttl
	}
	return copy.PresignView(ctx, ref.WorkspaceID, ref.SceneID, files)
}
