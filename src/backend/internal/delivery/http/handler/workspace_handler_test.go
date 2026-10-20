package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devtool/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockWorkspaceUsecase struct {
	domain.WorkspaceUsecase
	updateErr   error
	updatedName string
}

func (m *mockWorkspaceUsecase) UpdateName(_ context.Context, userID, workspaceID uuid.UUID, name string) (*domain.Workspace, error) {
	m.updatedName = name
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return &domain.Workspace{ID: workspaceID, OwnerID: userID, Name: name}, nil
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
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"ownerId"`)) {
		t.Errorf("expected camelCase ownerId JSON field, got %s", rr.Body.String())
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

func TestWorkspaceHandler_Update(t *testing.T) {
	userID, workspaceID := uuid.New(), uuid.New()
	ucase := &mockWorkspaceUsecase{}
	router := chi.NewRouter()
	router.Patch("/{id}", NewWorkspaceHandler(ucase).Update)
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"valid", `{"name":"  New workspace  "}`, http.StatusOK},
		{"blank", `{"name":"   "}`, http.StatusBadRequest},
		{"short", `{"name":"ab"}`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/"+workspaceID.String(), strings.NewReader(tc.body))
			req = req.WithContext(context.WithValue(req.Context(), domain.ContextKeyUserID, userID))
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
	if ucase.updatedName != "New workspace" {
		t.Fatalf("name=%q", ucase.updatedName)
	}
	ucase.updateErr = domain.ErrAccessDenied
	req := httptest.NewRequest(http.MethodPatch, "/"+workspaceID.String(), strings.NewReader(`{"name":"Denied"}`))
	req = req.WithContext(context.WithValue(req.Context(), domain.ContextKeyUserID, userID))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
