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
	hasher      InvitationPasswordHasher
}

type InvitationPasswordHasher interface {
	HashPassword(string) (string, error)
}

type bcryptInvitationPasswordHasher struct{}

func (bcryptInvitationPasswordHasher) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func NewInvitationUsecase(repo domain.InvitationRepository, access domain.AccessService, frontendURL string, clocks ...func() time.Time) domain.InvitationUsecase {
	return NewInvitationUsecaseWithPasswordHasher(repo, access, frontendURL, bcryptInvitationPasswordHasher{}, clocks...)
}

func NewInvitationUsecaseWithPasswordHasher(repo domain.InvitationRepository, access domain.AccessService, frontendURL string, hasher InvitationPasswordHasher, clocks ...func() time.Time) domain.InvitationUsecase {
	now := time.Now
	if len(clocks) > 0 {
		now = clocks[0]
	}
	if hasher == nil {
		hasher = bcryptInvitationPasswordHasher{}
	}
	return &invitationUsecase{repo: repo, access: access, frontendURL: strings.TrimRight(frontendURL, "/"), now: now, validate: validator.New(), hasher: hasher}
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

func validateInvitationState(invitation *domain.Invitation, now time.Time) error {
	if invitation == nil || invitation.AcceptedAt != nil || invitation.RevokedAt != nil {
		return domain.ErrInvitationInvalid
	}
	if !now.Before(invitation.ExpiresAt) {
		return domain.ErrInvitationExpired
	}
	return nil
}

func (u *invitationUsecase) prepareInvitation(ctx domain.Context, token string) (*domain.Invitation, string, error) {
	if len(token) != invitationTokenLength {
		return nil, "", domain.ErrInvitationInvalid
	}
	if _, err := base64.RawURLEncoding.DecodeString(token); err != nil {
		return nil, "", domain.ErrInvitationInvalid
	}
	hash := invitationHash(token)
	initial, err := u.repo.FindByHash(ctx, hash, false)
	if err != nil {
		return nil, "", err
	}
	if err = validateInvitationState(initial, u.now()); err != nil {
		return nil, "", err
	}
	return initial, hash, nil
}

func (u *invitationUsecase) withPreparedInvitation(ctx domain.Context, actor uuid.UUID, initial *domain.Invitation, hash string, fn func(domain.Context, *domain.Invitation, time.Time) error) error {
	return u.access.WithWorkspaceLock(ctx, initial.WorkspaceID, actor, func(ctx domain.Context) error {
		i, err := u.repo.FindByHash(ctx, hash, true)
		if err != nil {
			return err
		}
		if i.WorkspaceID != initial.WorkspaceID {
			return domain.ErrInvitationInvalid
		}
		now := u.now()
		if err = validateInvitationState(i, now); err != nil {
			return err
		}
		if err = u.access.RequireOwner(ctx, i.InviterID, i.WorkspaceID); err != nil {
			return err
		}
		return fn(ctx, i, now)
	})
}

func (u *invitationUsecase) withInvitation(ctx domain.Context, actor uuid.UUID, token string, fn func(domain.Context, *domain.Invitation, time.Time) error) error {
	initial, hash, err := u.prepareInvitation(ctx, token)
	if err != nil {
		return err
	}
	return u.withPreparedInvitation(ctx, actor, initial, hash, fn)
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
	initial, tokenHash, e := u.prepareInvitation(ctx, token)
	if e != nil {
		return user, registrationInvitationError(e)
	}
	// Hash only after initial token validation and outside the authority lock;
	// expensive password work must not block revocation.
	value, e := u.hasher.HashPassword(input.Password)
	if e != nil {
		return user, domain.ErrRegistrationRejected
	}
	err = u.withPreparedInvitation(ctx, uuid.Nil, initial, tokenHash, func(ctx domain.Context, i *domain.Invitation, now time.Time) error {
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
		err = registrationInvitationError(err)
	}
	return
}

func registrationInvitationError(err error) error {
	if errors.Is(err, domain.ErrInvitationInvalid) || errors.Is(err, domain.ErrInvitationEmail) || errors.Is(err, domain.ErrInaccessible) || errors.Is(err, domain.ErrAccessDenied) || errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrRegistrationRejected
	}
	return err
}
