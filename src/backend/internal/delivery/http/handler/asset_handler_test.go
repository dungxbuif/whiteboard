package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"devtool/internal/domain"
)

type mockAssetUsecase struct {
	domain.AssetUsecase
}

func (m *mockAssetUsecase) PresignUpload(ctx domain.Context, workspaceID, userID, sceneID uuid.UUID, assets []domain.AssetUploadRequest) (map[string]domain.AssetUploadResponse, error) {
	result := make(map[string]domain.AssetUploadResponse)
	for _, a := range assets {
		result[a.FileID] = domain.AssetUploadResponse{
			AssetID:    uuid.New(),
			UploadUrl:  "https://mock.upload/" + a.FileID,
			StorageKey: "mock-key",
			ExpiresAt:  time.Now().Add(time.Hour),
		}
	}
	return result, nil
}

func (m *mockAssetUsecase) PresignView(ctx domain.Context, workspaceID, sceneID uuid.UUID, fileIDs []string) (map[string]domain.AssetViewResponse, error) {
	result := make(map[string]domain.AssetViewResponse)
	for _, fid := range fileIDs {
		result[fid] = domain.AssetViewResponse{
			AssetID:   uuid.New(),
			ViewUrl:   "https://mock.view/" + fid,
			MimeType:  "image/png",
			ExpiresAt: time.Now().Add(time.Hour),
		}
	}
	return result, nil
}

func TestAssetHandler_PresignUpload(t *testing.T) {
	ucase := &mockAssetUsecase{}
	h := NewAssetHandler(ucase)

	workspaceID := uuid.New()
	userID := uuid.New()
	sceneID := uuid.New()

	body, _ := json.Marshal(map[string]interface{}{
		"sceneId": sceneID.String(),
		"assets": []map[string]interface{}{
			{"fileId": "file1", "mimeType": "image/png", "byteSize": 100},
		},
	})
	req := httptest.NewRequest("POST", "/assets/presign-upload", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-ID", workspaceID.String())

	ctx := context.WithValue(req.Context(), domain.ContextKeyUserID, userID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.PresignUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var res map[string]map[string]domain.AssetUploadResponse
	json.Unmarshal(rr.Body.Bytes(), &res)

	urls, ok := res["urls"]
	if !ok || len(urls) != 1 {
		t.Fatalf("expected 1 url, got %v", urls)
	}
	if urls["file1"].UploadUrl != "https://mock.upload/file1" {
		t.Errorf("unexpected upload url: %s", urls["file1"].UploadUrl)
	}
}

func TestAssetHandler_PresignView(t *testing.T) {
	ucase := &mockAssetUsecase{}
	h := NewAssetHandler(ucase)

	workspaceID := uuid.New()
	userID := uuid.New()
	sceneID := uuid.New()

	body, _ := json.Marshal(map[string]interface{}{
		"sceneId": sceneID.String(),
		"fileIds": []string{"file1", "file2"},
	})
	req := httptest.NewRequest("POST", "/assets/presign-view", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-ID", workspaceID.String())

	ctx := context.WithValue(req.Context(), domain.ContextKeyUserID, userID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.PresignView(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var res map[string]map[string]domain.AssetViewResponse
	json.Unmarshal(rr.Body.Bytes(), &res)

	urls, ok := res["urls"]
	if !ok || len(urls) != 2 {
		t.Fatalf("expected 2 urls, got %v", urls)
	}
	if urls["file1"].ViewUrl != "https://mock.view/file1" {
		t.Errorf("unexpected view url: %s", urls["file1"].ViewUrl)
	}
}
