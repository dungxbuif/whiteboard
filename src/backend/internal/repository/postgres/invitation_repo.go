package postgres

import (
	"errors"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const postgresUniqueViolation = "23505"

type InvitationRepository struct {
	db *gorm.DB
	*UserProvisioningRepository
}

func NewInvitationRepository(db *gorm.DB) *InvitationRepository {
	return &InvitationRepository{db: db, UserProvisioningRepository: NewUserProvisioningRepository(db)}
}
func (r *InvitationRepository) Create(ctx domain.Context, i *domain.Invitation) error {
	db := accessDB(ctx, r.db)
	if err := db.Create(i).Error; err != nil {
		return err
	}
	return db.Exec("SELECT collab_bump_acl(?,NULL,NULL,'invitation.created')", i.WorkspaceID).Error
}
func (r *InvitationRepository) ListPending(ctx domain.Context, w uuid.UUID, now time.Time) ([]domain.Invitation, error) {
	rows := []domain.Invitation{}
	err := accessDB(ctx, r.db).Omit("token_hash").Where("workspace_id=? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at>?", w, now).Order("created_at,id").Find(&rows).Error
	return rows, err
}
func (r *InvitationRepository) Revoke(ctx domain.Context, w, id uuid.UUID, now time.Time) error {
	db := accessDB(ctx, r.db)
	result := db.Model(&domain.Invitation{}).Where("workspace_id=? AND id=? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at>?", w, id, now).Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrInvitationInvalid
	}
	return db.Exec("SELECT collab_bump_acl(?,NULL,NULL,'invitation.revoked')", w).Error
}
func (r *InvitationRepository) FindByHash(ctx domain.Context, hash string, lock bool) (*domain.Invitation, error) {
	var row domain.Invitation
	db := accessDB(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := db.First(&row, "token_hash=?", hash).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = domain.ErrInvitationInvalid
	}
	return &row, err
}
func (r *InvitationRepository) FindUser(ctx domain.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := accessDB(ctx, r.db).First(&u, "id=?", id).Error
	return &u, err
}
func (r *InvitationRepository) EmailExists(ctx domain.Context, email string) (bool, error) {
	var count int64
	err := accessDB(ctx, r.db).Unscoped().Model(&domain.User{}).Where("lower(email)=?", email).Count(&count).Error
	return count > 0, err
}
func (r *InvitationRepository) CreateUser(ctx domain.Context, user *domain.User) error {
	err := accessDB(ctx, r.db).Create(user).Error
	var conflict *pgconn.PgError
	if errors.As(err, &conflict) && conflict.Code == postgresUniqueViolation {
		return domain.ErrRegistrationRejected
	}
	return err
}
func (r *InvitationRepository) Join(ctx domain.Context, i *domain.Invitation, user uuid.UUID, now time.Time) (domain.WorkspaceMember, error) {
	db := accessDB(ctx, r.db)
	m := domain.WorkspaceMember{WorkspaceID: i.WorkspaceID, UserID: user, Role: i.Role, CreatedAt: now}
	// Owner/editor membership cannot be reduced by a viewer invite.
	err := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "workspace_id"}, {Name: "user_id"}}, DoUpdates: clause.Assignments(map[string]interface{}{"role": gorm.Expr("CASE WHEN workspace_members.role='owner' THEN 'owner' WHEN workspace_members.role='editor' OR EXCLUDED.role='editor' THEN 'editor' ELSE 'viewer' END")})}).Create(&m).Error
	if err != nil {
		return m, err
	}
	result := db.Model(&domain.Invitation{}).Where("id=? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at>?", i.ID, now).Update("accepted_at", now)
	if result.Error != nil {
		return m, result.Error
	}
	if result.RowsAffected != 1 {
		return m, domain.ErrInvitationInvalid
	}
	var managed scimUserRow
	available, availabilityErr := scimProvenanceAvailable(db)
	if availabilityErr != nil {
		return m, availabilityErr
	}
	managedErr := gorm.ErrRecordNotFound
	if available {
		managedErr = db.Where("workspace_id=? AND user_id=?", i.WorkspaceID, user).First(&managed).Error
	}
	if managedErr == nil {
		source := membershipSourceRow{WorkspaceID: i.WorkspaceID, UserID: user, SourceKind: domain.SCIMMembershipSourceManual, SourceID: i.ID, Role: string(i.Role), CreatedAt: now}
		if err = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&source).Error; err != nil {
			return m, err
		}
		if err = db.Model(&scimUserRow{}).Where("workspace_id=? AND user_id=?", i.WorkspaceID, user).Update("retained_manual_access", true).Error; err != nil {
			return m, err
		}
		if err = recomputeMembership(db, i.WorkspaceID, user, false, now); err != nil {
			return m, err
		}
	} else if !errors.Is(managedErr, gorm.ErrRecordNotFound) {
		return m, managedErr
	}
	err = db.First(&m, "workspace_id=? AND user_id=?", i.WorkspaceID, user).Error
	return m, err
}
