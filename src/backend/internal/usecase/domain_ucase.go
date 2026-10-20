package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"devtool/internal/domain"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/idna"
)

const domainChallengeLifetime = 24 * time.Hour

type domainAccess interface {
	RequireOwner(domain.Context, uuid.UUID, uuid.UUID) error
	WithWorkspaceLock(domain.Context, uuid.UUID, uuid.UUID, func(domain.Context) error) error
}

type DomainService struct {
	repo     domain.DomainRepository
	access   domainAccess
	resolver domain.TXTResolver
	now      func() time.Time
	random   io.Reader
}

type DomainOption func(*DomainService)

func WithDomainClock(now func() time.Time) DomainOption {
	return func(s *DomainService) { s.now = now }
}
func WithDomainRandom(read func([]byte) (int, error)) DomainOption {
	return func(s *DomainService) { s.random = readerFunc(read) }
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

func NewDomainService(repo domain.DomainRepository, access domainAccess, resolver domain.TXTResolver, options ...DomainOption) *DomainService {
	s := &DomainService{repo: repo, access: access, resolver: resolver, now: time.Now, random: rand.Reader}
	for _, option := range options {
		option(s)
	}
	return s
}

func normalizeWorkspaceDomain(value string) (string, error) {
	value = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
	if value == "" || strings.ContainsAny(value, "/:@ \\") {
		return "", domain.ErrInvalidDomain
	}
	ascii, err := idna.Lookup.ToASCII(value)
	if err != nil {
		return "", domain.ErrInvalidDomain
	}
	ascii = strings.ToLower(ascii)
	if len(ascii) > 253 || net.ParseIP(ascii) != nil || !strings.Contains(ascii, ".") {
		return "", domain.ErrInvalidDomain
	}
	for _, label := range strings.Split(ascii, ".") {
		if len(label) == 0 || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", domain.ErrInvalidDomain
		}
	}
	return ascii, nil
}

func domainDNSName(value string) string { return "_whiteboard-verification." + value }
func domainPayloadHash(action, value string) string {
	sum := sha256.Sum256([]byte(action + "\x00" + value))
	return hex.EncodeToString(sum[:])
}
func storedDomainResponse(status int, value any) (domain.StoredDomainResponse, error) {
	body, err := json.Marshal(value)
	return domain.StoredDomainResponse{Status: status, Body: body}, err
}

func (s *DomainService) requireCurrentProof(ctx domain.Context, current domain.WorkspaceDomain) error {
	if !s.now().UTC().Before(current.ChallengeExpiresAt) {
		return domain.ErrDomainChallengeExpired
	}
	records, err := s.resolver.LookupTXT(ctx, domainDNSName(current.Domain))
	if err != nil {
		return domain.ErrDomainChallengeFailed
	}
	for _, record := range records {
		sum := sha256.Sum256([]byte(record))
		if len(current.ChallengeHash) == len(sum) && subtle.ConstantTimeCompare(current.ChallengeHash, sum[:]) == 1 {
			return nil
		}
	}
	return domain.ErrDomainChallengeFailed
}

func (s *DomainService) authorize(ctx context.Context, principal domain.Principal, workspaceID uuid.UUID) (uuid.UUID, error) {
	if principal.Kind != domain.PrincipalUser || principal.UserID == nil || workspaceID == uuid.Nil {
		return uuid.Nil, domain.ErrAccessDenied
	}
	if err := s.access.RequireOwner(ctx, *principal.UserID, workspaceID); err != nil {
		return uuid.Nil, err
	}
	return *principal.UserID, nil
}

func (s *DomainService) Begin(ctx context.Context, principal domain.Principal, workspaceID uuid.UUID, value string, requestID uuid.UUID) (domain.DomainChallenge, error) {
	owner, err := s.authorize(ctx, principal, workspaceID)
	if err != nil {
		return domain.DomainChallenge{}, err
	}
	if requestID == uuid.Nil {
		return domain.DomainChallenge{}, domain.ErrInvalidDomain
	}
	normalized, err := normalizeWorkspaceDomain(value)
	if err != nil {
		return domain.DomainChallenge{}, err
	}
	var response domain.StoredDomainResponse
	err = s.access.WithWorkspaceLock(ctx, workspaceID, owner, func(tx domain.Context) error {
		if ownerErr := s.access.RequireOwner(tx, owner, workspaceID); ownerErr != nil {
			return ownerErr
		}
		var mutationErr error
		response, mutationErr = s.repo.WithMutation(tx, principal, workspaceID, requestID, "begin", domainPayloadHash("begin", normalized), func(tx domain.Context) (domain.StoredDomainResponse, error) {
			secret := make([]byte, 32)
			if _, e := io.ReadFull(s.random, secret); e != nil {
				return domain.StoredDomainResponse{}, e
			}
			txt := "whiteboard-verification=" + base64.RawURLEncoding.EncodeToString(secret)
			digest := sha256.Sum256([]byte(txt))
			now := s.now().UTC()
			created, e := s.repo.Create(tx, domain.WorkspaceDomain{ID: uuid.New(), WorkspaceID: workspaceID, Domain: normalized, Status: domain.DomainPending, ChallengeHash: digest[:], ChallengeExpiresAt: now.Add(domainChallengeLifetime), Revision: 1, CreatedAt: now, UpdatedAt: now})
			if e != nil {
				return domain.StoredDomainResponse{}, e
			}
			return storedDomainResponse(201, domain.DomainChallenge{WorkspaceDomain: created, DNSName: domainDNSName(normalized), TXTValue: txt})
		})
		return mutationErr
	})
	if err != nil {
		return domain.DomainChallenge{}, err
	}
	var challenge domain.DomainChallenge
	if err = json.Unmarshal(response.Body, &challenge); err != nil {
		return domain.DomainChallenge{}, err
	}
	return challenge, nil
}

func (s *DomainService) Verify(ctx context.Context, principal domain.Principal, workspaceID, id, requestID uuid.UUID) (domain.WorkspaceDomain, error) {
	owner, err := s.authorize(ctx, principal, workspaceID)
	if err != nil {
		return domain.WorkspaceDomain{}, err
	}
	if id == uuid.Nil || requestID == uuid.Nil {
		return domain.WorkspaceDomain{}, domain.ErrInvalidDomain
	}
	var response domain.StoredDomainResponse
	err = s.access.WithWorkspaceLock(ctx, workspaceID, owner, func(tx domain.Context) error {
		if ownerErr := s.access.RequireOwner(tx, owner, workspaceID); ownerErr != nil {
			return ownerErr
		}
		var mutationErr error
		response, mutationErr = s.repo.WithMutation(tx, principal, workspaceID, requestID, "verify", domainPayloadHash("verify", id.String()), func(tx domain.Context) (domain.StoredDomainResponse, error) {
			current, e := s.repo.Get(tx, workspaceID, id)
			if e != nil {
				return domain.StoredDomainResponse{}, e
			}
			if current.Status != domain.DomainPending {
				return domain.StoredDomainResponse{}, domain.ErrDomainChallengeExpired
			}
			if e = s.requireCurrentProof(tx, current); e != nil {
				return domain.StoredDomainResponse{}, e
			}
			verified, e := s.repo.Transition(tx, workspaceID, id, current.Revision, domain.DomainVerified, s.now().UTC())
			if e != nil {
				return domain.StoredDomainResponse{}, e
			}
			return storedDomainResponse(200, verified)
		})
		return mutationErr
	})
	if err != nil {
		return domain.WorkspaceDomain{}, err
	}
	var verified domain.WorkspaceDomain
	err = json.Unmarshal(response.Body, &verified)
	return verified, err
}

func (s *DomainService) Capture(ctx context.Context, principal domain.Principal, workspaceID, id, requestID uuid.UUID) (domain.MembershipResult, error) {
	owner, err := s.authorize(ctx, principal, workspaceID)
	if err != nil {
		return domain.MembershipResult{}, err
	}
	if id == uuid.Nil || requestID == uuid.Nil {
		return domain.MembershipResult{}, domain.ErrInvalidDomain
	}
	var response domain.StoredDomainResponse
	err = s.access.WithWorkspaceLock(ctx, workspaceID, owner, func(tx domain.Context) error {
		if ownerErr := s.access.RequireOwner(tx, owner, workspaceID); ownerErr != nil {
			return ownerErr
		}
		var mutationErr error
		response, mutationErr = s.repo.WithMutation(tx, principal, workspaceID, requestID, "capture", domainPayloadHash("capture", id.String()), func(tx domain.Context) (domain.StoredDomainResponse, error) {
			current, e := s.repo.Get(tx, workspaceID, id)
			if e != nil {
				return domain.StoredDomainResponse{}, e
			}
			if current.Status != domain.DomainVerified {
				return domain.StoredDomainResponse{}, domain.ErrDomainNotVerified
			}
			if e = s.requireCurrentProof(tx, current); e != nil {
				return domain.StoredDomainResponse{}, e
			}
			captured, e := s.repo.Transition(tx, workspaceID, id, current.Revision, domain.DomainCaptured, s.now().UTC())
			if e != nil {
				return domain.StoredDomainResponse{}, e
			}
			return storedDomainResponse(200, domain.MembershipResult{Domain: captured, DataMoved: false})
		})
		return mutationErr
	})
	if err != nil {
		return domain.MembershipResult{}, err
	}
	var captured domain.MembershipResult
	err = json.Unmarshal(response.Body, &captured)
	return captured, err
}

func (s *DomainService) Revoke(ctx context.Context, principal domain.Principal, workspaceID, id, requestID uuid.UUID) (domain.WorkspaceDomain, error) {
	owner, err := s.authorize(ctx, principal, workspaceID)
	if err != nil {
		return domain.WorkspaceDomain{}, err
	}
	if id == uuid.Nil || requestID == uuid.Nil {
		return domain.WorkspaceDomain{}, domain.ErrInvalidDomain
	}
	var response domain.StoredDomainResponse
	err = s.access.WithWorkspaceLock(ctx, workspaceID, owner, func(tx domain.Context) error {
		if ownerErr := s.access.RequireOwner(tx, owner, workspaceID); ownerErr != nil {
			return ownerErr
		}
		var mutationErr error
		response, mutationErr = s.repo.WithMutation(tx, principal, workspaceID, requestID, "revoke", domainPayloadHash("revoke", id.String()), func(tx domain.Context) (domain.StoredDomainResponse, error) {
			current, e := s.repo.Get(tx, workspaceID, id)
			if e != nil {
				return domain.StoredDomainResponse{}, e
			}
			if current.Status == domain.DomainPending {
				return domain.StoredDomainResponse{}, domain.ErrDomainNotVerified
			}
			if current.Status == domain.DomainRevoked {
				return domain.StoredDomainResponse{}, domain.ErrRevisionConflict
			}
			revoked, e := s.repo.Transition(tx, workspaceID, id, current.Revision, domain.DomainRevoked, s.now().UTC())
			if e != nil {
				return domain.StoredDomainResponse{}, e
			}
			return storedDomainResponse(200, revoked)
		})
		return mutationErr
	})
	if err != nil {
		return domain.WorkspaceDomain{}, err
	}
	var revoked domain.WorkspaceDomain
	err = json.Unmarshal(response.Body, &revoked)
	return revoked, err
}

func (s *DomainService) List(ctx context.Context, principal domain.Principal, workspaceID uuid.UUID) ([]domain.WorkspaceDomain, error) {
	if _, err := s.authorize(ctx, principal, workspaceID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, workspaceID)
}

func (s *DomainService) Discover(ctx context.Context, value string) (domain.DomainDiscovery, error) {
	normalized, err := normalizeWorkspaceDomain(value)
	if err != nil {
		return domain.DomainDiscovery{}, err
	}
	return s.repo.Discover(ctx, normalized)
}
