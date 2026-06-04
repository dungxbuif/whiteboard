package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email         string     `gorm:"uniqueIndex:users_email_lower_key;not null"`
	Name          string     `gorm:"not null"`
	AvatarURL     string
	GoogleSubject *string    `gorm:"uniqueIndex:users_google_subject_key"`
	LastLoginAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

type UserRepository interface {
	FindByID(ctx Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx Context, email string) (*User, error)
	Save(ctx Context, user *User) error
}
