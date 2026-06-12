package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"devtool/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type authUsecase struct {
	userRepo  domain.UserRepository
	jwtSecret []byte
}

func NewAuthUsecase(userRepo domain.UserRepository, jwtSecret []byte) domain.AuthUsecase {
	return &authUsecase{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (u *authUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return u.userRepo.FindByID(ctx, id)
}

func (u *authUsecase) Login(ctx context.Context, username, password string) (string, error) {
	if username != domain.TemporaryAdminUsername || password != domain.TemporaryAdminPassword {
		return "", fmt.Errorf("invalid credentials")
	}

	user, err := u.userRepo.FindByEmail(ctx, domain.TemporaryAdminEmail)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = &domain.User{
			ID:    uuid.New(),
			Email: domain.TemporaryAdminEmail,
			Name:  domain.TemporaryAdminName,
		}
	} else if err != nil {
		return "", fmt.Errorf("failed to load admin user: %w", err)
	}

	now := time.Now()
	user.LastLoginAt = &now
	if err := u.userRepo.Save(ctx, user); err != nil {
		return "", fmt.Errorf("failed to save admin user: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"exp":   time.Now().Add(domain.DefaultSessionDuration).Unix(),
	})

	tokenString, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}
