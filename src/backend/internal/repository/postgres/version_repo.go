package postgres

import (
	"devtool/internal/domain"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VersionRepository struct{ db *gorm.DB }

func NewVersionRepository(db *gorm.DB) *VersionRepository { return &VersionRepository{db: db} }

type sceneVersionRow struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	SceneID          uuid.UUID
	DocumentID       uuid.UUID
	SourceSequence   string
	Content          json.RawMessage
	ContentHash      string
	Label            *string
	Pinned           bool
	Kind             string
	CreatorKind      domain.PrincipalKind
	CreatorID        string
	MetadataRevision int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (sceneVersionRow) TableName() string { return "scene_versions" }
func versionFromRow(row sceneVersionRow) domain.SceneVersion {
	return domain.SceneVersion{ID: row.ID, WorkspaceID: row.WorkspaceID, SceneID: row.SceneID, DocumentID: row.DocumentID, Sequence: row.SourceSequence, Hash: row.ContentHash, Content: row.Content, Label: row.Label, Pinned: row.Pinned, Kind: row.Kind, Creator: domain.VersionCreator{Kind: row.CreatorKind, ID: row.CreatorID}, MetadataRevision: row.MetadataRevision, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func (r *VersionRepository) ListVersions(ctx domain.Context, ref domain.SceneRef, cursor *uuid.UUID, limit int) ([]domain.SceneVersion, error) {
	rows := []sceneVersionRow{}
	db := accessDB(ctx, r.db).Where("workspace_id=? AND scene_id=?", ref.WorkspaceID, ref.SceneID).Select("id,workspace_id,scene_id,document_id,source_sequence::text AS source_sequence,content_hash,label,pinned,kind,creator_kind,creator_id,metadata_revision,created_at,updated_at")
	if cursor != nil {
		db = db.Where("(created_at,id)<(SELECT created_at,id FROM scene_versions WHERE workspace_id=? AND scene_id=? AND id=?)", ref.WorkspaceID, ref.SceneID, *cursor)
	}
	if err := db.Order("created_at DESC,id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.SceneVersion, len(rows))
	for i, row := range rows {
		items[i] = versionFromRow(row)
	}
	return items, nil
}
func (r *VersionRepository) Version(ctx domain.Context, ref domain.SceneRef, id uuid.UUID) (domain.SceneVersion, error) {
	var row sceneVersionRow
	err := accessDB(ctx, r.db).Where("workspace_id=? AND scene_id=? AND id=?", ref.WorkspaceID, ref.SceneID, id).Select("id,workspace_id,scene_id,document_id,source_sequence::text AS source_sequence,content,content_hash,label,pinned,kind,creator_kind,creator_id,metadata_revision,created_at,updated_at").First(&row).Error
	if err != nil {
		return domain.SceneVersion{}, accessError(err)
	}
	return versionFromRow(row), nil
}
func (r *VersionRepository) SetVersionMetadata(ctx domain.Context, ref domain.SceneRef, id uuid.UUID, expected int64, label *string, pinned *bool) (domain.SceneVersion, error) {
	db, err := requireCommentTransaction(ctx, ref.WorkspaceID)
	if err != nil {
		return domain.SceneVersion{}, err
	}
	updates := map[string]any{"metadata_revision": gorm.Expr("metadata_revision+1")}
	if label != nil {
		updates["label"] = *label
	}
	if pinned != nil {
		updates["pinned"] = *pinned
	}
	result := db.Model(&sceneVersionRow{}).Where("workspace_id=? AND scene_id=? AND id=? AND metadata_revision=?", ref.WorkspaceID, ref.SceneID, id, expected).Updates(updates)
	if result.Error != nil {
		return domain.SceneVersion{}, result.Error
	}
	if result.RowsAffected != 1 {
		var count int64
		if e := db.Model(&sceneVersionRow{}).Where("workspace_id=? AND scene_id=? AND id=?", ref.WorkspaceID, ref.SceneID, id).Count(&count).Error; e != nil {
			return domain.SceneVersion{}, e
		}
		if count == 0 {
			return domain.SceneVersion{}, domain.ErrInaccessible
		}
		return domain.SceneVersion{}, domain.ErrRevisionConflict
	}
	return r.Version(ctx, ref, id)
}
