package usecase

import (
	"context"
	"testing"

	"devtool/internal/domain"
)

type fakeAIRepository struct{ response string }

func (f fakeAIRepository) Generate(_ domain.Context, _ string, _ string, _ string) (string, error) { return f.response, nil }

func TestAIGenerateDiagramCleansFencedJSON(t *testing.T) {
	u := NewAIUsecase(fakeAIRepository{response: "```json\n{\"elements\":[{\"type\":\"rectangle\"}]}\n```"}, "fast", "reasoning")
	result, err := u.GenerateDiagram(context.Background(), domain.AIRequest{Prompt: "draw a box"})
	if err != nil { t.Fatalf("GenerateDiagram() error = %v", err) }
	content, ok := result.Content.(map[string]any)
	if !ok { t.Fatalf("content type = %T", result.Content) }
	if _, ok := content["elements"]; !ok { t.Fatalf("elements missing: %#v", content) }
	if _, ok := content["appState"]; !ok { t.Fatalf("appState missing: %#v", content) }
}

func TestAIPromptLimit(t *testing.T) {
	u := NewAIUsecase(fakeAIRepository{response: "ok"}, "fast", "reasoning")
	_, err := u.Chat(context.Background(), domain.AIRequest{Prompt: string(make([]byte, maxAIPromptBytes+1))})
	if err == nil { t.Fatal("expected prompt length error") }
}

func TestAIToCodeStripsFence(t *testing.T) {
	u := NewAIUsecase(fakeAIRepository{response: "```tsx\nexport default function Diagram() {}\n```"}, "fast", "reasoning")
	result, err := u.ToCode(context.Background(), domain.AIRequest{Prompt: "make it responsive", Scene: []byte(`{"elements":[]}`)})
	if err != nil { t.Fatalf("ToCode() error = %v", err) }
	if result.Content != "export default function Diagram() {}" { t.Fatalf("unexpected code: %v", result.Content) }
}
