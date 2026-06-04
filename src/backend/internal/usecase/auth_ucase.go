package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"devtool/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type authUsecase struct {
	oauthConfig *oauth2.Config
	userRepo    domain.UserRepository
	jwtSecret   []byte
}

func NewAuthUsecase(oauthConfig *oauth2.Config, userRepo domain.UserRepository, jwtSecret []byte) domain.AuthUsecase {
	return &authUsecase{
		oauthConfig: oauthConfig,
		userRepo:    userRepo,
		jwtSecret:   jwtSecret,
	}
}

func (u *authUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return u.userRepo.FindByID(ctx, id)
}

func (u *authUsecase) GetGoogleLoginURL(state string) string {
	return u.oauthConfig.AuthCodeURL(state)
}

func (u *authUsecase) HandleGoogleCallback(ctx context.Context, code string) (string, error) {
	token, err := u.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}

	client := u.oauthConfig.Client(ctx, token)
	resp, err := client.Get(domain.GoogleUserInfoURL)
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google api returned status: %d", resp.StatusCode)
	}

	var googleUser struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return "", fmt.Errorf("failed to decode user info: %w", err)
	}

	user, err := u.userRepo.FindByEmail(ctx, googleUser.Email)
	if err != nil {
		now := time.Now()
		user = &domain.User{
			ID:            uuid.New(),
			Email:         googleUser.Email,
			Name:          googleUser.Name,
			AvatarURL:     googleUser.Picture,
			GoogleSubject: &googleUser.Sub,
			LastLoginAt:   &now,
		}
	} else {
		now := time.Now()
		user.LastLoginAt = &now
		user.Name = googleUser.Name
		user.AvatarURL = googleUser.Picture
		user.GoogleSubject = &googleUser.Sub
	}

	if err := u.userRepo.Save(ctx, user); err != nil {
		return "", fmt.Errorf("failed to save user: %w", err)
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"exp":   time.Now().Add(domain.DefaultSessionDuration).Unix(),
	})

	tokenString, err := jwtToken.SignedString(u.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
