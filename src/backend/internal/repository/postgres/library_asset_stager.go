package postgres

import (
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"encoding/json"
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

// StageSceneCopy copies every persisted scene asset and updates the Excalidraw
// file and image references to point at the copied, scene-owned assets.
func (s *LibraryAssetStager) StageSceneCopy(ctx domain.Context, principal domain.Principal, source, target domain.SceneRef, requestID uuid.UUID, content json.RawMessage) (json.RawMessage, error) {
	if s == nil || s.db == nil || source.WorkspaceID != target.WorkspaceID || !json.Valid(content) {
		return nil, domain.ErrInvalidLibrary
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(content, &document); err != nil {
		return nil, err
	}
	var files map[string]json.RawMessage
	if err := json.Unmarshal(document["files"], &files); err != nil {
		return nil, domain.ErrInvalidLibrary
	}
	pins := make([]domain.LibraryAssetPin, 0, len(files))
	for fileID, raw := range files {
		var file struct {
			AssetID string `json:"assetId"`
		}
		if err := json.Unmarshal(raw, &file); err != nil {
			return nil, domain.ErrInvalidLibrary
		}
		var assetID uuid.UUID
		if file.AssetID != "" {
			var err error
			assetID, err = uuid.Parse(file.AssetID)
			if err != nil {
				return nil, domain.ErrInvalidLibrary
			}
		} else {
			var sourceAsset domain.SceneAsset
			err := s.db.WithContext(ctx).Where("workspace_id=? AND scene_id=? AND file_id=? AND deleted_at IS NULL", source.WorkspaceID, source.SceneID, fileID).First(&sourceAsset).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			} // inline, unscoped legacy file
			if err != nil {
				return nil, err
			}
			assetID = sourceAsset.ID
		}
		pins = append(pins, domain.LibraryAssetPin{SourceSceneID: source.SceneID, AssetID: assetID, FileID: fileID})
	}
	staged, err := s.Stage(ctx, principal, target, requestID, pins)
	if err != nil {
		return nil, err
	}
	fileIDs := make(map[string]string, len(staged))
	for oldID, assetID := range staged {
		newID := libraryTargetFile(requestID, oldID)
		var file map[string]json.RawMessage
		if err := json.Unmarshal(files[oldID], &file); err != nil {
			return nil, domain.ErrInvalidLibrary
		}
		file["id"], _ = json.Marshal(newID)
		file["assetId"], _ = json.Marshal(assetID.String())
		encoded, err := json.Marshal(file)
		if err != nil {
			return nil, err
		}
		delete(files, oldID)
		files[newID] = encoded
		fileIDs[oldID] = newID
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(document["elements"], &elements); err != nil {
		return nil, domain.ErrInvalidLibrary
	}
	for i, raw := range elements {
		var element map[string]json.RawMessage
		if err := json.Unmarshal(raw, &element); err != nil {
			return nil, domain.ErrInvalidLibrary
		}
		var oldID string
		if json.Unmarshal(element["fileId"], &oldID) == nil {
			if newID, ok := fileIDs[oldID]; ok {
				element["fileId"], _ = json.Marshal(newID)
				elements[i], err = json.Marshal(element)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	document["files"], err = json.Marshal(files)
	if err != nil {
		return nil, err
	}
	document["elements"], err = json.Marshal(elements)
	if err != nil {
		return nil, err
	}
	return json.Marshal(document)
}

// CleanupSceneCopy removes staged object copies before their rows. A failure
// leaves the hidden scene and stage records available for a later retry.
func (s *LibraryAssetStager) CleanupSceneCopy(ctx domain.Context, target domain.SceneRef, requestID uuid.UUID) error {
	if s == nil || s.db == nil || s.objects == nil {
		return domain.ErrInvalidLibrary
	}
	var claimed int64
	if err := s.db.WithContext(ctx).Unscoped().Model(&domain.Scene{}).Where("workspace_id=? AND id=? AND pending_duplicate=TRUE AND duplicate_cleanup_started=TRUE AND deleted_at IS NOT NULL", target.WorkspaceID, target.SceneID).Count(&claimed).Error; err != nil {
		return err
	}
	if claimed != 1 {
		return domain.ErrInaccessible
	}
	var rows []libraryStageRow
	if err := s.db.WithContext(ctx).Where("workspace_id=? AND target_scene_id=? AND request_id=?", target.WorkspaceID, target.SceneID, requestID).Find(&rows).Error; err != nil {
		return err
	}
	var deleteErr error
	for _, row := range rows {
		if err := s.objects.Delete(ctx, row.StorageBucket, row.StorageKey); err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	if deleteErr != nil {
		return deleteErr
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if err := tx.Exec("DELETE FROM scene_assets WHERE workspace_id=? AND scene_id=? AND id=?", target.WorkspaceID, target.SceneID, row.TargetAssetID).Error; err != nil {
				return err
			}
		}
		return tx.Exec("DELETE FROM library_asset_stages WHERE workspace_id=? AND target_scene_id=? AND request_id=?", target.WorkspaceID, target.SceneID, requestID).Error
	})
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
