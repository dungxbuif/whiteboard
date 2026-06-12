package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type mockAuthUsecase struct {
	domain.AuthUsecase
}

func (mockAuthUsecase) Login(_ context.Context, username, password string) (string, error) {
	if username != domain.TemporaryAdminUsername || password != domain.TemporaryAdminPassword {
		return "", context.Canceled
	}
	return "test-token", nil
}

func (mockAuthUsecase) GetUserByID(context.Context, uuid.UUID) (*domain.User, error) {
	return &domain.User{}, nil
}

func TestAuthHandlerLogin(t *testing.T) {
	handler := NewAuthHandler(mockAuthUsecase{})
	body, err := json.Marshal(LoginRequest{
		Username: domain.TemporaryAdminUsername,
		Password: domain.TemporaryAdminPassword,
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.Login(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}
