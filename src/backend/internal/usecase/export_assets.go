package usecase

import (
	"context"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"net/http"
	"time"
)

type ExportAssets struct {
	agents  *AgentService
	assets  domain.SceneAssetRepository
	objects domain.ExportObjectReader
}

func NewExportAssets(a *AgentService, assets domain.SceneAssetRepository, objects domain.ExportObjectReader) *ExportAssets {
	return &ExportAssets{a, assets, objects}
}
func (s *ExportAssets) Get(ctx domain.Context, p domain.Principal, ref domain.SceneRef, fileID, assetID string) (domain.ExportAsset, error) {
	var result domain.ExportAsset
	id, e := uuid.Parse(assetID)
	if e != nil || validator.New().Var(fileID, "required,max=128") != nil {
		return result, domain.ErrAgentInput
	}
	lookup := func(ctx domain.Context) (domain.SceneAsset, error) {
		if e := s.agents.RequireScope(ctx, p, ref, domain.ScopeSceneExport); e != nil {
			return domain.SceneAsset{}, e
		}
		rows, e := s.assets.GetByFileIDs(ctx, ref.WorkspaceID, ref.SceneID, []string{fileID})
		if e != nil {
			return domain.SceneAsset{}, e
		}
		for _, r := range rows {
			if r.ID == id && r.WorkspaceID == ref.WorkspaceID && r.SceneID != nil && *r.SceneID == ref.SceneID && r.FileID != nil && *r.FileID == fileID && r.DeletedAt == nil {
				return r, nil
			}
		}
		return domain.SceneAsset{}, domain.ErrInaccessible
	}
	var row domain.SceneAsset
	if e = s.agents.access.WithWorkspaceLock(ctx, ref.WorkspaceID, uuid.Nil, func(ctx domain.Context) error { var err error; row, err = lookup(ctx); return err }); e != nil {
		return result, e
	}
	switch row.MimeType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
	default:
		return result, domain.ErrAgentInput
	}
	if row.ByteSize < 1 || row.ByteSize > domain.ExportAssetMaxBytes {
		return result, domain.ErrAgentProjection
	}
	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	data, e := s.objects.ReadExportObject(readCtx, row.StorageBucket, row.StorageKey, domain.ExportAssetMaxBytes)
	if e != nil {
		return result, e
	}
	if int64(len(data)) != row.ByteSize || http.DetectContentType(data) != row.MimeType {
		return result, domain.ErrAgentInput
	}
	if row.SHA256 != nil {
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != *row.SHA256 {
			return result, domain.ErrAgentInput
		}
	}
	e = s.agents.access.WithWorkspaceLock(ctx, ref.WorkspaceID, uuid.Nil, func(ctx domain.Context) error {
		current, err := lookup(ctx)
		if err != nil {
			return err
		}
		if current.StorageKey != row.StorageKey || current.StorageBucket != row.StorageBucket || current.MimeType != row.MimeType || current.ByteSize != row.ByteSize {
			return domain.ErrInaccessible
		}
		result = domain.ExportAsset{MimeType: row.MimeType, Data: data}
		return nil
	})
	return result, e
}
