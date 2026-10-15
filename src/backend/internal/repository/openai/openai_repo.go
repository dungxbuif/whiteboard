package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Repository talks to any provider implementing the OpenAI chat completions
// contract. DeepSeek is the default deployment target, but the base URL and
// models are configurable for other compatible providers.
type Repository struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

type requestBody struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type responseBody struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func New(apiKey, baseURL, model string) *Repository {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	return &Repository{
		apiKey: apiKey, baseURL: strings.TrimRight(baseURL, "/"), model: model,
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (r *Repository) Generate(ctx context.Context, prompt, imageDataURL, model string) (string, error) {
	if strings.TrimSpace(r.apiKey) == "" {
		return "", fmt.Errorf("AI_API_KEY is not configured")
	}
	if model == "" {
		model = r.model
	}
	parts := []contentPart{{Type: "text", Text: prompt}}
	if imageDataURL != "" {
		if _, _, err := parseDataURL(imageDataURL); err != nil {
			return "", err
		}
		parts = append(parts, contentPart{Type: "image_url", ImageURL: &imageURL{URL: imageDataURL}})
	}
	body, err := json.Marshal(requestBody{Model: model, Messages: []message{{Role: "user", Content: parts}}})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	responseBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("AI provider request failed with status %d", resp.StatusCode)
	}
	var parsed responseBody
	if err := json.Unmarshal(responseBytes, &parsed); err != nil || len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("AI provider returned an empty response")
	}
	return parsed.Choices[0].Message.Content, nil
}

func parseDataURL(value string) (string, string, error) {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "data:") {
		return "", "", fmt.Errorf("imageDataUrl must be a data URL")
	}
	metaParts := strings.Split(strings.TrimPrefix(parts[0], "data:"), ";")
	if len(metaParts) < 2 || metaParts[len(metaParts)-1] != "base64" {
		return "", "", fmt.Errorf("imageDataUrl must use base64 encoding")
	}
	if !strings.HasPrefix(metaParts[0], "image/") {
		return "", "", fmt.Errorf("imageDataUrl must contain an image MIME type")
	}
	if len(parts[1]) > 5<<20 {
		return "", "", fmt.Errorf("imageDataUrl is too large")
	}
	if _, err := base64.StdEncoding.DecodeString(parts[1]); err != nil {
		return "", "", fmt.Errorf("imageDataUrl contains invalid base64")
	}
	return metaParts[0], parts[1], nil
}
