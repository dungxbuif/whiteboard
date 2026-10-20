package postgres

import (
	"errors"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthSessionRepository struct{ db *gorm.DB }

func NewAuthSessionRepository(db *gorm.DB) *AuthSessionRepository {
	return &AuthSessionRepository{db: db}
}

type authSessionRow struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID          uuid.UUID
	Method          string
	ConnectionID    *uuid.UUID
	AuthenticatedAt time.Time
	ExpiresAt       time.Time
	RevokedAt       *time.Time
	CreatedAt       time.Time
}

func (authSessionRow) TableName() string { return "enterprise_login_sessions" }
func authSessionDomain(row authSessionRow) domain.AuthSession {
	return domain.AuthSession{ID: row.ID, UserID: row.UserID, Method: row.Method, ConnectionID: row.ConnectionID, AuthenticatedAt: row.AuthenticatedAt, ExpiresAt: row.ExpiresAt, RevokedAt: row.RevokedAt}
}
func (r *AuthSessionRepository) CreateSession(ctx domain.Context, s domain.AuthSession) error {
	return accessDB(ctx, r.db).Create(&authSessionRow{ID: s.ID, UserID: s.UserID, Method: s.Method, ConnectionID: s.ConnectionID, AuthenticatedAt: s.AuthenticatedAt, ExpiresAt: s.ExpiresAt, RevokedAt: s.RevokedAt}).Error
}
func (r *AuthSessionRepository) GetSession(ctx domain.Context, id uuid.UUID) (domain.AuthSession, error) {
	var row authSessionRow
	e := accessDB(ctx, r.db).Where("id=?", id).First(&row).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return domain.AuthSession{}, domain.ErrSessionRevoked
	}
	return authSessionDomain(row), e
}
func (r *AuthSessionRepository) RevokeWorkspaceSessions(ctx domain.Context, workspace, user uuid.UUID, connection *uuid.UUID, now time.Time) error {
	db := accessDB(ctx, r.db).Model(&authSessionRow{}).Where("user_id=? AND revoked_at IS NULL", user)
	if connection != nil {
		db = db.Where("connection_id=?", *connection)
	} else {
		db = db.Where("connection_id IN (SELECT id FROM workspace_sso_connections WHERE workspace_id=?)", workspace)
	}
	return db.Update("revoked_at", now).Error
}
func (r *AuthSessionRepository) HasTestedBreakGlass(ctx domain.Context, workspace, user uuid.UUID, since time.Time) (bool, error) {
	var count int64
	e := accessDB(ctx, r.db).Table("workspace_break_glass").Where("workspace_id=? AND user_id=? AND revoked_at IS NULL AND tested_at>=?", workspace, user, since).Count(&count).Error
	return count > 0, e
}
