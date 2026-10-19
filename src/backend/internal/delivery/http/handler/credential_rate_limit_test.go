package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"

	"github.com/google/uuid"
)

type rateLimitedAuthUsecase struct {
	domain.AuthUsecase
	calls int
}

func (u *rateLimitedAuthUsecase) PasswordLogin(context.Context, string, string) (string, error) {
	u.calls++
	return "token", nil
}

type rateLimitedInvitationUsecase struct {
	domain.InvitationUsecase
	calls int
}

func (u *rateLimitedInvitationUsecase) RegisterWithInvitation(context.Context, string, domain.RegistrationInput) (domain.User, error) {
	u.calls++
	return domain.User{ID: uuid.New()}, nil
}

type rateLimitedRoomService struct {
	domain.RoomService
	calls int
}

func (s *rateLimitedRoomService) ExchangePublicSession(context.Context, string, string, string) (domain.SceneSession, error) {
	s.calls++
	return domain.SceneSession{Token: "session"}, nil
}

func newHandlerCredentialLimiter(t *testing.T) *middleware.CredentialRateLimiter {
	t.Helper()
	limiter, err := middleware.NewDefaultCredentialRateLimiter()
	if err != nil {
		t.Fatal(err)
	}
	return limiter
}

func TestPasswordLoginStopsBeforeUsecaseWhenAccountLimitIsExhausted(t *testing.T) {
	uc := &rateLimitedAuthUsecase{}
	h := NewAuthHandler(uc)
	h.SetCredentialLimiter(newHandlerCredentialLimiter(t))
	for attempt := 1; attempt <= 11; attempt++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"USER@example.test","password":"invalid-password"}`))
		h.PasswordLogin(rr, req)
		if attempt <= 10 && rr.Code != http.StatusOK {
			t.Fatalf("attempt %d status=%d body=%s", attempt, rr.Code, rr.Body.String())
		}
		if attempt == 11 && rr.Code != http.StatusTooManyRequests {
			t.Fatalf("limited attempt status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
	if uc.calls != 10 {
		t.Fatalf("login usecase calls=%d, want 10", uc.calls)
	}
}

func TestInvitationRegistrationStopsBeforeUsecaseWhenTokenLimitIsExhausted(t *testing.T) {
	uc := &rateLimitedInvitationUsecase{}
	h := NewInvitationHandler(uc)
	h.SetCredentialLimiter(newHandlerCredentialLimiter(t))
	token := strings.Repeat("a", 43)
	for attempt := 1; attempt <= 6; attempt++ {
		rr := httptest.NewRecorder()
		body := `{"token":"` + token + `","email":"new@example.test","password":"valid-password","name":"New User"}`
		h.Register(rr, httptest.NewRequest(http.MethodPost, "/api/v1/invitations/register", strings.NewReader(body)))
		if attempt <= 5 && rr.Code != http.StatusCreated {
			t.Fatalf("attempt %d status=%d body=%s", attempt, rr.Code, rr.Body.String())
		}
		if attempt == 6 && rr.Code != http.StatusTooManyRequests {
			t.Fatalf("limited attempt status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
	if uc.calls != 5 {
		t.Fatalf("invitation usecase calls=%d, want 5", uc.calls)
	}
}

func TestPublicSceneExchangeStopsBeforePasswordWorkWhenTokenLimitIsExhausted(t *testing.T) {
	rooms := &rateLimitedRoomService{}
	h := NewRoomHandler(rooms, nil)
	h.SetCredentialLimiter(newHandlerCredentialLimiter(t))
	for attempt := 1; attempt <= 16; attempt++ {
		rr := httptest.NewRecorder()
		body := `{"token":"public-token","password":"invalid-password"}`
		h.Exchange(rr, httptest.NewRequest(http.MethodPost, "/api/v1/public/scene-session", strings.NewReader(body)))
		if attempt <= 15 && rr.Code != http.StatusOK {
			t.Fatalf("attempt %d status=%d body=%s", attempt, rr.Code, rr.Body.String())
		}
		if attempt == 16 && rr.Code != http.StatusTooManyRequests {
			t.Fatalf("limited attempt status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
	if rooms.calls != 15 {
		t.Fatalf("room usecase calls=%d, want 15", rooms.calls)
	}
}
