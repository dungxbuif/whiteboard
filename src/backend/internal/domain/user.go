package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID
	Email         string
	Name          string
	AvatarURL     string
	GoogleSubject string
	LastLoginAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type UserRepository interface {
	FindByID(ctx Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx Context, email string) (*User, error)
	Save(ctx Context, user *User) error
}
