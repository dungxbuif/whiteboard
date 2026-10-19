package usecase

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"devtool/internal/domain"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const invitationTokenBytes = 32
const invitationTokenLength = 43

type invitationUsecase struct {
	repo        domain.InvitationRepository
	access      domain.AccessService
	frontendURL string
	now         func() time.Time
	validate    *validator.Validate
}

func NewInvitationUsecase(repo domain.InvitationRepository, access domain.AccessService, frontendURL string, clocks ...func() time.Time) domain.InvitationUsecase {
	now := time.Now
	if len(clocks) > 0 {
		now = clocks[0]
	}
	return &invitationUsecase{repo: repo, access: access, frontendURL: strings.TrimRight(frontendURL, "/"), now: now, validate: validator.New()}
}
func (u *invitationUsecase) CreateInvitation(ctx domain.Context, owner, w uuid.UUID, email string, role domain.WorkspaceRole) (domain.InviteResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if u.validate.Var(email, "required,email,max=254") != nil || (role != domain.WorkspaceRoleEditor && role != domain.WorkspaceRoleViewer) {
		return domain.InviteResult{}, domain.ErrInvitationInput
	}
	bytes := make([]byte, invitationTokenBytes)
	if _, e := rand.Read(bytes); e != nil {
		return domain.InviteResult{}, e
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	now := u.now().UTC().Truncate(time.Microsecond)
	i := domain.Invitation{ID: uuid.New(), WorkspaceID: w, Email: email, Role: role, InviterID: owner, TokenHash: invitationHash(token), CreatedAt: now, ExpiresAt: now.Add(domain.InvitationLifetime)}
	err := u.access.WithWorkspaceLock(ctx, w, owner, func(ctx domain.Context) error {
		if e := u.access.RequireOwner(ctx, owner, w); e != nil {
			return e
		}
		return u.repo.Create(ctx, &i)
	})
	if err != nil {
		return domain.InviteResult{}, err
	}
	i.TokenHash = ""
	return domain.InviteResult{Invitation: i, InviteURL: u.frontendURL + "/invite/" + token}, nil
}
func (u *invitationUsecase) ListInvitations(ctx domain.Context, owner, w uuid.UUID) (rows []domain.Invitation, err error) {
	err = u.access.WithWorkspaceLock(ctx, w, owner, func(ctx domain.Context) error {
		if e := u.access.RequireOwner(ctx, owner, w); e != nil {
			return e
		}
		var e error
		rows, e = u.repo.ListPending(ctx, w, u.now())
		return e
	})
	return
}
func (u *invitationUsecase) RevokeInvitation(ctx domain.Context, owner, w, id uuid.UUID) error {
	return u.access.WithWorkspaceLock(ctx, w, owner, func(ctx domain.Context) error {
		if e := u.access.RequireOwner(ctx, owner, w); e != nil {
			return e
		}
		return u.repo.Revoke(ctx, w, id, u.now())
	})
}
func invitationHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
func (u *invitationUsecase) withInvitation(ctx domain.Context, actor uuid.UUID, token string, fn func(domain.Context, *domain.Invitation, time.Time) error) error {
	if len(token) != invitationTokenLength {
		return domain.ErrInvitationInvalid
	}
	if _, err := base64.RawURLEncoding.DecodeString(token); err != nil {
		return domain.ErrInvitationInvalid
	}
	hash := invitationHash(token)
	initial, err := u.repo.FindByHash(ctx, hash, false)
	if err != nil {
		return err
	}
	return u.access.WithWorkspaceLock(ctx, initial.WorkspaceID, actor, func(ctx domain.Context) error {
		i, err := u.repo.FindByHash(ctx, hash, true)
		if err != nil {
			return err
		}
		if i.AcceptedAt != nil || i.RevokedAt != nil {
			return domain.ErrInvitationInvalid
		}
		now := u.now()
		if !now.Before(i.ExpiresAt) {
			return domain.ErrInvitationExpired
		}
		if err = u.access.RequireOwner(ctx, i.InviterID, i.WorkspaceID); err != nil {
			return err
		}
		return fn(ctx, i, now)
	})
}
func (u *invitationUsecase) AcceptInvitation(ctx domain.Context, user uuid.UUID, token string) (membership domain.WorkspaceMember, err error) {
	err = u.withInvitation(ctx, user, token, func(ctx domain.Context, i *domain.Invitation, now time.Time) error {
		account, e := u.repo.FindUser(ctx, user)
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return domain.ErrAccessDenied
		}
		if e != nil {
			return e
		}
		if strings.ToLower(strings.TrimSpace(account.Email)) != i.Email {
			return domain.ErrInvitationEmail
		}
		membership, e = u.repo.Join(ctx, i, user, now)
		return e
	})
	return
}
func (u *invitationUsecase) RegisterWithInvitation(ctx domain.Context, token string, input domain.RegistrationInput) (user domain.User, err error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Name = strings.TrimSpace(input.Name)
	if u.validate.Struct(input) != nil {
		return user, domain.ErrRegistrationRejected
	}
	// Hash outside the authority lock; expensive password work must not block revocation.
	hash, e := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if e != nil {
		return user, domain.ErrRegistrationRejected
	}
	value := string(hash)
	err = u.withInvitation(ctx, uuid.Nil, token, func(ctx domain.Context, i *domain.Invitation, now time.Time) error {
		if i.Email != input.Email {
			return domain.ErrRegistrationRejected
		}
		exists, e := u.repo.EmailExists(ctx, input.Email)
		if e != nil {
			return e
		}
		if exists {
			return domain.ErrRegistrationRejected
		}
		user = domain.User{ID: uuid.New(), Email: input.Email, Name: input.Name, PasswordHash: &value}
		if e = u.repo.CreateUser(ctx, &user); e != nil {
			return e
		}
		if e = EnsureDefaultWorkspace(ctx, u.repo, user.ID); e != nil {
			return e
		}
		_, e = u.repo.Join(ctx, i, user.ID, now)
		return e
	})
	if err != nil {
		user = domain.User{}
		if errors.Is(err, domain.ErrInvitationInvalid) || errors.Is(err, domain.ErrInvitationEmail) || errors.Is(err, domain.ErrInaccessible) || errors.Is(err, domain.ErrAccessDenied) || errors.Is(err, gorm.ErrDuplicatedKey) {
			err = domain.ErrRegistrationRejected
		}
	}
	return
}
