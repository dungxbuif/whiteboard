package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email         string         `json:"email" gorm:"uniqueIndex:users_email_lower_key;not null"`
	Name          string         `json:"name" gorm:"not null"`
	AvatarURL     string         `json:"avatarUrl,omitempty"`
	GoogleSubject *string        `json:"-" gorm:"uniqueIndex:users_google_subject_key"`
	LastLoginAt   *time.Time     `json:"lastLoginAt,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

type UserRepository interface {
	FindByID(ctx Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx Context, email string) (*User, error)
	Save(ctx Context, user *User) error
}
