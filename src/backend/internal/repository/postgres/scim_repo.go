package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SCIMRepository struct{ db *gorm.DB }

func NewSCIMRepository(db *gorm.DB) *SCIMRepository { return &SCIMRepository{db: db} }

type scimCredentialRow struct {
	WorkspaceID uuid.UUID `gorm:"primaryKey"`
	TokenHash   []byte
	TokenPrefix string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	RevokedAt   *time.Time
}

func (scimCredentialRow) TableName() string { return "workspace_scim_credentials" }

type scimUserRow struct {
	ID, WorkspaceID, UserID uuid.UUID
	ExternalID              *string
	UserName, DisplayName   string
	Active                  bool
	DirectRole              string
	RetainedManualAccess    bool
	Version                 int64
	CreatedAt, UpdatedAt    time.Time
}

func (scimUserRow) TableName() string { return "workspace_scim_users" }

type scimGroupRow struct {
	ID, WorkspaceID      uuid.UUID
	ExternalID           *string
	DisplayName, Role    string
	Version              int64
	CreatedAt, UpdatedAt time.Time
}

func (scimGroupRow) TableName() string { return "workspace_scim_groups" }

type scimGroupMemberRow struct {
	WorkspaceID, GroupID, SCIMUserID uuid.UUID
	CreatedAt                        time.Time
}

func (scimGroupMemberRow) TableName() string { return "workspace_scim_group_members" }

type membershipSourceRow struct {
	WorkspaceID, UserID uuid.UUID
	SourceKind          string
	SourceID            uuid.UUID
	Role                string
	CreatedAt           time.Time
}

func (membershipSourceRow) TableName() string { return "workspace_membership_sources" }

func scimRepoError(e error) error {
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return domain.ErrSCIMNotFound
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) && pg.Code == "23505" {
		return domain.ErrSCIMConflict
	}
	return e
}
func etag(version int64) string { return `W/"` + strconv.FormatInt(version, 10) + `"` }
func checkETag(raw string, version int64) error {
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "*" {
		return nil
	}
	if strings.TrimSpace(raw) != etag(version) {
		return domain.ErrSCIMPrecondition
	}
	return nil
}
func ptrString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
func scimUserDomain(row scimUserRow) domain.SCIMUser {
	return domain.SCIMUser{Schemas: []string{domain.SCIMUserSchema}, ID: row.ID.String(), ExternalID: valueString(row.ExternalID), UserName: row.UserName, DisplayName: row.DisplayName, Emails: []domain.SCIMEmail{{Value: row.UserName, Type: "work", Primary: true}}, Active: row.Active, Role: row.DirectRole, RetainedManualAccess: row.RetainedManualAccess, Meta: domain.SCIMMeta{ResourceType: "User", Created: row.CreatedAt, LastModified: row.UpdatedAt, Version: etag(row.Version)}}
}
func valueString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (r *SCIMRepository) ResolveCredential(ctx context.Context, hash []byte) (uuid.UUID, error) {
	var row scimCredentialRow
	e := r.db.WithContext(ctx).Where("token_hash=? AND revoked_at IS NULL", hash).First(&row).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return uuid.Nil, domain.ErrSCIMUnauthorized
	}
	return row.WorkspaceID, e
}
func (r *SCIMRepository) CredentialStatus(ctx context.Context, w uuid.UUID) (domain.SCIMTokenStatus, error) {
	var row scimCredentialRow
	e := accessDB(ctx, r.db).Where("workspace_id=? AND revoked_at IS NULL", w).First(&row).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return domain.SCIMTokenStatus{Configured: false}, nil
	}
	if e != nil {
		return domain.SCIMTokenStatus{}, e
	}
	return domain.SCIMTokenStatus{Configured: true, Prefix: row.TokenPrefix, CreatedAt: &row.CreatedAt}, nil
}
func (r *SCIMRepository) RotateCredential(ctx context.Context, c domain.SCIMCredential) error {
	return accessDB(ctx, r.db).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "workspace_id"}}, DoUpdates: clause.Assignments(map[string]any{"token_hash": c.TokenHash, "token_prefix": c.Prefix, "created_by": c.CreatedBy, "created_at": c.CreatedAt, "revoked_at": nil})}).Create(&scimCredentialRow{WorkspaceID: c.WorkspaceID, TokenHash: c.TokenHash, TokenPrefix: c.Prefix, CreatedBy: c.CreatedBy, CreatedAt: c.CreatedAt}).Error
}
func (r *SCIMRepository) RevokeCredential(ctx context.Context, w uuid.UUID, now time.Time) error {
	return accessDB(ctx, r.db).Model(&scimCredentialRow{}).Where("workspace_id=? AND revoked_at IS NULL", w).Update("revoked_at", now).Error
}

func (r *SCIMRepository) locked(ctx context.Context, w uuid.UUID, fn func(*gorm.DB) error) error {
	if current, ok := ctx.Value(accessTransactionKey{}).(accessTransaction); ok {
		if current.workspace != w {
			return domain.ErrAccessDenied
		}
		return fn(current.db.WithContext(ctx))
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var workspace domain.Workspace
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", w).First(&workspace).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return domain.ErrSCIMNotFound
			}
			return e
		}
		return fn(tx)
	})
}
func appendSCIMEvent(tx *gorm.DB, w, user uuid.UUID, kind string, now time.Time) error {
	var id int64
	key := uuid.NewString()
	e := tx.Raw(`INSERT INTO workspace_events(workspace_id,actor_kind,actor_id,kind,subject_id,occurred_at,safe_metadata,source,source_key) VALUES(?,'system','scim',?,?,?,'{}','scim',?) RETURNING id`, w, kind, user.String(), now, key).Scan(&id).Error
	if e != nil {
		return e
	}
	return tx.Exec("INSERT INTO workspace_event_outbox(event_id,workspace_id,created_at) VALUES(?,?,?)", id, w, now).Error
}
func ensureSCIMUserAccount(tx *gorm.DB, email, name string, now time.Time) (uuid.UUID, error) {
	id := uuid.New()
	res := tx.Exec(`INSERT INTO users(id,email,name,created_at,updated_at) VALUES(?,?,?,?,?) ON CONFLICT DO NOTHING`, id, email, name, now, now)
	if res.Error != nil {
		return uuid.Nil, res.Error
	}
	if res.RowsAffected == 0 {
		var user domain.User
		if e := tx.Where("lower(email)=lower(?)", email).First(&user).Error; e != nil {
			return uuid.Nil, e
		}
		id = user.ID
		if id == uuid.Nil {
			return uuid.Nil, domain.ErrSCIMConflict
		}
		return id, nil
	}
	workspace := uuid.New()
	if e := tx.Exec("INSERT INTO workspaces(id,name,owner_id,created_at,updated_at) VALUES(?,'My Workspace',?,?,?)", workspace, id, now, now).Error; e != nil {
		return uuid.Nil, e
	}
	return id, nil
}
func existingMembership(tx *gorm.DB, w, u uuid.UUID) (domain.WorkspaceRole, bool, error) {
	var member domain.WorkspaceMember
	e := tx.Where("workspace_id=? AND user_id=?", w, u).First(&member).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	return member.Role, e == nil, e
}
func hasSources(tx *gorm.DB, w, u uuid.UUID) (bool, error) {
	var n int64
	e := tx.Table("workspace_membership_sources").Where("workspace_id=? AND user_id=?", w, u).Count(&n).Error
	return n > 0, e
}
func (r *SCIMRepository) CreateUser(ctx context.Context, w uuid.UUID, in domain.SCIMUser, now time.Time) (out domain.SCIMUser, e error) {
	if in.Role != string(domain.WorkspaceRoleEditor) && in.Role != string(domain.WorkspaceRoleViewer) {
		return out, domain.ErrSCIMInvalid
	}
	e = r.locked(ctx, w, func(tx *gorm.DB) error {
		if in.ExternalID != "" {
			var existing scimUserRow
			err := tx.Where("workspace_id=? AND external_id=?", w, in.ExternalID).First(&existing).Error
			if err == nil {
				if !strings.EqualFold(existing.UserName, in.UserName) {
					return domain.ErrSCIMConflict
				}
				out = scimUserDomain(existing)
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		userID, err := ensureSCIMUserAccount(tx, in.UserName, firstNonEmpty(in.DisplayName, in.UserName), now)
		if err != nil {
			return err
		}
		role, member, err := existingMembership(tx, w, userID)
		if err != nil {
			return err
		}
		sources, err := hasSources(tx, w, userID)
		if err != nil {
			return err
		}
		manual := member && !sources
		row := scimUserRow{ID: uuid.New(), WorkspaceID: w, UserID: userID, ExternalID: ptrString(in.ExternalID), UserName: in.UserName, DisplayName: in.DisplayName, Active: in.Active, DirectRole: in.Role, RetainedManualAccess: manual, Version: 1, CreatedAt: now, UpdatedAt: now}
		if err = tx.Create(&row).Error; err != nil {
			return scimRepoError(err)
		}
		if in.Active && !manual {
			if !member {
				if err = tx.Exec("INSERT INTO workspace_members(workspace_id,user_id,role,created_at) VALUES(?,?,?,?)", w, userID, in.Role, now).Error; err != nil {
					return err
				}
			} else if role != domain.WorkspaceRoleOwner {
				if err = tx.Exec("UPDATE workspace_members SET role=? WHERE workspace_id=? AND user_id=?", in.Role, w, userID).Error; err != nil {
					return err
				}
			}
			if err = tx.Create(&membershipSourceRow{WorkspaceID: w, UserID: userID, SourceKind: domain.SCIMMembershipSourceDirect, SourceID: row.ID, Role: in.Role, CreatedAt: now}).Error; err != nil {
				return err
			}
		}
		if err = appendSCIMEvent(tx, w, userID, "scim.user.created", now); err != nil {
			return err
		}
		out = scimUserDomain(row)
		return nil
	})
	return out, e
}
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return "Provisioned user"
}
func (r *SCIMRepository) getUserDB(db *gorm.DB, w, id uuid.UUID) (scimUserRow, error) {
	var row scimUserRow
	e := db.Where("workspace_id=? AND id=?", w, id).First(&row).Error
	return row, scimRepoError(e)
}
func (r *SCIMRepository) GetUser(ctx context.Context, w, id uuid.UUID) (domain.SCIMUser, error) {
	row, e := r.getUserDB(r.db.WithContext(ctx), w, id)
	return scimUserDomain(row), e
}
func userFilterQuery(db *gorm.DB, f domain.SCIMFilter) *gorm.DB {
	switch f.Attribute {
	case "userName":
		return db.Where("lower(user_name)=lower(?)", f.Value)
	case "externalId":
		return db.Where("external_id=?", f.Value)
	case "id":
		return db.Where("id=?", f.Value)
	}
	return db
}
func (r *SCIMRepository) ListUsers(ctx context.Context, w uuid.UUID, f domain.SCIMFilter, start, count int) (domain.SCIMPage[domain.SCIMUser], error) {
	db := userFilterQuery(r.db.WithContext(ctx).Model(&scimUserRow{}).Where("workspace_id=?", w), f)
	var total int64
	if e := db.Count(&total).Error; e != nil {
		return domain.SCIMPage[domain.SCIMUser]{}, e
	}
	var rows []scimUserRow
	if e := db.Order("created_at,id").Offset(start - 1).Limit(count).Find(&rows).Error; e != nil {
		return domain.SCIMPage[domain.SCIMUser]{}, e
	}
	items := make([]domain.SCIMUser, len(rows))
	for i, row := range rows {
		items[i] = scimUserDomain(row)
	}
	return domain.SCIMPage[domain.SCIMUser]{Schemas: []string{domain.SCIMListSchema}, TotalResults: int(total), StartIndex: start, ItemsPerPage: len(items), Resources: items}, nil
}

func revokeUserSessions(tx *gorm.DB, w, u uuid.UUID, now time.Time) error {
	return tx.Exec(`UPDATE enterprise_login_sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL AND connection_id IN (SELECT id FROM workspace_sso_connections WHERE workspace_id=?)`, now, u, w).Error
}
func recomputeMembership(tx *gorm.DB, w, u uuid.UUID, manual bool, now time.Time) error {
	if manual {
		return nil
	}
	var sources []membershipSourceRow
	if e := tx.Where("workspace_id=? AND user_id=?", w, u).Find(&sources).Error; e != nil {
		return e
	}
	if len(sources) == 0 {
		return tx.Exec("DELETE FROM workspace_members WHERE workspace_id=? AND user_id=? AND role<>'owner'", w, u).Error
	}
	role := string(domain.WorkspaceRoleViewer)
	for _, source := range sources {
		if source.Role == string(domain.WorkspaceRoleEditor) {
			role = string(domain.WorkspaceRoleEditor)
			break
		}
	}
	return tx.Exec(`INSERT INTO workspace_members(workspace_id,user_id,role,created_at) VALUES(?,?,?,?) ON CONFLICT(workspace_id,user_id) DO UPDATE SET role=EXCLUDED.role WHERE workspace_members.role<>'owner'`, w, u, role, now).Error
}
func (r *SCIMRepository) replaceUser(tx *gorm.DB, w, id uuid.UUID, match string, in domain.SCIMUser, now time.Time) (domain.SCIMUser, error) {
	row, e := r.getUserDB(tx.Clauses(clause.Locking{Strength: "UPDATE"}), w, id)
	if e != nil {
		return domain.SCIMUser{}, e
	}
	if e = checkETag(match, row.Version); e != nil {
		return domain.SCIMUser{}, e
	}
	row.ExternalID = ptrString(in.ExternalID)
	row.UserName = in.UserName
	row.DisplayName = in.DisplayName
	row.Active = in.Active
	row.DirectRole = in.Role
	row.Version++
	row.UpdatedAt = now
	if e = tx.Save(&row).Error; e != nil {
		return domain.SCIMUser{}, scimRepoError(e)
	}
	if e = tx.Where("workspace_id=? AND user_id=? AND source_kind='direct'", w, row.UserID).Delete(&membershipSourceRow{}).Error; e != nil {
		return domain.SCIMUser{}, e
	}
	if in.Active && !row.RetainedManualAccess {
		if e = tx.Create(&membershipSourceRow{WorkspaceID: w, UserID: row.UserID, SourceKind: domain.SCIMMembershipSourceDirect, SourceID: row.ID, Role: in.Role, CreatedAt: now}).Error; e != nil {
			return domain.SCIMUser{}, e
		}
		if e = tx.Exec(`INSERT INTO workspace_membership_sources(workspace_id,user_id,source_kind,source_id,role,created_at)
			SELECT g.workspace_id,?, 'group',g.id,g.role,? FROM workspace_scim_groups g
			JOIN workspace_scim_group_members gm ON gm.workspace_id=g.workspace_id AND gm.group_id=g.id
			WHERE g.workspace_id=? AND gm.scim_user_id=? ON CONFLICT DO NOTHING`, row.UserID, now, w, row.ID).Error; e != nil {
			return domain.SCIMUser{}, e
		}
	}
	if !in.Active {
		if e = tx.Where("workspace_id=? AND user_id=?", w, row.UserID).Delete(&membershipSourceRow{}).Error; e != nil {
			return domain.SCIMUser{}, e
		}
	}
	if e = recomputeMembership(tx, w, row.UserID, row.RetainedManualAccess, now); e != nil {
		return domain.SCIMUser{}, e
	}
	if e = revokeUserSessions(tx, w, row.UserID, now); e != nil {
		return domain.SCIMUser{}, e
	}
	kind := "scim.user.updated"
	if !in.Active {
		kind = "scim.user.deprovisioned"
	}
	if e = appendSCIMEvent(tx, w, row.UserID, kind, now); e != nil {
		return domain.SCIMUser{}, e
	}
	return scimUserDomain(row), nil
}
func (r *SCIMRepository) ReplaceUser(ctx context.Context, w, id uuid.UUID, match string, in domain.SCIMUser, now time.Time) (out domain.SCIMUser, e error) {
	if in.Role != string(domain.WorkspaceRoleEditor) && in.Role != string(domain.WorkspaceRoleViewer) {
		return out, domain.ErrSCIMInvalid
	}
	e = r.locked(ctx, w, func(tx *gorm.DB) error {
		var err error
		out, err = r.replaceUser(tx, w, id, match, in, now)
		return err
	})
	return out, e
}
func (r *SCIMRepository) DeleteUser(ctx context.Context, w, id uuid.UUID, match string, now time.Time) (domain.SCIMUser, error) {
	current, e := r.GetUser(ctx, w, id)
	if e != nil {
		return domain.SCIMUser{}, e
	}
	current.Active = false
	return r.ReplaceUser(ctx, w, id, match, current, now)
}
func (r *SCIMRepository) AdoptUser(ctx context.Context, w, id uuid.UUID, now time.Time) (out domain.SCIMUser, e error) {
	e = r.locked(ctx, w, func(tx *gorm.DB) error {
		row, err := r.getUserDB(tx.Clauses(clause.Locking{Strength: "UPDATE"}), w, id)
		if err != nil {
			return err
		}
		if !row.RetainedManualAccess {
			out = scimUserDomain(row)
			return nil
		}
		row.RetainedManualAccess = false
		row.Version++
		row.UpdatedAt = now
		if err = tx.Save(&row).Error; err != nil {
			return err
		}
		if row.Active {
			if err = tx.Create(&membershipSourceRow{WorkspaceID: w, UserID: row.UserID, SourceKind: domain.SCIMMembershipSourceAdopted, SourceID: row.ID, Role: row.DirectRole, CreatedAt: now}).Error; err != nil {
				return err
			}
		}
		if err = recomputeMembership(tx, w, row.UserID, false, now); err != nil {
			return err
		}
		if err = appendSCIMEvent(tx, w, row.UserID, "scim.user.adopted", now); err != nil {
			return err
		}
		out = scimUserDomain(row)
		return nil
	})
	return out, e
}

func (r *SCIMRepository) groupDomain(db *gorm.DB, row scimGroupRow) (domain.SCIMGroup, error) {
	var members []struct {
		ID          uuid.UUID
		DisplayName string
	}
	e := db.Raw(`SELECT u.id,u.display_name FROM workspace_scim_group_members m JOIN workspace_scim_users u ON u.id=m.scim_user_id AND u.workspace_id=m.workspace_id WHERE m.workspace_id=? AND m.group_id=? ORDER BY u.id`, row.WorkspaceID, row.ID).Scan(&members).Error
	if e != nil {
		return domain.SCIMGroup{}, e
	}
	out := domain.SCIMGroup{Schemas: []string{domain.SCIMGroupSchema}, ID: row.ID.String(), ExternalID: valueString(row.ExternalID), DisplayName: row.DisplayName, Role: row.Role, Meta: domain.SCIMMeta{ResourceType: "Group", Created: row.CreatedAt, LastModified: row.UpdatedAt, Version: etag(row.Version)}}
	for _, m := range members {
		out.Members = append(out.Members, domain.SCIMGroupMember{Value: m.ID.String(), Display: m.DisplayName})
	}
	return out, nil
}
func groupMemberIDs(in domain.SCIMGroup) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]bool{}
	ids := make([]uuid.UUID, 0, len(in.Members))
	for _, m := range in.Members {
		id, e := uuid.Parse(m.Value)
		if e != nil || id == uuid.Nil {
			return nil, domain.ErrSCIMInvalid
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}
func (r *SCIMRepository) setGroupMembers(tx *gorm.DB, w uuid.UUID, row scimGroupRow, ids []uuid.UUID, now time.Time) error {
	var old []uuid.UUID
	if e := tx.Model(&scimGroupMemberRow{}).Where("workspace_id=? AND group_id=?", w, row.ID).Pluck("scim_user_id", &old).Error; e != nil {
		return e
	}
	affected := map[uuid.UUID]bool{}
	for _, id := range old {
		affected[id] = true
	}
	for _, id := range ids {
		affected[id] = true
	}
	if e := tx.Where("workspace_id=? AND group_id=?", w, row.ID).Delete(&scimGroupMemberRow{}).Error; e != nil {
		return e
	}
	if e := tx.Where("workspace_id=? AND source_kind='group' AND source_id=?", w, row.ID).Delete(&membershipSourceRow{}).Error; e != nil {
		return e
	}
	for _, id := range ids {
		var user scimUserRow
		if e := tx.Where("workspace_id=? AND id=? AND active=true", w, id).First(&user).Error; e != nil {
			return scimRepoError(e)
		}
		if e := tx.Create(&scimGroupMemberRow{WorkspaceID: w, GroupID: row.ID, SCIMUserID: id, CreatedAt: now}).Error; e != nil {
			return e
		}
		if !user.RetainedManualAccess {
			if e := tx.Create(&membershipSourceRow{WorkspaceID: w, UserID: user.UserID, SourceKind: domain.SCIMMembershipSourceGroup, SourceID: row.ID, Role: row.Role, CreatedAt: now}).Error; e != nil {
				return e
			}
		}
	}
	for id := range affected {
		var user scimUserRow
		if e := tx.Where("workspace_id=? AND id=?", w, id).First(&user).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				continue
			}
			return e
		}
		if e := recomputeMembership(tx, w, user.UserID, user.RetainedManualAccess, now); e != nil {
			return e
		}
		if e := revokeUserSessions(tx, w, user.UserID, now); e != nil {
			return e
		}
	}
	return nil
}
func (r *SCIMRepository) CreateGroup(ctx context.Context, w uuid.UUID, in domain.SCIMGroup, now time.Time) (out domain.SCIMGroup, e error) {
	if in.Role != string(domain.WorkspaceRoleEditor) && in.Role != string(domain.WorkspaceRoleViewer) {
		return out, domain.ErrSCIMInvalid
	}
	ids, e := groupMemberIDs(in)
	if e != nil {
		return out, e
	}
	e = r.locked(ctx, w, func(tx *gorm.DB) error {
		if in.ExternalID != "" {
			var existing scimGroupRow
			err := tx.Where("workspace_id=? AND external_id=?", w, in.ExternalID).First(&existing).Error
			if err == nil {
				if !strings.EqualFold(existing.DisplayName, in.DisplayName) {
					return domain.ErrSCIMConflict
				}
				out, err = r.groupDomain(tx, existing)
				return err
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		row := scimGroupRow{ID: uuid.New(), WorkspaceID: w, ExternalID: ptrString(in.ExternalID), DisplayName: in.DisplayName, Role: in.Role, Version: 1, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&row).Error; err != nil {
			return scimRepoError(err)
		}
		if err := r.setGroupMembers(tx, w, row, ids, now); err != nil {
			return err
		}
		var err error
		out, err = r.groupDomain(tx, row)
		return err
	})
	return out, e
}
func (r *SCIMRepository) getGroupDB(db *gorm.DB, w, id uuid.UUID) (scimGroupRow, error) {
	var row scimGroupRow
	e := db.Where("workspace_id=? AND id=?", w, id).First(&row).Error
	return row, scimRepoError(e)
}
func (r *SCIMRepository) GetGroup(ctx context.Context, w, id uuid.UUID) (domain.SCIMGroup, error) {
	row, e := r.getGroupDB(r.db.WithContext(ctx), w, id)
	if e != nil {
		return domain.SCIMGroup{}, e
	}
	return r.groupDomain(r.db.WithContext(ctx), row)
}
func groupFilterQuery(db *gorm.DB, f domain.SCIMFilter) *gorm.DB {
	switch f.Attribute {
	case "displayName":
		return db.Where("lower(display_name)=lower(?)", f.Value)
	case "externalId":
		return db.Where("external_id=?", f.Value)
	case "id":
		return db.Where("id=?", f.Value)
	}
	return db
}
func (r *SCIMRepository) ListGroups(ctx context.Context, w uuid.UUID, f domain.SCIMFilter, start, count int) (domain.SCIMPage[domain.SCIMGroup], error) {
	db := groupFilterQuery(r.db.WithContext(ctx).Model(&scimGroupRow{}).Where("workspace_id=?", w), f)
	var total int64
	if e := db.Count(&total).Error; e != nil {
		return domain.SCIMPage[domain.SCIMGroup]{}, e
	}
	var rows []scimGroupRow
	if e := db.Order("created_at,id").Offset(start - 1).Limit(count).Find(&rows).Error; e != nil {
		return domain.SCIMPage[domain.SCIMGroup]{}, e
	}
	items := make([]domain.SCIMGroup, 0, len(rows))
	for _, row := range rows {
		g, e := r.groupDomain(r.db.WithContext(ctx), row)
		if e != nil {
			return domain.SCIMPage[domain.SCIMGroup]{}, e
		}
		items = append(items, g)
	}
	return domain.SCIMPage[domain.SCIMGroup]{Schemas: []string{domain.SCIMListSchema}, TotalResults: int(total), StartIndex: start, ItemsPerPage: len(items), Resources: items}, nil
}
func (r *SCIMRepository) ReplaceGroup(ctx context.Context, w, id uuid.UUID, match string, in domain.SCIMGroup, now time.Time) (out domain.SCIMGroup, e error) {
	if in.Role != string(domain.WorkspaceRoleEditor) && in.Role != string(domain.WorkspaceRoleViewer) {
		return out, domain.ErrSCIMInvalid
	}
	ids, e := groupMemberIDs(in)
	if e != nil {
		return out, e
	}
	e = r.locked(ctx, w, func(tx *gorm.DB) error {
		row, err := r.getGroupDB(tx.Clauses(clause.Locking{Strength: "UPDATE"}), w, id)
		if err != nil {
			return err
		}
		if err = checkETag(match, row.Version); err != nil {
			return err
		}
		row.ExternalID = ptrString(in.ExternalID)
		row.DisplayName = in.DisplayName
		row.Role = in.Role
		row.Version++
		row.UpdatedAt = now
		if err = tx.Save(&row).Error; err != nil {
			return scimRepoError(err)
		}
		if err = r.setGroupMembers(tx, w, row, ids, now); err != nil {
			return err
		}
		out, err = r.groupDomain(tx, row)
		return err
	})
	return out, e
}
func (r *SCIMRepository) DeleteGroup(ctx context.Context, w, id uuid.UUID, match string, now time.Time) error {
	return r.locked(ctx, w, func(tx *gorm.DB) error {
		row, e := r.getGroupDB(tx.Clauses(clause.Locking{Strength: "UPDATE"}), w, id)
		if e != nil {
			return e
		}
		if e = checkETag(match, row.Version); e != nil {
			return e
		}
		var members []uuid.UUID
		if e = tx.Model(&scimGroupMemberRow{}).Where("workspace_id=? AND group_id=?", w, id).Pluck("scim_user_id", &members).Error; e != nil {
			return e
		}
		if e = tx.Delete(&row).Error; e != nil {
			return e
		}
		for _, sid := range members {
			var user scimUserRow
			if e = tx.Where("workspace_id=? AND id=?", w, sid).First(&user).Error; e != nil {
				return e
			}
			if e = recomputeMembership(tx, w, user.UserID, user.RetainedManualAccess, now); e != nil {
				return e
			}
			if e = revokeUserSessions(tx, w, user.UserID, now); e != nil {
				return e
			}
		}
		return appendSCIMEvent(tx, w, uuid.Nil, "scim.group.deleted", now)
	})
}

var _ domain.SCIMRepository = (*SCIMRepository)(nil)
