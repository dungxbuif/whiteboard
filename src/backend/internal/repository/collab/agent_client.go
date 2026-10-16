package collab

import (
	"bytes"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const requestTimeout = 8 * time.Second
const maxErrorBytes = 4096

var allowedCodes = map[string]bool{"INVALID_UPDATE": true, "UNAUTHENTICATED": true, "FORBIDDEN": true, "STALE_AUTHORITY": true, "CONFLICT": true, "REQUEST_ID_REUSED": true, "RATE_LIMITED": true, "UNAVAILABLE": true, "COLLAB_DISABLED": true, "PROMOTION_REQUIRED": true, "SNAPSHOT_HISTORY_LIMIT": true, "PROJECTION_TOO_LARGE": true}

type AgentClient struct {
	origin string
	secret string
	client *http.Client
}

func NewAgentClient(origin, secret string) (*AgentClient, error) {
	u, e := url.Parse(origin)
	if e != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || len(secret) < 32 {
		return nil, errors.New("invalid private collaboration origin/secret")
	}
	return &AgentClient{strings.TrimSuffix(origin, "/"), secret, &http.Client{Timeout: requestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *AgentClient) Call(ctx domain.Context, operation string, ref domain.SceneRef, envelope map[string]any, limit int64) (json.RawMessage, error) {
	if operation != "commands" && operation != "projection" && operation != "activity" && operation != "export-projection" {
		return nil, domain.ErrAgentInput
	}
	payload, e := json.Marshal(envelope)
	if e != nil {
		return nil, e
	}
	req, e := http.NewRequestWithContext(ctx, "POST", c.origin+"/internal/scenes/"+ref.SceneID.String()+"/"+operation, bytes.NewReader(payload))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	req.Header.Set("Content-Type", "application/json")
	response, e := c.client.Do(req)
	if e != nil {
		return nil, &domain.AgentCollabError{Status: 503, Code: "UNAVAILABLE"}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		var fault struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, maxErrorBytes)).Decode(&fault)
		if !allowedCodes[fault.Code] {
			return nil, &domain.AgentCollabError{Status: 503, Code: "UNAVAILABLE"}
		}
		return nil, &domain.AgentCollabError{Status: response.StatusCode, Code: fault.Code}
	}
	result, e := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if e != nil {
		return nil, &domain.AgentCollabError{Status: 503, Code: "UNAVAILABLE"}
	}
	if int64(len(result)) > limit {
		return nil, domain.ErrAgentProjection
	}
	if !json.Valid(result) {
		return nil, &domain.AgentCollabError{Status: 503, Code: "UNAVAILABLE"}
	}
	return json.RawMessage(result), nil
}
