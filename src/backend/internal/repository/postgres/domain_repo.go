package postgres

import (
	"devtool/internal/domain"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type DomainRepository struct{ db *gorm.DB }

func NewDomainRepository(db *gorm.DB) *DomainRepository { return &DomainRepository{db: db} }

type workspaceDomainRow struct {
	ID                 uuid.UUID
	WorkspaceID        uuid.UUID
	Domain             string
	Status             domain.DomainStatus
	ChallengeHash      []byte
	ChallengeExpiresAt time.Time
	VerifiedAt         *time.Time
	CapturedAt         *time.Time
	Revision           int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (workspaceDomainRow) TableName() string { return "workspace_domains" }

type workspaceDomainRequestRow struct {
	WorkspaceID uuid.UUID
	RequestID   uuid.UUID
	Action      string
	PayloadHash string
	CreatedAt   time.Time
}

func (workspaceDomainRequestRow) TableName() string { return "workspace_domain_requests" }

func domainValue(row workspaceDomainRow) domain.WorkspaceDomain {
	return domain.WorkspaceDomain{ID: row.ID, WorkspaceID: row.WorkspaceID, Domain: row.Domain, Status: row.Status, ChallengeHash: row.ChallengeHash, ChallengeExpiresAt: row.ChallengeExpiresAt, VerifiedAt: row.VerifiedAt, CapturedAt: row.CapturedAt, Revision: row.Revision, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func domainWriteError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return domain.ErrDomainConflict
	}
	return accessError(err)
}

func (r *DomainRepository) ClaimRequest(ctx domain.Context, workspaceID, requestID uuid.UUID, action, payloadHash string) error {
	db, err := requireCollectionTransaction(ctx, workspaceID)
	if err != nil {
		return err
	}
	var existing workspaceDomainRequestRow
	lookup := db.Where("workspace_id=? AND request_id=?", workspaceID, requestID).First(&existing)
	if lookup.Error == nil {
		return domain.ErrRequestIDReused
	}
	if !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
		return lookup.Error
	}
	err = db.Create(&workspaceDomainRequestRow{WorkspaceID: workspaceID, RequestID: requestID, Action: action, PayloadHash: payloadHash, CreatedAt: time.Now().UTC()}).Error
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return domain.ErrRequestIDReused
		}
	}
	return err
}

func (r *DomainRepository) Create(ctx domain.Context, value domain.WorkspaceDomain) (domain.WorkspaceDomain, error) {
	db, err := requireCollectionTransaction(ctx, value.WorkspaceID)
	if err != nil {
		return domain.WorkspaceDomain{}, err
	}
	var existing workspaceDomainRow
	lookup := db.Where("domain=?", value.Domain).First(&existing)
	if lookup.Error == nil {
		if existing.WorkspaceID != value.WorkspaceID {
			return domain.WorkspaceDomain{}, domain.ErrDomainConflict
		}
		updates := map[string]any{"status": domain.DomainPending, "challenge_hash": value.ChallengeHash, "challenge_expires_at": value.ChallengeExpiresAt, "verified_at": nil, "captured_at": nil, "revision": gorm.Expr("revision+1"), "updated_at": value.UpdatedAt}
		result := db.Model(&workspaceDomainRow{}).Where("id=? AND workspace_id=? AND revision=?", existing.ID, existing.WorkspaceID, existing.Revision).Updates(updates)
		if result.Error != nil {
			return domain.WorkspaceDomain{}, domainWriteError(result.Error)
		}
		if result.RowsAffected != 1 {
			return domain.WorkspaceDomain{}, domain.ErrRevisionConflict
		}
		return r.Get(ctx, existing.WorkspaceID, existing.ID)
	}
	if lookup.Error != nil && !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
		return domain.WorkspaceDomain{}, lookup.Error
	}
	row := workspaceDomainRow{ID: value.ID, WorkspaceID: value.WorkspaceID, Domain: value.Domain, Status: value.Status, ChallengeHash: value.ChallengeHash, ChallengeExpiresAt: value.ChallengeExpiresAt, Revision: value.Revision, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
	if err = db.Create(&row).Error; err != nil {
		return domain.WorkspaceDomain{}, domainWriteError(err)
	}
	return domainValue(row), nil
}

func (r *DomainRepository) Get(ctx domain.Context, workspaceID, id uuid.UUID) (domain.WorkspaceDomain, error) {
	var row workspaceDomainRow
	err := accessDB(ctx, r.db).Where("workspace_id=? AND id=?", workspaceID, id).First(&row).Error
	return domainValue(row), accessError(err)
}
func (r *DomainRepository) List(ctx domain.Context, workspaceID uuid.UUID) ([]domain.WorkspaceDomain, error) {
	rows := []workspaceDomainRow{}
	if err := accessDB(ctx, r.db).Where("workspace_id=?", workspaceID).Order("created_at,id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.WorkspaceDomain, len(rows))
	for i, row := range rows {
		out[i] = domainValue(row)
	}
	return out, nil
}

func (r *DomainRepository) Discover(ctx domain.Context, normalized string) (domain.DomainDiscovery, error) {
	var result domain.DomainDiscovery
	query := `SELECT d.domain,COALESCE(w.public_id,w.id::text) AS workspace FROM workspace_domains d JOIN workspaces w ON w.id=d.workspace_id AND w.deleted_at IS NULL WHERE d.domain=? AND d.status='captured'`
	db := accessDB(ctx, r.db).Raw(query, normalized).Scan(&result)
	if db.Error != nil {
		return result, db.Error
	}
	if db.RowsAffected != 1 {
		return result, domain.ErrInaccessible
	}
	return result, nil
}

func (r *DomainRepository) Transition(ctx domain.Context, workspaceID, id uuid.UUID, revision int64, status domain.DomainStatus, now time.Time) (domain.WorkspaceDomain, error) {
	db, err := requireCollectionTransaction(ctx, workspaceID)
	if err != nil {
		return domain.WorkspaceDomain{}, err
	}
	updates := map[string]any{"status": status, "revision": gorm.Expr("revision+1"), "updated_at": now}
	if status == domain.DomainVerified {
		updates["verified_at"] = now
	}
	if status == domain.DomainCaptured {
		updates["captured_at"] = now
	}
	result := db.Model(&workspaceDomainRow{}).Where("workspace_id=? AND id=? AND revision=?", workspaceID, id, revision).Updates(updates)
	if result.Error != nil {
		return domain.WorkspaceDomain{}, domainWriteError(result.Error)
	}
	if result.RowsAffected != 1 {
		return domain.WorkspaceDomain{}, domain.ErrRevisionConflict
	}
	return r.Get(ctx, workspaceID, id)
}
