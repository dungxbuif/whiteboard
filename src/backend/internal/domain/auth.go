package domain

import (
	"context"
)

type AuthUsecase interface {
	GetGoogleLoginURL(state string) string
	HandleGoogleCallback(ctx context.Context, code string) (string, error)
}
