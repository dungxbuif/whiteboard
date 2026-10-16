package usecase

import (
	"devtool/internal/domain"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type ShareLinkService struct {
	repo     domain.ShareLinkRepository
	access   domain.AccessService
	validate *validator.Validate
}

func NewShareLinkService(repo domain.ShareLinkRepository, access domain.AccessService) *ShareLinkService {
	return &ShareLinkService{repo: repo, access: access, validate: validator.New()}
}
func (s *ShareLinkService) owner(ctx domain.Context, actor uuid.UUID, ref domain.SceneRef) error {
	if e := s.access.RequireOwner(ctx, actor, ref.WorkspaceID); e != nil {
		return e
	}
	_, e := s.access.Resolve(ctx, domain.UserPrincipal(actor), ref)
	return e
}
func (s *ShareLinkService) Create(ctx domain.Context, actor uuid.UUID, ref domain.SceneRef, in domain.CreateShareLink) (domain.CreatedShareLink, error) {
	var result domain.CreatedShareLink
	if len(in.Password) > 72 || s.validate.Struct(in) != nil || (in.ExpiresAt != nil && !in.ExpiresAt.After(time.Now())) {
		return result, domain.ErrInvalidShareInput
	}
	// Reject unauthorized work before spending CPU on bcrypt, then recheck under lock.
	if e := s.owner(ctx, actor, ref); e != nil {
		return result, e
	}
	raw, e := randomShareSecret()
	if e != nil {
		return result, e
	}
	hash := ""
	if in.Password != "" {
		b, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if e != nil {
			return result, e
		}
		hash = string(b)
	}
	link := domain.ShareLink{ID: uuid.New(), WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, TokenHash: shareTokenHash(raw), PasswordHash: hash, Access: in.Access, Active: true, ExpiresAt: in.ExpiresAt}
	e = s.access.WithWorkspaceLock(ctx, ref.WorkspaceID, actor, func(ctx domain.Context) error {
		if e := s.owner(ctx, actor, ref); e != nil {
			return e
		}
		return s.repo.Create(ctx, &link)
	})
	if e != nil {
		return result, e
	}
	return domain.CreatedShareLink{ShareLink: link, Token: raw}, nil
}
func (s *ShareLinkService) List(ctx domain.Context, actor uuid.UUID, ref domain.SceneRef) ([]domain.ShareLink, error) {
	var links []domain.ShareLink
	e := s.access.WithWorkspaceLock(ctx, ref.WorkspaceID, actor, func(ctx domain.Context) error {
		if e := s.owner(ctx, actor, ref); e != nil {
			return e
		}
		var e error
		links, e = s.repo.List(ctx, ref)
		return e
	})
	return links, e
}
func (s *ShareLinkService) Revoke(ctx domain.Context, actor uuid.UUID, ref domain.SceneRef, id uuid.UUID) error {
	return s.access.WithWorkspaceLock(ctx, ref.WorkspaceID, actor, func(ctx domain.Context) error {
		if e := s.owner(ctx, actor, ref); e != nil {
			return e
		}
		return s.repo.Revoke(ctx, ref, id)
	})
}
