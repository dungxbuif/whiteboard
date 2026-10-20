package handler

import (
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

type missingParentFolderUsecase struct{ domain.FolderUsecase }

func (missingParentFolderUsecase) Update(context.Context, uuid.UUID, uuid.UUID, *string, **uuid.UUID) error {
	return domain.ErrInaccessible
}

func TestFolderHandler_UpdateMissingParent(t *testing.T) {
	folderID, parentID := uuid.New(), uuid.New()
	router := chi.NewRouter()
	router.Patch("/folders/{id}", NewFolderHandler(missingParentFolderUsecase{}).Update)
	request := httptest.NewRequest(http.MethodPatch, "/folders/"+folderID.String(), strings.NewReader(`{"newParentId":"`+parentID.String()+`"}`))
	request = request.WithContext(context.WithValue(request.Context(), domain.ContextKeyWorkspaceID, uuid.New()))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Code != "NOT_FOUND" {
		t.Fatalf("body=%s error=%v", response.Body.String(), err)
	}
}
