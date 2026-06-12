package domain

import (
	"context"

	"github.com/google/uuid"
)

type AuthUsecase interface {
	Login(ctx context.Context, username, password string) (string, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
}
