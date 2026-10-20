package postgres

import (
	"devtool/internal/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PresentationRepository struct{ db *gorm.DB }

func NewPresentationRepository(db *gorm.DB) *PresentationRepository {
	return &PresentationRepository{db: db}
}

func (r *PresentationRepository) CreateLink(ctx domain.Context, link *domain.PresentationLink) error {
	return accessDB(ctx, r.db).Create(link).Error
}

func (r *PresentationRepository) FindLinkByHash(ctx domain.Context, hash string) (*domain.PresentationLink, error) {
	var link domain.PresentationLink
	err := accessDB(ctx, r.db).Where("token_hash=? AND active=true AND expires_at>clock_timestamp()", hash).First(&link).Error
	return &link, accessError(err)
}

func (r *PresentationRepository) FindLink(ctx domain.Context, ref domain.SceneRef, sessionID, id uuid.UUID) (*domain.PresentationLink, error) {
	var link domain.PresentationLink
	err := accessDB(ctx, r.db).Where("workspace_id=? AND scene_id=? AND session_id=? AND id=?", ref.WorkspaceID, ref.SceneID, sessionID, id).First(&link).Error
	return &link, accessError(err)
}

func (r *PresentationRepository) RevokeLink(ctx domain.Context, ref domain.SceneRef, sessionID, id uuid.UUID) error {
	now := time.Now().UTC()
	result := accessDB(ctx, r.db).Model(&domain.PresentationLink{}).Where("workspace_id=? AND scene_id=? AND session_id=? AND id=? AND active=true", ref.WorkspaceID, ref.SceneID, sessionID, id).Updates(map[string]any{"active": false, "revoked_at": now, "revision": gorm.Expr("revision + 1")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrInaccessible
	}
	return nil
}
