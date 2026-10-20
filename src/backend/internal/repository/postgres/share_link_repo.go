package postgres

import (
	"context"
	"devtool/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strconv"
)

// Credential queries suppress SQL parameter logging even when a caller enables
// verbose GORM logging. HTTP errors retain safe diagnostics without secrets.
func shareDB(ctx context.Context, db *gorm.DB) *gorm.DB {
	return accessDB(ctx, db).Session(&gorm.Session{Logger: db.Logger.LogMode(logger.Silent)})
}

type ShareLinkRepository struct{ db *gorm.DB }

func NewShareLinkRepository(db *gorm.DB) *ShareLinkRepository { return &ShareLinkRepository{db: db} }
func (r *ShareLinkRepository) Create(ctx context.Context, link *domain.ShareLink) error {
	return shareDB(ctx, r.db).Create(link).Error
}
func (r *ShareLinkRepository) List(ctx context.Context, ref domain.SceneRef) ([]domain.ShareLink, error) {
	links := []domain.ShareLink{}
	e := shareDB(ctx, r.db).Select("id,access,allow_comments,active,expires_at").Where("workspace_id=? AND scene_id=?", ref.WorkspaceID, ref.SceneID).Order("created_at,id").Find(&links).Error
	return links, e
}
func (r *ShareLinkRepository) FindByHash(ctx context.Context, hash string) (*domain.ShareLink, error) {
	var link domain.ShareLink
	e := shareDB(ctx, r.db).Where("token_hash=? AND active=true AND (expires_at IS NULL OR expires_at>clock_timestamp())", hash).First(&link).Error
	return &link, accessError(e)
}
func (r *ShareLinkRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.ShareLink, error) {
	var link domain.ShareLink
	e := shareDB(ctx, r.db).Where("id=? AND active=true AND (expires_at IS NULL OR expires_at>clock_timestamp())", id).First(&link).Error
	return &link, accessError(e)
}
func (r *ShareLinkRepository) Revoke(ctx context.Context, ref domain.SceneRef, id uuid.UUID) error {
	db := shareDB(ctx, r.db)
	result := db.Model(&domain.ShareLink{}).Where("id=? AND workspace_id=? AND scene_id=? AND active=true", id, ref.WorkspaceID, ref.SceneID).Update("active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var n int64
		if e := db.Model(&domain.ShareLink{}).Where("id=? AND workspace_id=? AND scene_id=?", id, ref.WorkspaceID, ref.SceneID).Count(&n).Error; e != nil {
			return e
		}
		if n == 0 {
			return domain.ErrInaccessible
		}
	}
	return nil
}
func (r *ShareLinkRepository) UserDisplayName(ctx context.Context, id uuid.UUID) (string, error) {
	var user domain.User
	if e := shareDB(ctx, r.db).First(&user, "id=?", id).Error; e != nil {
		return "", accessError(e)
	}
	name := []rune(user.Name)
	if len(name) > domain.MaxPublicDisplayName {
		name = name[:domain.MaxPublicDisplayName]
	}
	return string(name), nil
}
func (r *AccessRepository) PublicSceneState(ctx context.Context, p domain.Principal, ref domain.SceneRef) (domain.AccessDecision, error) {
	var state struct {
		Access     string
		ACLVersion int64
		Revision   int64
	}
	result := shareDB(ctx, r.db).Raw(`SELECT l.access,w.acl_version,l.revision FROM share_links l JOIN scenes s ON s.id=l.scene_id AND s.workspace_id=l.workspace_id JOIN workspaces w ON w.id=s.workspace_id WHERE l.id=? AND l.workspace_id=? AND l.scene_id=? AND l.active=true AND (l.expires_at IS NULL OR l.expires_at>clock_timestamp()) AND s.deleted_at IS NULL AND w.deleted_at IS NULL`, p.GrantID, ref.WorkspaceID, ref.SceneID).Scan(&state)
	if result.Error != nil {
		return domain.AccessDecision{}, result.Error
	}
	if result.RowsAffected != 1 || strconv.FormatInt(state.Revision, 10) != p.GrantVersion {
		return domain.AccessDecision{}, domain.ErrInaccessible
	}
	return domain.AccessDecision{Access: state.Access, ACLVersion: state.ACLVersion}, nil
}
