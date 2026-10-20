package postgres

import (
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LibraryAssetStager struct {
	db      *gorm.DB
	objects domain.LibraryObjectCopier
	grace   time.Duration
}

func NewLibraryAssetStager(db *gorm.DB, objects domain.LibraryObjectCopier, grace time.Duration) *LibraryAssetStager {
	if grace < time.Hour {
		grace = 24 * time.Hour
	}
	return &LibraryAssetStager{db: db, objects: objects, grace: grace}
}

type libraryStageRow struct {
	WorkspaceID   uuid.UUID
	TargetSceneID uuid.UUID
	RequestID     uuid.UUID
	SourceAssetID uuid.UUID
	SourceFileID  string
	TargetAssetID uuid.UUID
	TargetFileID  string
	StorageBucket string
	StorageKey    string
	State         string
	CleanupAfter  time.Time
}

func (libraryStageRow) TableName() string { return "library_asset_stages" }

func libraryTargetFile(requestID uuid.UUID, fileID string) string {
	sum := sha256.Sum256([]byte(requestID.String() + ":file:" + fileID))
	return "l_" + hex.EncodeToString(sum[:])[:30]
}
func (s *LibraryAssetStager) Stage(ctx domain.Context, principal domain.Principal, target domain.SceneRef, requestID uuid.UUID, pins []domain.LibraryAssetPin) (map[string]uuid.UUID, error) {
	if s == nil || s.db == nil || s.objects == nil || requestID == uuid.Nil {
		return nil, domain.ErrInvalidLibrary
	}
	result := make(map[string]uuid.UUID, len(pins))
	for _, pin := range pins {
		targetFile := libraryTargetFile(requestID, pin.FileID)
		targetAsset := uuid.NewSHA1(uuid.NameSpaceOID, []byte(target.WorkspaceID.String()+":"+target.SceneID.String()+":"+requestID.String()+":"+pin.FileID))
		var source domain.SceneAsset
		if err := s.db.WithContext(ctx).Where("workspace_id=? AND scene_id=? AND id=? AND file_id=? AND deleted_at IS NULL", target.WorkspaceID, pin.SourceSceneID, pin.AssetID, pin.FileID).First(&source).Error; err != nil {
			return nil, accessError(err)
		}
		keySum := sha256.Sum256([]byte(target.WorkspaceID.String() + ":" + target.SceneID.String() + ":" + requestID.String() + ":" + pin.FileID))
		targetKey := "library-staged/" + target.WorkspaceID.String() + "/" + target.SceneID.String() + "/" + hex.EncodeToString(keySum[:])
		stage := libraryStageRow{WorkspaceID: target.WorkspaceID, TargetSceneID: target.SceneID, RequestID: requestID, SourceAssetID: pin.AssetID, SourceFileID: pin.FileID, TargetAssetID: targetAsset, TargetFileID: targetFile, StorageBucket: source.StorageBucket, StorageKey: targetKey, State: "pending", CleanupAfter: time.Now().Add(s.grace)}
		if err := s.db.WithContext(ctx).Exec(`INSERT INTO library_asset_stages(workspace_id,target_scene_id,request_id,source_asset_id,source_file_id,target_asset_id,target_file_id,storage_bucket,storage_key,state,cleanup_after) VALUES(?,?,?,?,?,?,?,?,?,'pending',?) ON CONFLICT(workspace_id,target_scene_id,request_id,source_file_id) DO NOTHING`, stage.WorkspaceID, stage.TargetSceneID, stage.RequestID, stage.SourceAssetID, stage.SourceFileID, stage.TargetAssetID, stage.TargetFileID, stage.StorageBucket, stage.StorageKey, stage.CleanupAfter).Error; err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Where("workspace_id=? AND target_scene_id=? AND request_id=? AND source_file_id=?", target.WorkspaceID, target.SceneID, requestID, pin.FileID).First(&stage).Error; err != nil {
			return nil, err
		}
		if stage.SourceAssetID != pin.AssetID || stage.TargetFileID != targetFile {
			return nil, domain.ErrRequestIDReused
		}
		var existing domain.SceneAsset
		existingErr := s.db.WithContext(ctx).Where("workspace_id=? AND scene_id=? AND id=? AND file_id=? AND deleted_at IS NULL", target.WorkspaceID, target.SceneID, stage.TargetAssetID, stage.TargetFileID).First(&existing).Error
		if errors.Is(existingErr, gorm.ErrRecordNotFound) {
			if err := s.objects.CopyObject(ctx, source.StorageBucket, source.StorageKey, stage.StorageBucket, stage.StorageKey); err != nil {
				return nil, err
			}
			if err := s.db.WithContext(ctx).Exec(`INSERT INTO scene_assets(id,workspace_id,scene_id,uploaded_principal_id,uploaded_principal_type,asset_kind,file_id,storage_bucket,storage_key,mime_type,byte_size,sha256,width,height)
VALUES(?,?,?, ?,?,'excalidraw_file',?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`, stage.TargetAssetID, target.WorkspaceID, target.SceneID, principal.ID, string(principal.Kind), stage.TargetFileID, stage.StorageBucket, stage.StorageKey, source.MimeType, source.ByteSize, source.SHA256, source.Width, source.Height).Error; err != nil {
				return nil, err
			}
			if err := s.db.WithContext(ctx).Where("workspace_id=? AND scene_id=? AND id=? AND file_id=? AND storage_bucket=? AND storage_key=? AND deleted_at IS NULL", target.WorkspaceID, target.SceneID, stage.TargetAssetID, stage.TargetFileID, stage.StorageBucket, stage.StorageKey).First(&existing).Error; err != nil {
				return nil, accessError(err)
			}
		} else if existingErr != nil {
			return nil, existingErr
		}
		if err := s.db.WithContext(ctx).Model(&libraryStageRow{}).Where("workspace_id=? AND target_scene_id=? AND request_id=? AND source_file_id=?", target.WorkspaceID, target.SceneID, requestID, pin.FileID).Updates(map[string]any{"state": "ready", "updated_at": gorm.Expr("clock_timestamp()")}).Error; err != nil {
			return nil, err
		}
		result[pin.FileID] = stage.TargetAssetID
	}
	return result, nil
}
