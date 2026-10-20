package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SSORepository struct{ db *gorm.DB }

func NewSSORepository(db *gorm.DB) *SSORepository { return &SSORepository{db: db} }

type ssoConnectionRow struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID        uuid.UUID `gorm:"type:uuid"`
	Issuer             string
	ClientID           string
	ClientSecretCipher []byte
	Mode               string
	PublicLinksEnabled bool
	Revision           int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (ssoConnectionRow) TableName() string { return "workspace_sso_connections" }

func connectionDomain(r ssoConnectionRow) domain.SSOConnection {
	return domain.SSOConnection{ID: r.ID, WorkspaceID: r.WorkspaceID, Issuer: r.Issuer, ClientID: r.ClientID, ClientSecretCipher: append([]byte(nil), r.ClientSecretCipher...), Mode: domain.SSOConnectionMode(r.Mode), PublicLinksEnabled: r.PublicLinksEnabled, Revision: r.Revision, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (r *SSORepository) GetConnection(ctx domain.Context, workspaceID uuid.UUID) (domain.SSOConnection, error) {
	var row ssoConnectionRow
	err := accessDB(ctx, r.db).Where("workspace_id=?", workspaceID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.SSOConnection{}, domain.ErrSSONotConfigured
	}
	return connectionDomain(row), err
}

func (r *SSORepository) GetConnectionByID(ctx domain.Context, id uuid.UUID) (domain.SSOConnection, error) {
	var row ssoConnectionRow
	err := accessDB(ctx, r.db).Where("id=?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.SSOConnection{}, domain.ErrSSONotConfigured
	}
	return connectionDomain(row), err
}

func (r *SSORepository) UpsertConnection(ctx domain.Context, c domain.SSOConnection) (domain.SSOConnection, error) {
	db := accessDB(ctx, r.db)
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
		c.Revision = 1
		row := ssoConnectionRow{ID: c.ID, WorkspaceID: c.WorkspaceID, Issuer: c.Issuer, ClientID: c.ClientID, ClientSecretCipher: c.ClientSecretCipher, Mode: string(c.Mode), PublicLinksEnabled: c.PublicLinksEnabled, Revision: c.Revision}
		if err := db.Create(&row).Error; err != nil {
			return domain.SSOConnection{}, accessError(err)
		}
		return connectionDomain(row), nil
	}
	result := db.Model(&ssoConnectionRow{}).Where("id=? AND workspace_id=? AND revision=?", c.ID, c.WorkspaceID, c.Revision).Updates(map[string]any{"issuer": c.Issuer, "client_id": c.ClientID, "client_secret_cipher": c.ClientSecretCipher, "mode": c.Mode, "public_links_enabled": c.PublicLinksEnabled, "revision": gorm.Expr("revision+1"), "updated_at": gorm.Expr("clock_timestamp()")})
	if result.Error != nil {
		return domain.SSOConnection{}, accessError(result.Error)
	}
	if result.RowsAffected != 1 {
		return domain.SSOConnection{}, domain.ErrRevisionConflict
	}
	return r.GetConnection(ctx, c.WorkspaceID)
}

type ssoTransactionRow struct {
	StateHash          []byte `gorm:"primaryKey"`
	ConnectionID       uuid.UUID
	WorkspaceID        uuid.UUID
	BrowserBindingHash []byte
	NonceHash          []byte
	PKCEVerifierCipher []byte
	ReturnPath         string
	LinkUserID         *uuid.UUID
	LinkRequestID      *uuid.UUID
	ExpiresAt          time.Time
	ConsumedAt         *time.Time
}

func (ssoTransactionRow) TableName() string { return "sso_flow_transactions" }

func (r *SSORepository) CreateTransaction(ctx domain.Context, value domain.SSOTransaction) error {
	return accessDB(ctx, r.db).Create(&ssoTransactionRow{StateHash: value.StateHash, ConnectionID: value.ConnectionID, WorkspaceID: value.WorkspaceID, BrowserBindingHash: value.BrowserBindingHash, NonceHash: value.NonceHash, PKCEVerifierCipher: value.PKCEVerifierCipher, ReturnPath: value.ReturnPath, LinkUserID: value.LinkUserID, LinkRequestID: value.LinkRequestID, ExpiresAt: value.ExpiresAt}).Error
}

func (r *SSORepository) ConsumeTransaction(ctx domain.Context, stateHash, browserHash []byte, now time.Time) (domain.SSOTransaction, error) {
	var out domain.SSOTransaction
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row ssoTransactionRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("state_hash=?", stateHash).First(&row).Error; err != nil {
			return domain.ErrSSOInvalid
		}
		if row.ConsumedAt != nil {
			return domain.ErrSSOReplay
		}
		if !now.Before(row.ExpiresAt) {
			return domain.ErrSSOExpired
		}
		if !bytes.Equal(row.BrowserBindingHash, browserHash) {
			return domain.ErrSSOInvalid
		}
		if err := tx.Model(&row).Update("consumed_at", now).Error; err != nil {
			return err
		}
		out = domain.SSOTransaction{StateHash: row.StateHash, ConnectionID: row.ConnectionID, WorkspaceID: row.WorkspaceID, BrowserBindingHash: row.BrowserBindingHash, NonceHash: row.NonceHash, PKCEVerifierCipher: row.PKCEVerifierCipher, ReturnPath: row.ReturnPath, LinkUserID: row.LinkUserID, LinkRequestID: row.LinkRequestID, ExpiresAt: row.ExpiresAt, ConsumedAt: &now}
		return nil
	})
	return out, err
}

type externalIdentityRow struct {
	ID           uuid.UUID
	ConnectionID uuid.UUID
	Issuer       string
	Subject      string
	Email        string
	UserID       uuid.UUID
	CreatedAt    time.Time
}

func (externalIdentityRow) TableName() string { return "sso_external_identities" }
func identityDomain(r externalIdentityRow) domain.ExternalIdentity {
	return domain.ExternalIdentity{ID: r.ID, ConnectionID: r.ConnectionID, Issuer: r.Issuer, Subject: r.Subject, Email: r.Email, UserID: r.UserID, CreatedAt: r.CreatedAt}
}
func (r *SSORepository) FindIdentity(ctx domain.Context, issuer, subject string) (domain.ExternalIdentity, error) {
	var row externalIdentityRow
	e := accessDB(ctx, r.db).Where("issuer=? AND subject=?", issuer, subject).First(&row).Error
	return identityDomain(row), e
}
func (r *SSORepository) FindIdentityForUser(ctx domain.Context, user uuid.UUID, issuer, subject string) (domain.ExternalIdentity, error) {
	var row externalIdentityRow
	e := accessDB(ctx, r.db).Where("user_id=? AND issuer=? AND subject=?", user, issuer, subject).First(&row).Error
	return identityDomain(row), e
}
func (r *SSORepository) CreateIdentity(ctx domain.Context, value domain.ExternalIdentity) error {
	if value.ID == uuid.Nil {
		value.ID = uuid.New()
	}
	return accessDB(ctx, r.db).Create(&externalIdentityRow{ID: value.ID, ConnectionID: value.ConnectionID, Issuer: value.Issuer, Subject: value.Subject, Email: value.Email, UserID: value.UserID}).Error
}

type exchangeCodeRow struct {
	CodeHash           []byte `gorm:"primaryKey"`
	SessionID          uuid.UUID
	BrowserBindingHash []byte
	ReturnPath         string
	ExpiresAt          time.Time
	ConsumedAt         *time.Time
}

func (exchangeCodeRow) TableName() string { return "sso_exchange_codes" }
func (r *SSORepository) CreateExchange(ctx domain.Context, value domain.ExchangeCode) error {
	return accessDB(ctx, r.db).Create(&exchangeCodeRow{CodeHash: value.CodeHash, SessionID: value.SessionID, BrowserBindingHash: value.BrowserBindingHash, ReturnPath: value.ReturnPath, ExpiresAt: value.ExpiresAt}).Error
}
func (r *SSORepository) ConsumeExchange(ctx domain.Context, codeHash, browserHash []byte, now time.Time) (domain.ExchangeCode, error) {
	var out domain.ExchangeCode
	e := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row exchangeCodeRow
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code_hash=?", codeHash).First(&row).Error != nil {
			return domain.ErrSSOInvalid
		}
		if row.ConsumedAt != nil {
			return domain.ErrSSOReplay
		}
		if !now.Before(row.ExpiresAt) {
			return domain.ErrSSOExpired
		}
		if !bytes.Equal(row.BrowserBindingHash, browserHash) {
			return domain.ErrSSOInvalid
		}
		if e := tx.Model(&row).Update("consumed_at", now).Error; e != nil {
			return e
		}
		out = domain.ExchangeCode{CodeHash: row.CodeHash, SessionID: row.SessionID, BrowserBindingHash: row.BrowserBindingHash, ReturnPath: row.ReturnPath, ExpiresAt: row.ExpiresAt, ConsumedAt: &now}
		return nil
	})
	return out, e
}

func (r *SSORepository) FindUser(ctx domain.Context, id uuid.UUID) (*domain.User, error) {
	return NewUserRepository(r.db).FindByID(ctx, id)
}
func (r *SSORepository) FindUserByEmail(ctx domain.Context, email string) (*domain.User, error) {
	return NewUserRepository(r.db).FindByEmail(ctx, email)
}
func (r *SSORepository) CreateUserWithDefaultWorkspace(ctx domain.Context, u *domain.User) error {
	return r.CreateUserIdentityWithDefaultWorkspace(ctx, u, domain.ExternalIdentity{})
}
func (r *SSORepository) CreateUserIdentityWithDefaultWorkspace(ctx domain.Context, u *domain.User, identity domain.ExternalIdentity) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		alias, err := domain.NewPublicID()
		if err != nil {
			return err
		}
		workspace := domain.Workspace{ID: uuid.New(), PublicID: &alias, Name: "My Workspace", OwnerID: u.ID}
		if err = tx.Create(&workspace).Error; err != nil {
			return fmt.Errorf("create default workspace: %w", err)
		}
		if identity.ConnectionID != uuid.Nil {
			if identity.ID == uuid.Nil {
				identity.ID = uuid.New()
			}
			identity.UserID = u.ID
			if err = tx.Create(&externalIdentityRow{ID: identity.ID, ConnectionID: identity.ConnectionID, Issuer: identity.Issuer, Subject: identity.Subject, Email: identity.Email, UserID: u.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type ssoMutationReceiptRow struct {
	WorkspaceID  uuid.UUID `gorm:"primaryKey"`
	RequestID    uuid.UUID `gorm:"primaryKey"`
	PrincipalID  uuid.UUID
	Action       string
	PayloadHash  string
	ResponseBody []byte `gorm:"type:bytea"`
	CreatedAt    time.Time
}

func (ssoMutationReceiptRow) TableName() string { return "sso_mutation_receipts" }

func (r *SSORepository) WithMutation(ctx domain.Context, workspace, requestID, principal uuid.UUID, action string, payload []byte, fn func(domain.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	if requestID == uuid.Nil || principal == uuid.Nil || action == "" || !json.Valid(payload) {
		return domain.StoredFeatureResponse{}, domain.ErrSSOInvalid
	}
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])
	db := accessDB(ctx, r.db)
	var existing ssoMutationReceiptRow
	e := db.Where("workspace_id=? AND request_id=?", workspace, requestID).First(&existing).Error
	if e == nil {
		if existing.PrincipalID != principal || existing.Action != action || existing.PayloadHash != hash {
			return domain.StoredFeatureResponse{}, domain.ErrRequestIDReused
		}
		return domain.StoredFeatureResponse{Status: 200, Body: existing.ResponseBody}, nil
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return domain.StoredFeatureResponse{}, e
	}
	response, e := fn(ctx)
	if e != nil {
		return domain.StoredFeatureResponse{}, e
	}
	if !json.Valid(response.Body) {
		return domain.StoredFeatureResponse{}, domain.ErrSSOInvalid
	}
	row := ssoMutationReceiptRow{WorkspaceID: workspace, RequestID: requestID, PrincipalID: principal, Action: action, PayloadHash: hash, ResponseBody: response.Body}
	if e = db.Create(&row).Error; e != nil {
		return domain.StoredFeatureResponse{}, e
	}
	return response, nil
}
