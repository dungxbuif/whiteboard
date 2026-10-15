package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestTenantMiddleware(t *testing.T) {
	validUUID := uuid.New()

	tests := []struct {
		name           string
		workspaceHeader string
		expectedStatus int
		expectID       bool
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
