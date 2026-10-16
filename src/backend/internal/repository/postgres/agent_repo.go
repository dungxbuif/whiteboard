package postgres

import (
	"devtool/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"slices"
	"time"
)

type AgentRepository struct{ db *gorm.DB }

func NewAgentRepository(db *gorm.DB) *AgentRepository { return &AgentRepository{db} }
func (r *AgentRepository) dbFor(ctx domain.Context) *gorm.DB {
	return accessDB(ctx, r.db).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
}
func (r *AgentRepository) Create(ctx domain.Context, k *domain.AgentKey) error {
	return r.dbFor(ctx).Create(k).Error
}
func (r *AgentRepository) active(ctx domain.Context) *gorm.DB {
	return r.dbFor(ctx).Model(&domain.AgentKey{}).Select("agent_keys.*").Joins("JOIN users u ON u.id=agent_keys.owner_id AND u.deleted_at IS NULL").Joins("JOIN workspaces w ON w.id=agent_keys.workspace_id AND w.deleted_at IS NULL").Joins("JOIN workspace_members m ON m.workspace_id=w.id AND m.user_id=u.id").Where("agent_keys.revoked_at IS NULL")
}
func (r *AgentRepository) FindHash(ctx domain.Context, h string) (domain.AgentKey, error) {
	var k domain.AgentKey
	e := r.active(ctx).Where("token_hash=?", h).First(&k).Error
	if e != nil {
		return k, domain.ErrAgentCredential
	}
	return k, nil
}
func (r *AgentRepository) FindActive(ctx domain.Context, id uuid.UUID) (domain.AgentKey, error) {
	var k domain.AgentKey
	e := r.active(ctx).Where("agent_keys.id=?", id).First(&k).Error
	if e != nil {
		return k, domain.ErrAgentCredential
	}
	return k, nil
}
func (r *AgentRepository) List(ctx domain.Context, w uuid.UUID) ([]domain.AgentKey, error) {
	out := []domain.AgentKey{}
	e := r.dbFor(ctx).Select("id,workspace_id,owner_id,scopes,scene_ids,created_at,revoked_at").Where("workspace_id=?", w).Order("created_at,id").Find(&out).Error
	return out, e
}
func (r *AgentRepository) Revoke(ctx domain.Context, w, id uuid.UUID) error {
	q := r.dbFor(ctx).Model(&domain.AgentKey{}).Where("workspace_id=? AND id=? AND revoked_at IS NULL", w, id).Update("revoked_at", time.Now())
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected != 1 {
		return domain.ErrInaccessible
	}
	return nil
}
func (r *AgentRepository) Workspace(ctx domain.Context, w uuid.UUID) (domain.AgentWorkspace, error) {
	var x domain.AgentWorkspace
	e := r.dbFor(ctx).Model(&domain.Workspace{}).Select("id,name").Where("id=?", w).First(&x).Error
	return x, accessError(e)
}
func (r *AgentRepository) Scenes(ctx domain.Context, k domain.AgentKey, c uuid.UUID, n int) ([]domain.AgentScene, error) {
	out := []domain.AgentScene{}
	q := r.dbFor(ctx).Model(&domain.Scene{}).Select("id,workspace_id,title,content_version").Where("workspace_id=?", k.WorkspaceID)
	if len(k.SceneIDs) > 0 {
		q = q.Where("id IN ?", k.SceneIDs)
	}
	if c != uuid.Nil {
		q = q.Where("id > ?", c)
	}
	e := q.Order("id").Limit(n).Find(&out).Error
	return out, e
}
func (r *AgentRepository) Scene(ctx domain.Context, ref domain.SceneRef) (domain.AgentScene, error) {
	var x domain.AgentScene
	e := r.dbFor(ctx).Model(&domain.Scene{}).Select("id,workspace_id,title,content,content_version").Where("id=? AND workspace_id=?", ref.SceneID, ref.WorkspaceID).First(&x).Error
	return x, accessError(e)
}
func (r *AgentRepository) CreateScene(ctx domain.Context, s *domain.Scene) error {
	return r.dbFor(ctx).Omit("WriterMode").Create(s).Error
}
func (r *AccessRepository) AgentSceneState(ctx domain.Context, p domain.Principal, ref domain.SceneRef) (domain.AccessDecision, error) {
	if p.UserID != nil {
		return domain.AccessDecision{}, domain.ErrAccessDenied
	}
	id, e := uuid.Parse(p.GrantID)
	if e != nil || id == uuid.Nil || p.ID != "agent:"+id.String() {
		return domain.AccessDecision{}, domain.ErrAccessDenied
	}
	k, e := NewAgentRepository(r.db).FindActive(ctx, id)
	if e != nil {
		return domain.AccessDecision{}, e
	}
	if k.WorkspaceID != ref.WorkspaceID || (len(k.SceneIDs) > 0 && !slices.Contains(k.SceneIDs, ref.SceneID)) {
		return domain.AccessDecision{}, domain.ErrInaccessible
	}
	state, e := r.SceneState(ctx, k.OwnerID, ref)
	if e != nil {
		return domain.AccessDecision{}, e
	}
	role := state.Role
	if role != domain.WorkspaceRoleOwner && state.Override != "" {
		role = state.Override
	}
	edit := role == domain.WorkspaceRoleOwner || role == domain.WorkspaceRoleEditor
	if slices.Contains(k.Scopes, domain.ScopeSceneWrite) && edit {
		return domain.AccessDecision{Access: domain.AccessEdit, ACLVersion: state.ACLVersion}, nil
	}
	if (slices.Contains(k.Scopes, domain.ScopeSceneRead) || slices.Contains(k.Scopes, domain.ScopeSceneExport)) && (edit || role == domain.WorkspaceRoleViewer) {
		return domain.AccessDecision{Access: domain.AccessView, ACLVersion: state.ACLVersion}, nil
	}
	return domain.AccessDecision{}, domain.ErrAccessDenied
}

func (r *AccessRepository) RequireAgentScope(ctx domain.Context, p domain.Principal, ref domain.SceneRef, scope string) error {
	if p.Kind != domain.PrincipalAgent || p.UserID != nil {
		return domain.ErrAccessDenied
	}
	id, e := uuid.Parse(p.GrantID)
	if e != nil || p.ID != "agent:"+id.String() {
		return domain.ErrAccessDenied
	}
	k, e := NewAgentRepository(r.db).FindActive(ctx, id)
	if e != nil {
		return e
	}
	if k.WorkspaceID != ref.WorkspaceID || (len(k.SceneIDs) > 0 && !slices.Contains(k.SceneIDs, ref.SceneID)) {
		return domain.ErrInaccessible
	}
	if !slices.Contains(k.Scopes, scope) {
		return domain.ErrAccessDenied
	}
	state, e := r.SceneState(ctx, k.OwnerID, ref)
	if e != nil {
		return e
	}
	role := state.Role
	if role != domain.WorkspaceRoleOwner && state.Override != "" {
		role = state.Override
	}
	if scope == domain.ScopeSceneWrite || scope == domain.ScopeSceneMove || scope == domain.ScopeSceneTrash {
		if role != domain.WorkspaceRoleOwner && role != domain.WorkspaceRoleEditor {
			return domain.ErrAccessDenied
		}
	}
	return nil
}

// Audit identity survives revocation so a previously authenticated request can
// record its denial without turning that historical identity into authority.
func (r *AgentRepository) FindAuditKey(ctx domain.Context, id uuid.UUID) (domain.AgentKey, error) {
	var k domain.AgentKey
	e := r.dbFor(ctx).Select("id,workspace_id").Where("id=?", id).First(&k).Error
	return k, accessError(e)
}
func (r *AgentRepository) RecordOperation(ctx domain.Context, event *domain.AgentOperationAudit) error {
	return r.dbFor(ctx).Create(event).Error
}

func (r *AgentRepository) RecordOperationFailure(ctx domain.Context, event *domain.AgentOperationAudit) error {
	// Bypass any inherited access transaction: its eventual rollback must not
	// discard a failure record. Audit metadata has no FK lock dependencies.
	return r.db.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Create(event).Error
}
