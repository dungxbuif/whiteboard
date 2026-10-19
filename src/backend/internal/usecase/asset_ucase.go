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

func (u *assetUsecase) PresignUpload(ctx domain.Context, workspaceID, userID, sceneID uuid.UUID, requests []domain.AssetUploadRequest) (map[string]domain.AssetUploadResponse, error) {
	return u.PresignPrincipalUpload(ctx, domain.UserPrincipal(userID), domain.SceneRef{WorkspaceID: workspaceID, SceneID: sceneID}, requests, time.Now().Add(u.urlExpiry))
}
func (u *assetUsecase) deadline(parent time.Time) (time.Time, error) {
	now := time.Now()
	deadline := now.Add(u.urlExpiry)
	if parent.Before(deadline) {
		deadline = parent
	}
	deadline = deadline.Truncate(time.Second)
	if deadline.Sub(now) < time.Second {
		return time.Time{}, domain.ErrInvalidSceneSession
	}
	return deadline, nil
}
func assetDeadlineValid(deadline time.Time) error {
	if time.Until(deadline) < time.Second {
		return domain.ErrInvalidSceneSession
	}
	return nil
}
func equalAssetValue[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
func assetIntentMatches(asset domain.SceneAsset, request domain.AssetUploadRequest) bool {
	return asset.AssetKind == domain.SceneAssetKindExcalidrawFile && asset.MimeType == request.MimeType && asset.ByteSize == request.ByteSize && equalAssetValue(asset.SHA256, request.SHA256) && equalAssetValue(asset.Width, request.Width) && equalAssetValue(asset.Height, request.Height)
}
func (u *assetUsecase) PresignPrincipalUpload(ctx domain.Context, p domain.Principal, ref domain.SceneRef, requests []domain.AssetUploadRequest, parent time.Time) (map[string]domain.AssetUploadResponse, error) {
	if e := validateAssets(requests); e != nil {
		return nil, e
	}
	deadline, e := u.deadline(parent)
	if e != nil {
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
	if _, e = u.sceneRepo.FindByID(ctx, ref.WorkspaceID, ref.SceneID); e != nil {
		return nil, e
	}
	fileIDs := make([]string, len(requests))
	for i, r := range requests {
		fileIDs[i] = r.FileID
	}
	rows, e := u.assetRepo.GetByFileIDs(ctx, ref.WorkspaceID, ref.SceneID, fileIDs)
	if e != nil {
		return nil, e
	}
	existing := make(map[string]domain.SceneAsset, len(rows))
	for _, row := range rows {
		if row.FileID != nil && row.SceneID != nil && *row.SceneID == ref.SceneID && row.WorkspaceID == ref.WorkspaceID && row.DeletedAt == nil {
			existing[*row.FileID] = row
		}
	}
	// Validate the complete batch before signing any refresh or inserting metadata.
	for _, r := range requests {
		if row, found := existing[r.FileID]; found && !assetIntentMatches(row, r) {
			return nil, domain.ErrAssetConflict
		}
	}
	result := make(map[string]domain.AssetUploadResponse, len(requests))
	for _, r := range requests {
		if e = assetDeadlineValid(deadline); e != nil {
			return nil, e
		}
		row, found := existing[r.FileID]
		if !found {
			id := uuid.New()
			file := r.FileID
			row = domain.SceneAsset{ID: id, WorkspaceID: ref.WorkspaceID, SceneID: &ref.SceneID, UploadedBy: user, UploadedPrincipalID: p.ID, UploadedPrincipalType: string(p.Kind), AssetKind: domain.SceneAssetKindExcalidrawFile, FileID: &file, StorageBucket: u.bucket, StorageKey: fmt.Sprintf("workspaces/%s/scenes/%s/assets/%s", ref.WorkspaceID, ref.SceneID, id), MimeType: r.MimeType, ByteSize: r.ByteSize, SHA256: r.SHA256, Width: r.Width, Height: r.Height}
		}
		upload, headers, e := constrained.PresignUploadConstrained(ctx, row.StorageBucket, row.StorageKey, r.MimeType, r.ByteSize, deadline)
		if e != nil {
			return nil, e
		}
		if e = assetDeadlineValid(deadline); e != nil {
			return nil, e
		}
		if !found {
			if e = u.assetRepo.Create(ctx, &row); e != nil {
				return nil, e
			}
		}
		result[r.FileID] = domain.AssetUploadResponse{AssetID: row.ID, StorageKey: row.StorageKey, UploadUrl: upload, Headers: headers, ExpiresAt: deadline}
	}
	if e = assetDeadlineValid(deadline); e != nil {
		return nil, e
	}
	return result, nil
}
func (u *assetUsecase) PresignView(ctx domain.Context, w, scene uuid.UUID, files []string) (map[string]domain.AssetViewResponse, error) {
	return u.PresignScopedView(ctx, domain.SceneRef{WorkspaceID: w, SceneID: scene}, files, time.Now().Add(u.urlExpiry))
}
func (u *assetUsecase) PresignScopedView(ctx domain.Context, ref domain.SceneRef, files []string, parent time.Time) (map[string]domain.AssetViewResponse, error) {
	deadline, e := u.deadline(parent)
	if e != nil {
		return nil, e
	}
	signer, ok := u.s3Repo.(domain.ScopedDownloadRepository)
	if !ok {
		return nil, fmt.Errorf("object storage cannot enforce download deadline")
	}
	if _, e = u.sceneRepo.FindByID(ctx, ref.WorkspaceID, ref.SceneID); e != nil {
		return nil, e
	}
	rows, e := u.assetRepo.GetByFileIDs(ctx, ref.WorkspaceID, ref.SceneID, files)
	if e != nil {
		return nil, e
	}
	result := make(map[string]domain.AssetViewResponse, len(rows))
	for _, row := range rows {
		if e = assetDeadlineValid(deadline); e != nil {
			return nil, e
		}
		if row.FileID == nil || row.SceneID == nil || *row.SceneID != ref.SceneID || row.WorkspaceID != ref.WorkspaceID || row.DeletedAt != nil {
			continue
		}
		expires := deadline
		var signed string
		if row.PresignedURL != nil && row.URLExpiresAt != nil && row.URLExpiresAt.After(time.Now().Add(time.Second)) && !row.URLExpiresAt.After(deadline) {
			signed = *row.PresignedURL
			expires = *row.URLExpiresAt
		} else {
			signed, e = signer.PresignDownloadUntil(ctx, row.StorageBucket, row.StorageKey, deadline)
			if e != nil {
				return nil, e
			}
			if e = assetDeadlineValid(deadline); e != nil {
				return nil, e
			}
			if e = u.assetRepo.UpdatePresignedURL(ctx, row.ID, signed, expires); e != nil {
				return nil, e
			}
		}
		result[*row.FileID] = domain.AssetViewResponse{AssetID: row.ID, ViewUrl: signed, MimeType: row.MimeType, ExpiresAt: expires}
	}
	if e = assetDeadlineValid(deadline); e != nil {
		return nil, e
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
