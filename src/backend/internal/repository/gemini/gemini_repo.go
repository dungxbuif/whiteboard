package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Repository struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

type requestBody struct {
	Contents []content `json:"contents"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inlineData,omitempty"`
}

type inlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type responseBody struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func New(apiKey, model string) *Repository {
	if model == "" {
		model = "gemini-1.5-flash"
	}
	return &Repository{apiKey: apiKey, model: model, baseURL: "https://generativelanguage.googleapis.com/v1beta", httpClient: &http.Client{Timeout: 45 * time.Second}}
}

func (r *Repository) Generate(ctx context.Context, prompt, imageDataURL, model string) (string, error) {
	if strings.TrimSpace(r.apiKey) == "" {
		return "", fmt.Errorf("GEMINI_API_KEY is not configured")
	}
	if model == "" {
		model = r.model
	}
	parts := []part{{Text: prompt}}
	if imageDataURL != "" {
		mimeType, data, err := parseDataURL(imageDataURL)
		if err != nil {
			return "", err
		}
		parts = append(parts, part{InlineData: &inlineData{MimeType: mimeType, Data: data}})
	}
	body, err := json.Marshal(requestBody{Contents: []content{{Parts: parts}}})
	if err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", strings.TrimRight(r.baseURL, "/"), url.PathEscape(model), url.QueryEscape(r.apiKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	responseBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini request failed with status %d", resp.StatusCode)
	}
	var parsed responseBody
	if err := json.Unmarshal(responseBytes, &parsed); err != nil || len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned an empty response")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

func parseDataURL(value string) (string, string, error) {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "data:") {
		return "", "", fmt.Errorf("imageDataUrl must be a data URL")
	}
	meta := strings.TrimPrefix(parts[0], "data:")
	metaParts := strings.Split(meta, ";")
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
