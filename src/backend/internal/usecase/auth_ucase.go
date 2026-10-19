package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"devtool/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

const googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
const dummyPasswordHash = "$2a$10$fVg.L7MsDdhllg9xhYREuuKgNfjqX4Gr.KPYRoFypvnbgTvlsz1M2"

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	FrontendURL  string
	UserInfoURL  string
	Endpoint     oauth2.Endpoint
}

type authUsecase struct {
	userRepo        domain.UserRepository
	jwtSecret       []byte
	oauth           *oauth2.Config
	userInfoURL     string
	httpClient      *http.Client
	comparePassword func([]byte, []byte) error
}

func NewAuthUsecase(userRepo domain.UserRepository, jwtSecret []byte, configs ...OAuthConfig) domain.AuthUsecase {
	config := OAuthConfig{}
	if len(configs) > 0 {
		config = configs[0]
	}
	userInfoURL := config.UserInfoURL
	if userInfoURL == "" {
		userInfoURL = googleUserInfoURL
	}
	endpoint := config.Endpoint
	if endpoint.AuthURL == "" || endpoint.TokenURL == "" {
		endpoint = google.Endpoint
	}
	return &authUsecase{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		oauth: &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Endpoint:     endpoint,
			RedirectURL:  config.RedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
		},
		userInfoURL:     userInfoURL,
		httpClient:      http.DefaultClient,
		comparePassword: bcrypt.CompareHashAndPassword,
	}
}

func (u *authUsecase) OAuthLoginURL(state string) string {
	return u.oauth.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (u *authUsecase) OAuthCallback(ctx context.Context, code string) (string, error) {
	if code == "" {
		return "", fmt.Errorf("OAuth authorization code is required")
	}
	if u.oauth.ClientID == "" || u.oauth.ClientSecret == "" || u.oauth.RedirectURL == "" {
		return "", fmt.Errorf("Google OAuth is not configured")
	}
	token, err := u.oauth.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange Google OAuth code: %w", err)
	}
	client := u.oauth.Client(ctx, token)
	if u.httpClient != http.DefaultClient {
		client = u.httpClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.userInfoURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Google userinfo request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Google profile: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Google profile returned status %d", response.StatusCode)
	}
	var profile struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(response.Body).Decode(&profile); err != nil {
		return "", fmt.Errorf("failed to decode Google profile: %w", err)
	}
	if profile.Subject == "" || profile.Email == "" {
		return "", fmt.Errorf("Google profile is missing subject or email")
	}

	user, err := u.userRepo.FindByGoogleSubject(ctx, profile.Subject)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user, err = u.userRepo.FindByEmail(ctx, profile.Email)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = &domain.User{ID: uuid.New(), Email: profile.Email}
		} else if err != nil {
			return "", fmt.Errorf("failed to load user by email: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("failed to load user by Google subject: %w", err)
	}

	user.Email = profile.Email
	user.Name = profile.Name
	user.AvatarURL = profile.Picture
	user.GoogleSubject = &profile.Subject
	now := time.Now()
	user.LastLoginAt = &now
	if err := u.userRepo.Save(ctx, user); err != nil {
		return "", fmt.Errorf("failed to save Google user: %w", err)
	}
	return u.issueToken(user)
}

func (u *authUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return u.userRepo.FindByID(ctx, id)
}

func (u *authUsecase) PasswordLogin(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return "", errors.New("email and password are required")
	}
	user, err := u.userRepo.FindByEmail(ctx, email)
	passwordHash := dummyPasswordHash
	validAccount := err == nil && user != nil && user.PasswordHash != nil && !user.DeletedAt.Valid
	if validAccount {
		passwordHash = *user.PasswordHash
	}
	passwordMatches := u.comparePassword([]byte(passwordHash), []byte(password)) == nil
	if !validAccount || !passwordMatches {
		return "", errors.New("invalid email or password")
	}
	now := time.Now()
	user.LastLoginAt = &now
	if err := u.userRepo.Save(ctx, user); err != nil {
		return "", fmt.Errorf("failed to update last login: %w", err)
	}
	return u.issueToken(user)
}

func (u *authUsecase) issueToken(user *domain.User) (string, error) {
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
