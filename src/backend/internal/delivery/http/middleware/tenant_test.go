package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"devtool/internal/domain"
	"github.com/google/uuid"
)

type workspaceAccessRepo struct {
	workspace *domain.Workspace
	err       error
}

func (r workspaceAccessRepo) FindByID(context.Context, uuid.UUID) (*domain.Workspace, error) {
	return r.workspace, r.err
}

func (workspaceAccessRepo) ListForUser(context.Context, uuid.UUID) ([]domain.Workspace, error) {
	return nil, nil
}

func (workspaceAccessRepo) Save(context.Context, *domain.Workspace) error { return nil }

func TestTenantMiddleware(t *testing.T) {
	validUUID := uuid.New()

	tests := []struct {
		name            string
		workspaceHeader string
		expectedStatus  int
		expectID        bool
	}{
		{
			name:            "valid workspace ID",
			workspaceHeader: validUUID.String(),
			expectedStatus:  http.StatusOK,
			expectID:        true,
		},
		{
			name:            "missing workspace ID",
			workspaceHeader: "",
			expectedStatus:  http.StatusBadRequest,
			expectID:        false,
		},
		{
			name:            "invalid workspace ID format",
			workspaceHeader: "not-a-uuid",
			expectedStatus:  http.StatusBadRequest,
			expectID:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, ok := WorkspaceIDFromContext(r.Context())
				if tt.expectID {
					if !ok {
						t.Errorf("expected workspace ID in context, but not found")
					}
					if id != validUUID {
						t.Errorf("expected workspace ID %v, got %v", validUUID, id)
					}
				} else if ok {
					t.Errorf("did not expect workspace ID in context, but found %v", id)
				}
				w.WriteHeader(http.StatusOK)
			})

			handler := TenantMiddleware(nextHandler)

			req := httptest.NewRequest("GET", "/", nil)
			if tt.workspaceHeader != "" {
				req.Header.Set("X-Workspace-ID", tt.workspaceHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestWorkspaceAccessMiddlewareRequiresOwnership(t *testing.T) {
	ownerID := uuid.New()
	workspaceID := uuid.New()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	tests := []struct {
		name      string
		userID    uuid.UUID
		workspace *domain.Workspace
		repoErr   error
		expected  int
	}{
		{name: "owner is allowed", userID: ownerID, workspace: &domain.Workspace{ID: workspaceID, OwnerID: ownerID}, expected: http.StatusNoContent},
		{name: "different user is forbidden", userID: uuid.New(), workspace: &domain.Workspace{ID: workspaceID, OwnerID: ownerID}, expected: http.StatusForbidden},
		{name: "unknown workspace is forbidden", userID: ownerID, repoErr: context.Canceled, expected: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx := context.WithValue(req.Context(), domain.ContextKeyUserID, tt.userID)
			ctx = context.WithValue(ctx, domain.ContextKeyWorkspaceID, workspaceID)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			WorkspaceAccessMiddleware(workspaceAccessRepo{workspace: tt.workspace, err: tt.repoErr})(next).ServeHTTP(rr, req)
			if rr.Code != tt.expected {
				t.Fatalf("expected status %d, got %d", tt.expected, rr.Code)
			}
		})
	}
}
