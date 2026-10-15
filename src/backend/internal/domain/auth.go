package domain

import (
	"context"

	"github.com/google/uuid"
)

type AuthUsecase interface {
	OAuthLoginURL(state string) string
	OAuthCallback(ctx context.Context, code string) (string, error)
	PasswordLogin(ctx context.Context, email, password string) (string, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
}
