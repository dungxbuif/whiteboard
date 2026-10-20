package usecase

import (
	"devtool/internal/domain"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"strconv"
	"time"
	"unicode"
)

type CapabilityService struct {
	access domain.AccessService
	links  domain.ShareLinkRepository
	assets domain.PrincipalAssetUploader
	secret []byte
}

func NewCapabilityService(a domain.AccessService, l domain.ShareLinkRepository, u domain.PrincipalAssetUploader, key []byte) (*CapabilityService, error) {
	if a == nil || len(key) < 32 {
		return nil, errors.New("mutation authority requires access service and MUTATION_GRANT_SECRET of at least 32 bytes")
	}
	return &CapabilityService{a, l, u, append([]byte(nil), key...)}, nil
}
func (s *CapabilityService) expiry(ctx domain.Context, p domain.Principal, ttl time.Duration) (time.Time, error) {
	now := time.Now()
	expires := now.Add(ttl)
	if p.Kind != domain.PrincipalUser && p.Kind != domain.PrincipalPublic && p.Kind != domain.PrincipalAgent {
		return time.Time{}, domain.ErrAccessDenied
	}
	if p.CredentialExpiresAt.IsZero() || !p.CredentialExpiresAt.After(now) {
		return time.Time{}, domain.ErrInvalidSceneSession
	}
	if p.CredentialExpiresAt.Before(expires) {
		expires = p.CredentialExpiresAt
	}
	if p.Kind == domain.PrincipalPublic {
		if s.links == nil {
			return time.Time{}, domain.ErrAccessDenied
		}
		id, e := uuid.Parse(p.GrantID)
		if e != nil {
			return time.Time{}, domain.ErrAccessDenied
		}
		link, e := s.links.FindByID(ctx, id)
		if e != nil {
			return time.Time{}, e
		}
		if !link.Active || strconv.FormatInt(link.Revision, 10) != p.GrantVersion {
			return time.Time{}, domain.ErrAccessDenied
		}
		if link.ExpiresAt != nil && link.ExpiresAt.Before(expires) {
			expires = *link.ExpiresAt
		}
	}
	expires = expires.Truncate(time.Second)
	if !expires.After(now) {
		return time.Time{}, domain.ErrInvalidSceneSession
	}
	return expires, nil
}
func validateMutation(m domain.SceneMutation) error {
	if validator.New().Struct(m) != nil {
		return domain.ErrInvalidCapability
	}
	w, _ := uuid.Parse(m.WorkspaceID)
	id, _ := uuid.Parse(m.SceneID)
	if w == uuid.Nil || id == uuid.Nil {
		return domain.ErrInvalidCapability
	}
	for _, c := range m.RequestID {
		if !unicode.IsPrint(c) {
			return domain.ErrInvalidCapability
		}
	}
	if m.ExpectedSequence != "" {
		n, e := strconv.ParseInt(m.ExpectedSequence, 10, 64)
		if e != nil || n < 0 || strconv.FormatInt(n, 10) != m.ExpectedSequence {
			return domain.ErrInvalidCapability
		}
	}
	if (m.Action == domain.ActionSceneRead || m.Action == domain.ActionSceneExport) && m.PayloadHash != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		return domain.ErrInvalidCapability
	}
	return nil
}
func (s *CapabilityService) AuthorizeCommand(ctx domain.Context, p domain.Principal, m domain.SceneMutation) (domain.SignedGrant, error) {
	var result domain.SignedGrant
	if e := validateMutation(m); e != nil {
		return result, e
	}
	w, _ := uuid.Parse(m.WorkspaceID)
	id, _ := uuid.Parse(m.SceneID)
	actor := uuid.Nil
	if p.UserID != nil {
		actor = *p.UserID
	}
	e := s.access.WithWorkspaceLock(ctx, w, actor, func(ctx domain.Context) error {
		expires, e := s.expiry(ctx, p, domain.MutationLifetime)
		if e != nil {
			return e
		}
		if p.Kind == domain.PrincipalAgent {
			scoped, ok := s.access.(domain.AgentScopeRepository)
			if !ok {
				return domain.ErrAccessDenied
			}
			scope := domain.ScopeSceneWrite
			if m.Action == domain.ActionSceneRead {
				scope = domain.ScopeSceneRead
			}
			if m.Action == domain.ActionSceneExport {
				scope = domain.ScopeSceneExport
			}
			if e := scoped.RequireAgentScope(ctx, p, domain.SceneRef{WorkspaceID: w, SceneID: id}, scope); e != nil {
				return e
			}
		}
		d, e := s.access.Resolve(ctx, p, domain.SceneRef{WorkspaceID: w, SceneID: id})
		if e != nil {
			return e
		}
		if d.ACLVersion < 1 || ((m.Action != domain.ActionSceneRead && m.Action != domain.ActionSceneExport) && d.Access != domain.AccessEdit) {
			return domain.ErrAccessDenied
		}
		c := domain.MutationClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: domain.RoomIssuer, Subject: p.ID, Audience: jwt.ClaimStrings{domain.MutationAudience}, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(expires), ID: uuid.NewString()}, Purpose: domain.MutationPurpose, SceneMutation: m, PrincipalType: p.Kind, PrincipalID: p.ID, GrantID: p.GrantID, GrantVersion: p.GrantVersion, Access: d.Access, ACLVersion: strconv.FormatInt(d.ACLVersion, 10), AuthMethod: p.AuthMethod}
		if p.SessionID != uuid.Nil {
			c.SessionID = p.SessionID.String()
		}
		token, e := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
		if e != nil {
			return e
		}
		result = domain.SignedGrant{Token: token, ExpiresAt: expires}
		return nil
	})
	if e != nil {
		return domain.SignedGrant{}, e
	}
	return result, nil
}
func (s *CapabilityService) CreateAssetCapability(ctx domain.Context, p domain.Principal, ref domain.SceneRef, requests []domain.AssetUploadRequest) (domain.AssetGrant, error) {
	if p.Kind == domain.PrincipalAgent {
		return domain.AssetGrant{}, domain.ErrAccessDenied
	}
	var result domain.AssetGrant
	if e := validateAssets(requests); e != nil {
		return result, e
	}
	if s.assets == nil {
		return result, domain.ErrAccessDenied
	}
	actor := uuid.Nil
	if p.UserID != nil {
		actor = *p.UserID
	}
	e := s.access.WithWorkspaceLock(ctx, ref.WorkspaceID, actor, func(ctx domain.Context) error {
		expires, e := s.expiry(ctx, p, domain.AssetCapabilityLifetime)
		if e != nil {
			return e
		}
		d, e := s.access.Resolve(ctx, p, ref)
		if e != nil {
			return e
		}
		if d.Access != domain.AccessEdit {
			return domain.ErrAccessDenied
		}
		urls, e := s.assets.PresignPrincipalUpload(ctx, p, ref, requests, expires)
		if e != nil {
			return e
		}
		result = domain.AssetGrant{URLs: urls, ExpiresAt: expires}
		return nil
	})
	if e != nil {
		return domain.AssetGrant{}, e
	}
	return result, nil
}

func (s *CapabilityService) AuthorizeScene(ctx domain.Context, p domain.Principal, ref domain.SceneRef, ttl time.Duration) (time.Time, error) {
	if p.Kind == domain.PrincipalAgent {
		return time.Time{}, domain.ErrAccessDenied
	}
	var expires time.Time
	actor := uuid.Nil
	if p.UserID != nil {
		actor = *p.UserID
	}
	e := s.access.WithWorkspaceLock(ctx, ref.WorkspaceID, actor, func(c domain.Context) error {
		var e error
		expires, e = s.expiry(c, p, ttl)
		if e != nil {
			return e
		}
		_, e = s.access.Resolve(c, p, ref)
		return e
	})
	if e != nil {
		return time.Time{}, e
	}
	return expires, nil
}
