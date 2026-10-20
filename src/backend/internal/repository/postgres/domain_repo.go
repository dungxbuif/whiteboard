package postgres

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DomainRepository struct {
	db        *gorm.DB
	aead      cipher.AEAD
	cipherErr error
}

func NewDomainRepository(db *gorm.DB, key ...[]byte) *DomainRepository {
	material := []byte(os.Getenv("DOMAIN_RECEIPT_KEY"))
	if len(material) == 0 {
		material = []byte(os.Getenv("JWT_SECRET"))
	}
	if len(key) > 0 {
		material = key[0]
	}
	r := &DomainRepository{db: db}
	if len(material) == 0 {
		r.cipherErr = errors.New("domain receipt encryption key unavailable")
		return r
	}
	sum := sha256.Sum256(append([]byte("workspace-domain-receipts\x00"), material...))
	block, err := aes.NewCipher(sum[:])
	if err == nil {
		r.aead, err = cipher.NewGCM(block)
	}
	r.cipherErr = err
	return r
}

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
	WorkspaceID        uuid.UUID
	RequestID          uuid.UUID
	Action             string
	PrincipalKind      string
	PrincipalID        string
	PayloadHash        string
	ResponseStatus     int
	ResponseNonce      []byte
	ResponseCiphertext []byte
	CreatedAt          time.Time
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

func domainReceiptAAD(workspaceID, requestID uuid.UUID, principal domain.Principal, action, payloadHash string) []byte {
	return []byte(fmt.Sprintf("%s|%s|%s|%s|%s|%s", workspaceID, requestID, principal.Kind, principal.ID, action, payloadHash))
}

func (r *DomainRepository) WithMutation(ctx domain.Context, principal domain.Principal, workspaceID, requestID uuid.UUID, action, payloadHash string, fn func(domain.Context) (domain.StoredDomainResponse, error)) (domain.StoredDomainResponse, error) {
	db, err := requireCollectionTransaction(ctx, workspaceID)
	if err != nil {
		return domain.StoredDomainResponse{}, err
	}
	if r.cipherErr != nil {
		return domain.StoredDomainResponse{}, r.cipherErr
	}
	var existing workspaceDomainRequestRow
	lookup := db.Where("workspace_id=? AND request_id=?", workspaceID, requestID).First(&existing)
	if lookup.Error == nil {
		if existing.Action != action || existing.PrincipalKind != string(principal.Kind) || existing.PrincipalID != principal.ID || existing.PayloadHash != payloadHash {
			return domain.StoredDomainResponse{}, domain.ErrRequestIDReused
		}
		body, openErr := r.aead.Open(nil, existing.ResponseNonce, existing.ResponseCiphertext, domainReceiptAAD(workspaceID, requestID, principal, action, payloadHash))
		if openErr != nil || !json.Valid(body) {
			return domain.StoredDomainResponse{}, errors.New("invalid domain receipt")
		}
		return domain.StoredDomainResponse{Status: existing.ResponseStatus, Body: body}, nil
	}
	if !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
		return domain.StoredDomainResponse{}, lookup.Error
	}
	response, err := fn(ctx)
	if err != nil {
		return domain.StoredDomainResponse{}, err
	}
	if response.Status < 100 || response.Status > 599 || !json.Valid(response.Body) {
		return domain.StoredDomainResponse{}, errors.New("invalid domain response")
	}
	nonce := make([]byte, r.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return domain.StoredDomainResponse{}, err
	}
	ciphertext := r.aead.Seal(nil, nonce, response.Body, domainReceiptAAD(workspaceID, requestID, principal, action, payloadHash))
	row := workspaceDomainRequestRow{WorkspaceID: workspaceID, RequestID: requestID, Action: action, PrincipalKind: string(principal.Kind), PrincipalID: principal.ID, PayloadHash: payloadHash, ResponseStatus: response.Status, ResponseNonce: nonce, ResponseCiphertext: ciphertext, CreatedAt: time.Now().UTC()}
	if err = db.Create(&row).Error; err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return domain.StoredDomainResponse{}, domain.ErrRequestIDReused
		}
		return domain.StoredDomainResponse{}, err
	}
	return response, nil
}

func (r *DomainRepository) Create(ctx domain.Context, value domain.WorkspaceDomain) (domain.WorkspaceDomain, error) {
	db, err := requireCollectionTransaction(ctx, value.WorkspaceID)
	if err != nil {
		return domain.WorkspaceDomain{}, err
	}
	var existing workspaceDomainRow
	lookup := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("domain=?", value.Domain).First(&existing)
	if lookup.Error == nil {
		if existing.WorkspaceID != value.WorkspaceID {
			if existing.Status != domain.DomainPending {
				return domain.WorkspaceDomain{}, domain.ErrDomainConflict
			}
			updates := map[string]any{"id": value.ID, "workspace_id": value.WorkspaceID, "status": domain.DomainPending, "challenge_hash": value.ChallengeHash, "challenge_expires_at": value.ChallengeExpiresAt, "verified_at": nil, "captured_at": nil, "revision": 1, "created_at": value.CreatedAt, "updated_at": value.UpdatedAt}
			result := db.Model(&workspaceDomainRow{}).Where("id=? AND workspace_id=? AND revision=? AND status=? AND challenge_expires_at<=clock_timestamp()", existing.ID, existing.WorkspaceID, existing.Revision, domain.DomainPending).Updates(updates)
			if result.Error != nil {
				return domain.WorkspaceDomain{}, domainWriteError(result.Error)
			}
			if result.RowsAffected != 1 {
				return domain.WorkspaceDomain{}, domain.ErrDomainConflict
			}
			return r.Get(ctx, value.WorkspaceID, value.ID)
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
