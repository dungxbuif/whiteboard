package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseDataURL(t *testing.T) {
	mime, data, err := parseDataURL("data:image/png;base64,aGVsbG8=")
	if err != nil {
		t.Fatalf("parseDataURL() error = %v", err)
	}
	if mime != "image/png" || data != "aGVsbG8=" {
		t.Fatalf("unexpected result: %s %s", mime, data)
	}
}

func TestParseDataURLRejectsNonBase64(t *testing.T) {
	if _, _, err := parseDataURL("data:image/png,hello"); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateUsesChatCompletionsContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header missing")
		}
		var body requestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "deepseek-chat" || len(body.Messages) != 1 {
			t.Fatalf("unexpected request: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	repo := New("test-key", server.URL+"/v1", "deepseek-chat")
	result, err := repo.Generate(context.Background(), "hello", "", "")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result != "ok" {
		t.Fatalf("result = %q", result)
	}
}
