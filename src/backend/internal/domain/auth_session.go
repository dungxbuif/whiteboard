package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type SSOConnectionMode string

const (
	SSOModeOptional SSOConnectionMode = "optional"
	SSOModeEnforced SSOConnectionMode = "enforced"

	AuthMethodPassword   = "password"
	AuthMethodOIDC       = "oidc"
	AuthMethodBreakGlass = "break_glass"

	SSOTransactionLifetime = 10 * time.Minute
	SSOExchangeLifetime    = time.Minute
	RecentAuthLifetime     = 10 * time.Minute
)

var (
	ErrSSOInvalid          = errors.New("invalid sso request")
	ErrSSOReplay           = errors.New("sso request was already used")
	ErrSSOExpired          = errors.New("sso request expired")
	ErrSSONotConfigured    = errors.New("sso connection is not configured")
	ErrSSOIdentityConflict = errors.New("external identity is already linked")
	ErrSSOEmailLinkDenied  = errors.New("email cannot link an existing account")
	ErrSessionRevoked      = errors.New("login session is revoked")
	ErrWorkspaceAssurance  = errors.New("workspace requires sso assurance")
	ErrRecentAuthRequired  = errors.New("recent owner authentication is required")
	ErrBreakGlassRequired  = errors.New("tested break-glass recovery is required")
)

type SSOConnection struct {
	ID                 uuid.UUID         `json:"id"`
	WorkspaceID        uuid.UUID         `json:"workspaceId"`
	Issuer             string            `json:"issuer"`
	ClientID           string            `json:"clientId"`
	ClientSecretCipher []byte            `json:"-"`
	Mode               SSOConnectionMode `json:"mode"`
	PublicLinksEnabled bool              `json:"publicLinksEnabled"`
	Revision           int64             `json:"revision"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

type SSOConnectionInput struct {
	Issuer             string            `json:"issuer"`
	ClientID           string            `json:"clientId"`
	ClientSecret       string            `json:"clientSecret"`
	Mode               SSOConnectionMode `json:"mode"`
	PublicLinksEnabled *bool             `json:"publicLinksEnabled"`
	RequestID          uuid.UUID         `json:"requestId"`
	ExpectedRevision   int64             `json:"expectedRevision"`
}

type AuthorizationRedirect struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type OneTimeExchange struct {
	Code       string    `json:"code"`
	ReturnPath string    `json:"returnPath"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type LoginResult struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type ExternalIdentity struct {
	ID           uuid.UUID `json:"id"`
	ConnectionID uuid.UUID `json:"connectionId"`
	Issuer       string    `json:"issuer"`
	Subject      string    `json:"subject"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Picture      string    `json:"picture,omitempty"`
	UserID       uuid.UUID `json:"userId"`
	CreatedAt    time.Time `json:"createdAt"`
}

type OIDCIdentity struct {
	ConnectionID  uuid.UUID
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

type AuthSession struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"userId"`
	Method          string     `json:"method"`
	ConnectionID    *uuid.UUID `json:"connectionId,omitempty"`
	AuthenticatedAt time.Time  `json:"authenticatedAt"`
	ExpiresAt       time.Time  `json:"expiresAt"`
	RevokedAt       *time.Time `json:"revokedAt,omitempty"`
}

type SSOTransaction struct {
	StateHash          []byte
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

type ExchangeCode struct {
	CodeHash           []byte
	SessionID          uuid.UUID
	BrowserBindingHash []byte
	ReturnPath         string
	ExpiresAt          time.Time
	ConsumedAt         *time.Time
}

type SSORepository interface {
	GetConnection(Context, uuid.UUID) (SSOConnection, error)
	GetConnectionByID(Context, uuid.UUID) (SSOConnection, error)
	UpsertConnection(Context, SSOConnection) (SSOConnection, error)
	CreateTransaction(Context, SSOTransaction) error
	ConsumeTransaction(Context, []byte, []byte, time.Time) (SSOTransaction, error)
	FindIdentity(Context, string, string) (ExternalIdentity, error)
	FindIdentityForUser(Context, uuid.UUID, string, string) (ExternalIdentity, error)
	CreateIdentity(Context, ExternalIdentity) error
	CreateExchange(Context, ExchangeCode) error
	ConsumeExchange(Context, []byte, []byte, time.Time) (ExchangeCode, error)
	FindUser(Context, uuid.UUID) (*User, error)
	FindUserByEmail(Context, string) (*User, error)
	CreateUserWithDefaultWorkspace(Context, *User) error
	CreateUserIdentityWithDefaultWorkspace(Context, *User, ExternalIdentity) error
	WithMutation(Context, uuid.UUID, uuid.UUID, uuid.UUID, string, []byte, func(Context) (StoredFeatureResponse, error)) (StoredFeatureResponse, error)
}

type SessionRepository interface {
	CreateSession(Context, AuthSession) error
	GetSession(Context, uuid.UUID) (AuthSession, error)
	RevokeWorkspaceSessions(Context, uuid.UUID, uuid.UUID, *uuid.UUID, time.Time) error
	HasTestedBreakGlass(Context, uuid.UUID, uuid.UUID, time.Time) (bool, error)
}

type OIDCProvider interface {
	AuthorizationURL(state, nonce, challenge string) string
	ExchangeAndVerify(Context, string, string, string) (OIDCIdentity, error)
}

type OIDCProviderFactory interface {
	Provider(Context, SSOConnection, string) (OIDCProvider, error)
}

type SecretCipher interface {
	Seal(string) ([]byte, error)
	Open([]byte) (string, error)
}

type SessionAssurance interface {
	RequireWorkspaceAssurance(Context, Principal, uuid.UUID) error
	RequirePublicLinkAccess(Context, uuid.UUID) error
	RevokeWorkspaceSessions(Context, uuid.UUID, uuid.UUID, *uuid.UUID) error
}

type LoginSessionIssuer interface {
	IssuePasswordSession(Context, uuid.UUID) (uuid.UUID, error)
}

type SSOUsecase interface {
	Begin(Context, uuid.UUID, string) (AuthorizationRedirect, error)
	BeginLink(Context, Principal, uuid.UUID, string, uuid.UUID) (AuthorizationRedirect, error)
	Callback(Context, string, string, string) (OneTimeExchange, error)
	Exchange(Context, string, string) (LoginResult, error)
	Link(Context, uuid.UUID, OIDCIdentity, uuid.UUID) error
	Configure(Context, Principal, uuid.UUID, SSOConnectionInput) (SSOConnection, error)
	GetConnection(Context, Principal, uuid.UUID) (SSOConnection, error)
}
