package domain

import "encoding/json"

const (
	AIKindChat    = "chat"
	AIKindDiagram = "diagram"
	AIKindCode    = "code"
)

type AIRequest struct {
	Prompt       string          `json:"prompt"`
	Scene        json.RawMessage `json:"scene,omitempty"`
	ImageDataURL string          `json:"imageDataUrl,omitempty"`
}

type AIResponse struct {
	Kind    string `json:"kind"`
	Content any    `json:"content"`
	Model   string `json:"model,omitempty"`
}

type AIRepository interface {
	Generate(ctx Context, prompt string, imageDataURL string, model string) (string, error)
}

type AIUsecase interface {
	Chat(ctx Context, request AIRequest) (*AIResponse, error)
	GenerateDiagram(ctx Context, request AIRequest) (*AIResponse, error)
	ToCode(ctx Context, request AIRequest) (*AIResponse, error)
}
