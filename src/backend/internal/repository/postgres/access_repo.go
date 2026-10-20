package postgres

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type accessTransactionKey struct{}
type accessTransaction struct {
	db        *gorm.DB
	workspace uuid.UUID
}

func accessDB(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(accessTransactionKey{}).(accessTransaction); ok {
		return tx.db.WithContext(ctx)
	}
	return db.WithContext(ctx)
}

type AccessRepository struct{ db *gorm.DB }

func NewAccessRepository(db *gorm.DB) *AccessRepository { return &AccessRepository{db: db} }
func accessError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrInaccessible
	}
	return err
}
func (r *AccessRepository) WorkspaceState(ctx context.Context, user, w uuid.UUID) (domain.AccessState, error) {
	var state domain.AccessState
	result := accessDB(ctx, r.db).Raw(`SELECT m.role,w.acl_version FROM workspaces w JOIN workspace_members m ON m.workspace_id=w.id JOIN users u ON u.id=m.user_id AND u.deleted_at IS NULL WHERE w.id=? AND m.user_id=? AND w.deleted_at IS NULL`, w, user).Scan(&state)
	if result.Error != nil {
		return state, result.Error
	}
	if result.RowsAffected == 0 {
		return state, domain.ErrInaccessible
	}
	return state, nil
}
func (r *AccessRepository) SceneState(ctx context.Context, user uuid.UUID, ref domain.SceneRef) (domain.AccessState, error) {
	var state domain.AccessState
	result := accessDB(ctx, r.db).Raw(`SELECT m.role,COALESCE(o.role,'') AS override,w.acl_version FROM scenes s JOIN workspaces w ON w.id=s.workspace_id JOIN workspace_members m ON m.workspace_id=w.id AND m.user_id=? JOIN users u ON u.id=m.user_id AND u.deleted_at IS NULL LEFT JOIN scene_member_overrides o ON o.scene_id=s.id AND o.user_id=m.user_id AND o.workspace_id=w.id WHERE s.id=? AND w.id=? AND s.deleted_at IS NULL AND w.deleted_at IS NULL`, user, ref.SceneID, ref.WorkspaceID).Scan(&state)
	if result.Error != nil {
		return state, result.Error
	}
	if result.RowsAffected == 0 {
		return state, domain.ErrInaccessible
	}
	return state, nil
}
func (r *AccessRepository) ResolveWorkspaceID(ctx context.Context, raw string) (uuid.UUID, error) {
	var w domain.Workspace
	q := accessDB(ctx, r.db)
	if id, e := uuid.Parse(raw); e == nil {
		q = q.Where("id=?", id)
	} else {
		q = q.Where("public_id=?", raw)
	}
	err := q.First(&w).Error
	return w.ID, accessError(err)
}
func (r *AccessRepository) ResolveSceneID(ctx context.Context, w uuid.UUID, raw string) (uuid.UUID, error) {
	var scene domain.Scene
	q := accessDB(ctx, r.db).Unscoped().Where("workspace_id=?", w)
	if id, e := uuid.Parse(raw); e == nil {
		q = q.Where("id=?", id)
	} else {
		q = q.Where("public_id=?", raw)
	}
	err := q.First(&scene).Error
	return scene.ID, accessError(err)
}
func (r *AccessRepository) WithWorkspaceLock(ctx context.Context, w, actor uuid.UUID, fn func(context.Context) error) error {
	if current, ok := ctx.Value(accessTransactionKey{}).(accessTransaction); ok {
		if current.workspace != w {
			return domain.ErrAccessDenied
		}
		return fn(ctx)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var workspace domain.Workspace
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&workspace, "id=?", w).Error; e != nil {
			return accessError(e)
		}
		if actor != uuid.Nil {
			if e := tx.Exec("SELECT set_config('app.actor_id', ?, true)", actor.String()).Error; e != nil {
				return e
			}
		}
		return fn(context.WithValue(ctx, accessTransactionKey{}, accessTransaction{db: tx, workspace: w}))
	})
}
func (r *AccessRepository) ListMembers(ctx context.Context, w uuid.UUID) ([]domain.WorkspaceMember, error) {
	members := []domain.WorkspaceMember{}
	err := accessDB(ctx, r.db).Where("workspace_id=?", w).Order("created_at,user_id").Find(&members).Error
	return members, err
}
func (r *AccessRepository) ChangeMember(ctx context.Context, w, user uuid.UUID, role *domain.WorkspaceRole) error {
	db := accessDB(ctx, r.db)
	q := db.Model(&domain.WorkspaceMember{}).Where("workspace_id=? AND user_id=? AND role <> 'owner'", w, user)
	var managed scimUserRow
	available, availabilityErr := scimProvenanceAvailable(db)
	if availabilityErr != nil {
		return availabilityErr
	}
	managedErr := gorm.ErrRecordNotFound
	if available {
		managedErr = db.Where("workspace_id=? AND user_id=?", w, user).First(&managed).Error
	}
	if managedErr == nil {
		if role == nil {
			if e := db.Where("workspace_id=? AND user_id=? AND source_kind='manual'", w, user).Delete(&membershipSourceRow{}).Error; e != nil {
				return e
			}
			if e := db.Model(&scimUserRow{}).Where("workspace_id=? AND user_id=?", w, user).Update("retained_manual_access", false).Error; e != nil {
				return e
			}
			var managedSources int64
			if e := db.Model(&membershipSourceRow{}).Where("workspace_id=? AND user_id=? AND source_kind IN ('direct','adopted')", w, user).Count(&managedSources).Error; e != nil {
				return e
			}
			if managed.Active && managedSources == 0 {
				if e := db.Create(&membershipSourceRow{WorkspaceID: w, UserID: user, SourceKind: domain.SCIMMembershipSourceDirect, SourceID: managed.ID, Role: managed.DirectRole, CreatedAt: time.Now().UTC()}).Error; e != nil {
					return e
				}
			}
			return recomputeMembership(db, w, user, false, time.Now().UTC())
		}
		source := membershipSourceRow{WorkspaceID: w, UserID: user, SourceKind: domain.SCIMMembershipSourceManual, SourceID: user, Role: string(*role), CreatedAt: time.Now().UTC()}
		if e := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "workspace_id"}, {Name: "user_id"}, {Name: "source_kind"}, {Name: "source_id"}}, DoUpdates: clause.AssignmentColumns([]string{"role"})}).Create(&source).Error; e != nil {
			return e
		}
		if e := db.Model(&scimUserRow{}).Where("workspace_id=? AND user_id=?", w, user).Update("retained_manual_access", true).Error; e != nil {
			return e
		}
		return recomputeMembership(db, w, user, false, time.Now().UTC())
	}
	if managedErr != nil && !errors.Is(managedErr, gorm.ErrRecordNotFound) {
		return managedErr
	}
	var result *gorm.DB
	if role == nil {
		result = q.Delete(&domain.WorkspaceMember{})
	} else {
		result = q.Update("role", *role)
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrInaccessible
	}
	return nil
}
func (r *AccessRepository) ChangeOverride(ctx context.Context, ref domain.SceneRef, user uuid.UUID, role *domain.WorkspaceRole) error {
	db := accessDB(ctx, r.db)
	if role == nil {
		return db.Where("workspace_id=? AND scene_id=? AND user_id=?", ref.WorkspaceID, ref.SceneID, user).Delete(&domain.SceneMemberOverride{}).Error
	}
	row := domain.SceneMemberOverride{WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, UserID: user, Role: *role}
	return db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "scene_id"}, {Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"role"})}).Create(&row).Error
}
func (r *AccessRepository) DescendantScenes(ctx context.Context, w, folder uuid.UUID) ([]uuid.UUID, error) {
	var f domain.Folder
	db := accessDB(ctx, r.db)
	if e := db.First(&f, "workspace_id=? AND id=?", w, folder).Error; e != nil {
		return nil, accessError(e)
	}
	ids := []uuid.UUID{}
	err := db.Model(&domain.Scene{}).Where("workspace_id=? AND folder_id IN (SELECT id FROM folders WHERE workspace_id=? AND path LIKE ?)", w, w, f.Path+"%").Pluck("id", &ids).Error
	return ids, err
}
