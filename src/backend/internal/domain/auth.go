package domain

import (
	"context"

	"github.com/google/uuid"
)

type AuthUsecase interface {
	GetGoogleLoginURL(state string) string
	HandleGoogleCallback(ctx context.Context, code string) (string, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
}
