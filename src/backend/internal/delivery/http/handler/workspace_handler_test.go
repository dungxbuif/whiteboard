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

type mockWorkspaceUsecase struct {
	domain.WorkspaceUsecase
}

func (m *mockWorkspaceUsecase) Create(ctx context.Context, userID uuid.UUID, name string) (*domain.Workspace, error) {
	return &domain.Workspace{ID: uuid.New(), Name: name, OwnerID: userID}, nil
}

func (m *mockWorkspaceUsecase) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	return []domain.Workspace{{ID: uuid.New(), Name: "WS 1", OwnerID: userID}}, nil
}

func TestWorkspaceHandler_Create(t *testing.T) {
	ucase := &mockWorkspaceUsecase{}
	h := NewWorkspaceHandler(ucase)
	userID := uuid.New()

	body, _ := json.Marshal(CreateWorkspaceRequest{Name: "Test WS"})
	req := httptest.NewRequest("POST", "/workspaces", bytes.NewBuffer(body))
	
	// Manually inject userID into context (simulating middleware)
	ctx := context.WithValue(req.Context(), domain.ContextKeyUserID, userID)
	req = req.WithContext(ctx)
	
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var res domain.Workspace
	json.Unmarshal(rr.Body.Bytes(), &res)
	if res.Name != "Test WS" {
		t.Errorf("expected name Test WS, got %s", res.Name)
	}
}

func TestWorkspaceHandler_List(t *testing.T) {
	ucase := &mockWorkspaceUsecase{}
	h := NewWorkspaceHandler(ucase)
	userID := uuid.New()

	req := httptest.NewRequest("GET", "/workspaces", nil)
	ctx := context.WithValue(req.Context(), domain.ContextKeyUserID, userID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var res []domain.Workspace
	json.Unmarshal(rr.Body.Bytes(), &res)
	if len(res) != 1 {
		t.Errorf("expected 1 workspace, got %d", len(res))
	}
}
