package usecase

import (
	"crypto/rand"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const publicSecretBytes = 32

func shareTokenHash(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
func randomShareSecret() (string, error) {
	var b [publicSecretBytes]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

type RoomService struct {
	repo   domain.ShareLinkRepository
	access domain.AccessService
	secret []byte
}

func NewRoomService(repo domain.ShareLinkRepository, access domain.AccessService, secret []byte) (*RoomService, error) {
	if len(secret) < publicSecretBytes {
		return nil, errors.New("COLLAB_JWT_SECRET must contain at least 32 bytes")
	}
	return &RoomService{repo: repo, access: access, secret: append([]byte(nil), secret...)}, nil
}

type roomClaims struct {
	jwt.RegisteredClaims
	Purpose       string               `json:"purpose"`
	WorkspaceID   string               `json:"workspaceId"`
	SceneID       string               `json:"sceneId"`
	PrincipalType domain.PrincipalKind `json:"principalType"`
	PrincipalID   string               `json:"principalId"`
	GrantID       string               `json:"grantId"`
	GrantVersion  string               `json:"grantVersion,omitempty"`
	DisplayName   string               `json:"displayName"`
	Access        string               `json:"access"`
	ACLVersion    string               `json:"aclVersion"`
	SessionID     string               `json:"sid,omitempty"`
	AuthMethod    string               `json:"authMethod,omitempty"`
}

func (s *RoomService) sign(p domain.Principal, ref domain.SceneRef, d domain.AccessDecision, audience, purpose string, expires time.Time) (string, error) {
	now := time.Now()
	claims := roomClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: domain.RoomIssuer, Subject: p.ID, Audience: jwt.ClaimStrings{audience}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires), ID: uuid.NewString()}, Purpose: purpose, WorkspaceID: ref.WorkspaceID.String(), SceneID: ref.SceneID.String(), PrincipalType: p.Kind, PrincipalID: p.ID, GrantID: p.GrantID, GrantVersion: p.GrantVersion, DisplayName: p.DisplayName, Access: d.Access, ACLVersion: strconv.FormatInt(d.ACLVersion, 10), AuthMethod: p.AuthMethod}
	if p.SessionID != uuid.Nil {
		claims.SessionID = p.SessionID.String()
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}
func (s *RoomService) ExchangePublicSession(ctx domain.Context, token, password, name string) (domain.SceneSession, error) {
	var result domain.SceneSession
	if token == "" || len(token) > 256 || len(password) > 72 {
		return result, domain.ErrInvalidShareInput
	}
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > domain.MaxPublicDisplayName {
		return result, domain.ErrInvalidShareInput
	}
	link, e := s.repo.FindByHash(ctx, shareTokenHash(token))
	if e != nil {
		return result, e
	}
	if name == "" && link.Access == domain.AccessView && link.AllowComments {
		name = "Guest"
	}
	// Password work precedes authority locking; recheck its hash inside the transaction.
	passwordHash := link.PasswordHash
	if passwordHash != "" && bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return result, domain.ErrSharePassword
	}
	guest, e := randomShareSecret()
	if e != nil {
		return result, e
	}
	p := domain.Principal{Kind: domain.PrincipalPublic, ID: "guest:" + guest, GrantID: link.ID.String(), GrantVersion: strconv.FormatInt(link.Revision, 10), DisplayName: name}
	ref := domain.SceneRef{WorkspaceID: link.WorkspaceID, SceneID: link.SceneID}
	e = s.access.WithWorkspaceLock(ctx, ref.WorkspaceID, uuid.Nil, func(ctx domain.Context) error {
		current, e := s.repo.FindByID(ctx, link.ID)
		if e != nil {
			return e
		}
		if current.PasswordHash != passwordHash || current.TokenHash != shareTokenHash(token) {
			return domain.ErrSharePassword
		}
		d, e := s.access.Resolve(ctx, p, ref)
		if e != nil {
			return e
		}
		if d.Access == domain.AccessEdit && name == "" {
			return domain.ErrInvalidShareInput
		}
		expires := time.Now().Add(domain.RoomTokenLifetime)
		if current.ExpiresAt != nil && current.ExpiresAt.Before(expires) {
			expires = *current.ExpiresAt
		}
		raw, e := s.sign(p, ref, d, domain.SceneSessionAudience, domain.SceneSessionPurpose, expires)
		if e != nil {
			return e
		}
		result = domain.SceneSession{Token: raw, ExpiresAt: expires, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, Access: d.Access, PrincipalID: p.ID, DisplayName: name}
		return nil
	})
	if e != nil {
		return domain.SceneSession{}, e
	}
	return result, nil
}
func (s *RoomService) MintRoomToken(ctx domain.Context, p domain.Principal, ref domain.SceneRef) (domain.RoomToken, error) {
	var result domain.RoomToken
	actor := uuid.Nil
	if p.Kind == domain.PrincipalUser {
		var e error
		actor, e = normalizedUser(p)
		if e != nil {
			return result, e
		}
	} else if p.Kind != domain.PrincipalPublic && p.Kind != domain.PrincipalAgent {
		return result, domain.ErrAccessDenied
	}
	e := s.access.WithWorkspaceLock(ctx, ref.WorkspaceID, actor, func(ctx domain.Context) error {
		if p.Kind == domain.PrincipalPublic && (p.CredentialExpiresAt.IsZero() || !time.Now().Before(p.CredentialExpiresAt)) {
			return domain.ErrInvalidSceneSession
		}
		d, e := s.access.Resolve(ctx, p, ref)
		if e != nil {
			return e
		}
		expires := time.Now().Add(domain.RoomTokenLifetime)
		if p.Kind == domain.PrincipalUser {
			p.DisplayName, e = s.repo.UserDisplayName(ctx, actor)
			if e != nil {
				return e
			}
			p.GrantID = "membership:" + ref.WorkspaceID.String() + ":" + actor.String()
		} else if p.Kind == domain.PrincipalPublic {
			if d.Access == domain.AccessEdit && strings.TrimSpace(p.DisplayName) == "" {
				return domain.ErrInvalidShareInput
			}
			id, e := uuid.Parse(p.GrantID)
			if e != nil {
				return domain.ErrAccessDenied
			}
			link, e := s.repo.FindByID(ctx, id)
			if e != nil {
				return e
			}
			if link.ExpiresAt != nil && link.ExpiresAt.Before(expires) {
				expires = *link.ExpiresAt
			}
		}
		if p.Kind == domain.PrincipalAgent {
			if !p.CredentialExpiresAt.IsZero() && p.CredentialExpiresAt.Before(expires) {
				expires = p.CredentialExpiresAt
			}
			p.DisplayName = "Agent"
		}
		raw, e := s.sign(p, ref, d, domain.RoomAudience, domain.RoomPurpose, expires)
		if e != nil {
			return e
		}
		result = domain.RoomToken{Token: raw, ExpiresAt: expires}
		if p.Kind == domain.PrincipalPublic {
			renewed, e := s.sign(p, ref, d, domain.SceneSessionAudience, domain.SceneSessionPurpose, expires)
			if e != nil {
				return e
			}
			result.SceneSession = &domain.SceneSession{Token: renewed, ExpiresAt: expires, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, Access: d.Access, PrincipalID: p.ID, DisplayName: p.DisplayName}
		}
		return nil
	})
	if e != nil {
		return domain.RoomToken{}, e
	}
	return result, nil
}
func (s *RoomService) parse(raw, audience, purpose string) (domain.SceneSessionIdentity, error) {
	var identity domain.SceneSessionIdentity
	claims := &roomClaims{}
	token, e := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return s.secret, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(domain.RoomIssuer), jwt.WithAudience(audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if e != nil || !token.Valid || claims.Purpose != purpose || claims.IssuedAt == nil || claims.ExpiresAt == nil || claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) > domain.RoomTokenLifetime || claims.PrincipalID == "" || claims.Subject != claims.PrincipalID || claims.ID == "" || len(claims.Audience) != 1 {
		return identity, domain.ErrInvalidSceneSession
	}
	if claims.Access != domain.AccessView && claims.Access != domain.AccessEdit {
		return identity, domain.ErrInvalidSceneSession
	}
	version, e := strconv.ParseInt(claims.ACLVersion, 10, 64)
	if e != nil || version < 1 {
		return identity, domain.ErrInvalidSceneSession
	}
	w, e := uuid.Parse(claims.WorkspaceID)
	if e != nil || w == uuid.Nil {
		return identity, domain.ErrInvalidSceneSession
	}
	scene, e := uuid.Parse(claims.SceneID)
	if e != nil || scene == uuid.Nil {
		return identity, domain.ErrInvalidSceneSession
	}
	p := domain.Principal{Kind: claims.PrincipalType, ID: claims.PrincipalID, GrantID: claims.GrantID, GrantVersion: claims.GrantVersion, DisplayName: claims.DisplayName, CredentialExpiresAt: claims.ExpiresAt.Time, AuthMethod: claims.AuthMethod}
	if claims.SessionID != "" {
		sid, err := uuid.Parse(claims.SessionID)
		if err != nil || sid == uuid.Nil {
			return identity, domain.ErrInvalidSceneSession
		}
		p.SessionID = sid
	}
	switch p.Kind {
	case domain.PrincipalPublic:
		if p.GrantVersion == "" || !strings.HasPrefix(p.ID, "guest:") || len(p.ID) <= len("guest:") {
			return identity, domain.ErrInvalidSceneSession
		}
		id, e := uuid.Parse(p.GrantID)
		if e != nil || id == uuid.Nil {
			return identity, domain.ErrInvalidSceneSession
		}
	case domain.PrincipalAgent:
		id, e := uuid.Parse(p.GrantID)
		if purpose != domain.RoomPurpose || e != nil || id == uuid.Nil || p.ID != "agent:"+id.String() || p.GrantVersion != "" {
			return identity, domain.ErrInvalidSceneSession
		}
	case domain.PrincipalUser:
		if purpose == domain.SceneSessionPurpose {
			return identity, domain.ErrInvalidSceneSession
		}
		id, e := uuid.Parse(p.ID)
		if e != nil || id == uuid.Nil || p.GrantID != "membership:"+w.String()+":"+id.String() {
			return identity, domain.ErrInvalidSceneSession
		}
		p.UserID = &id
	default:
		return identity, domain.ErrInvalidSceneSession
	}
	if utf8.RuneCountInString(p.DisplayName) > domain.MaxPublicDisplayName || (p.Kind == domain.PrincipalPublic && claims.Access == domain.AccessEdit && strings.TrimSpace(p.DisplayName) == "") {
		return identity, domain.ErrInvalidSceneSession
	}
	return domain.SceneSessionIdentity{Principal: p, SceneRef: domain.SceneRef{WorkspaceID: w, SceneID: scene}, ExpiresAt: claims.ExpiresAt.Time}, nil
}
func (s *RoomService) ParseSceneSession(raw string) (domain.SceneSessionIdentity, error) {
	return s.parse(raw, domain.SceneSessionAudience, domain.SceneSessionPurpose)
}
func (s *RoomService) VerifyRoomToken(raw string) (domain.SceneSessionIdentity, error) {
	return s.parse(raw, domain.RoomAudience, domain.RoomPurpose)
}
