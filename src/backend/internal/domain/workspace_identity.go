package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type DomainStatus string

const (
	DomainPending  DomainStatus = "pending"
	DomainVerified DomainStatus = "verified"
	DomainCaptured DomainStatus = "captured"
	DomainRevoked  DomainStatus = "revoked"
)

var (
	ErrInvalidDomain          = errors.New("invalid workspace domain")
	ErrDomainConflict         = errors.New("workspace domain is already claimed")
	ErrDomainChallengeExpired = errors.New("workspace domain challenge expired")
	ErrDomainChallengeFailed  = errors.New("workspace domain challenge was not found")
	ErrDomainNotVerified      = errors.New("workspace domain is not verified")
)

type WorkspaceDomain struct {
	ID                 uuid.UUID    `json:"id"`
	WorkspaceID        uuid.UUID    `json:"workspaceId"`
	Domain             string       `json:"domain"`
	Status             DomainStatus `json:"status"`
	ChallengeHash      []byte       `json:"-"`
	ChallengeExpiresAt time.Time    `json:"challengeExpiresAt"`
	VerifiedAt         *time.Time   `json:"verifiedAt,omitempty"`
	CapturedAt         *time.Time   `json:"capturedAt,omitempty"`
	Revision           int64        `json:"revision"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
}

type DomainChallenge struct {
	WorkspaceDomain
	DNSName  string `json:"dnsName"`
	TXTValue string `json:"txtValue"`
}

// MembershipResult records explicit owner acceptance. Domain capture changes no
// user, membership, scene, or asset ownership by itself.
type MembershipResult struct {
	Domain    WorkspaceDomain `json:"domain"`
	DataMoved bool            `json:"dataMoved"`
}

type DomainDiscovery struct {
	Domain    string `json:"domain"`
	Workspace string `json:"workspace"`
}

type DomainRepository interface {
	ClaimRequest(Context, uuid.UUID, uuid.UUID, string, string) error
	Create(Context, WorkspaceDomain) (WorkspaceDomain, error)
	Get(Context, uuid.UUID, uuid.UUID) (WorkspaceDomain, error)
	List(Context, uuid.UUID) ([]WorkspaceDomain, error)
	Discover(Context, string) (DomainDiscovery, error)
	Transition(Context, uuid.UUID, uuid.UUID, int64, DomainStatus, time.Time) (WorkspaceDomain, error)
}

type TXTResolver interface {
	LookupTXT(Context, string) ([]string, error)
}

type DomainUsecase interface {
	Begin(Context, Principal, uuid.UUID, string, uuid.UUID) (DomainChallenge, error)
	Verify(Context, Principal, uuid.UUID, uuid.UUID, uuid.UUID) (WorkspaceDomain, error)
	Capture(Context, Principal, uuid.UUID, uuid.UUID, uuid.UUID) (MembershipResult, error)
	Revoke(Context, Principal, uuid.UUID, uuid.UUID, uuid.UUID) (WorkspaceDomain, error)
	List(Context, Principal, uuid.UUID) ([]WorkspaceDomain, error)
	Discover(Context, string) (DomainDiscovery, error)
}
