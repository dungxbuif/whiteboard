package usecase

import (
	"encoding/json"
	"fmt"
	"strings"

	"devtool/internal/domain"
)

const maxAIPromptBytes = 12000

type aiUsecase struct {
	repo           domain.AIRepository
	fastModel      string
	reasoningModel string
}

func NewAIUsecase(repo domain.AIRepository, fastModel, reasoningModel string) domain.AIUsecase {
	return &aiUsecase{repo: repo, fastModel: fastModel, reasoningModel: reasoningModel}
}

func (u *aiUsecase) Chat(ctx domain.Context, request domain.AIRequest) (*domain.AIResponse, error) {
	prompt, err := cleanPrompt(request.Prompt)
	if err != nil {
		return nil, err
	}
	text, err := u.repo.Generate(ctx, prompt, request.ImageDataURL, u.fastModel)
	if err != nil {
		return nil, err
	}
	return &domain.AIResponse{Kind: domain.AIKindChat, Content: text, Model: u.fastModel}, nil
}

func (u *aiUsecase) GenerateDiagram(ctx domain.Context, request domain.AIRequest) (*domain.AIResponse, error) {
	prompt, err := cleanPrompt("Return only valid JSON for an Excalidraw scene with elements, appState, and files. User request: " + request.Prompt)
	if err != nil {
		return nil, err
	}
	text, err := u.repo.Generate(ctx, prompt, request.ImageDataURL, u.fastModel)
	if err != nil {
		return nil, err
	}
	content, err := parseSceneJSON(text)
	if err != nil {
		return nil, err
	}
	return &domain.AIResponse{Kind: domain.AIKindDiagram, Content: content, Model: u.fastModel}, nil
}

func (u *aiUsecase) ToCode(ctx domain.Context, request domain.AIRequest) (*domain.AIResponse, error) {
	scene, err := sceneDescription(request.Scene)
	if err != nil {
		return nil, err
	}
	prompt, err := cleanPrompt("Generate a self-contained React component that reflects this Excalidraw layout. Return code only. Layout: " + scene + "\nUser request: " + request.Prompt)
	if err != nil {
		return nil, err
	}
	text, err := u.repo.Generate(ctx, prompt, request.ImageDataURL, u.reasoningModel)
	if err != nil {
		return nil, err
	}
	return &domain.AIResponse{Kind: domain.AIKindCode, Content: stripCodeFence(text), Model: u.reasoningModel}, nil
}

func cleanPrompt(prompt string) (string, error) {
	value := strings.TrimSpace(prompt)
	if value == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if len([]byte(value)) > maxAIPromptBytes {
		return "", fmt.Errorf("prompt must be at most %d bytes", maxAIPromptBytes)
	}
	return value, nil
}

func stripCodeFence(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "```tsx")
	value = strings.TrimPrefix(value, "```jsx")
	value = strings.TrimPrefix(value, "```html")
	value = strings.TrimPrefix(value, "```")
	value = strings.TrimSuffix(value, "```")
	return strings.TrimSpace(value)
}

func parseSceneJSON(value string) (map[string]any, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "```json")
	value = strings.TrimPrefix(value, "```")
	value = strings.TrimSuffix(value, "```")
	value = strings.TrimSpace(value)
	start, end := strings.Index(value, "{"), strings.LastIndex(value, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("model did not return a JSON object")
	}
	var scene map[string]any
	if err := json.Unmarshal([]byte(value[start:end+1]), &scene); err != nil {
		return nil, fmt.Errorf("invalid diagram JSON: %w", err)
	}
	if _, ok := scene["elements"].([]any); !ok {
		return nil, fmt.Errorf("diagram JSON must include an elements array")
	}
	if _, ok := scene["appState"].(map[string]any); !ok {
		scene["appState"] = map[string]any{}
	}
	if _, ok := scene["files"].(map[string]any); !ok {
		scene["files"] = map[string]any{}
	}
	return scene, nil
}

func sceneDescription(scene json.RawMessage) (string, error) {
	if len(scene) == 0 {
		return "", fmt.Errorf("scene is required")
	}
	var value map[string]any
	if err := json.Unmarshal(scene, &value); err != nil {
		return "", fmt.Errorf("scene must be valid JSON")
	}
	data, _ := json.Marshal(value)
	return string(data), nil
}
