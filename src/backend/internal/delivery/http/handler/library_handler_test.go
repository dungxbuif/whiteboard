package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLibraryImportRejectsUnknownFieldsAndOversize(t *testing.T) {
	for _, body := range []string{`{"requestId":"11111111-1111-4111-8111-111111111111","name":"Flow","content":{},"credential":"secret"}`, strings.Repeat("x", 25*1024*1024+1)} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		w := httptest.NewRecorder()
		var value struct {
			RequestID string `json:"requestId"`
			Name      string `json:"name"`
			Content   any    `json:"content"`
		}
		if decodeLibraryRequest(w, r, &value) == nil {
			t.Fatal("unsafe import body accepted")
		}
	}
}
