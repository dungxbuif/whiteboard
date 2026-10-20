package collab

import (
	"bytes"
	"devtool/internal/domain"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func (c *AgentClient) presentationDecision(ctx domain.Context, payload map[string]any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.origin+"/internal/presentations/resolve", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	req.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(req)
	if err != nil {
		return &domain.AgentCollabError{Status: 503, Code: "UNAVAILABLE"}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		code := "UNAVAILABLE"
		if response.StatusCode == http.StatusConflict || response.StatusCode == http.StatusNotFound {
			code = "PRESENTATION_ENDED"
		}
		return &domain.AgentCollabError{Status: response.StatusCode, Code: code}
	}
	if err = json.NewDecoder(response.Body).Decode(out); err != nil {
		return &domain.AgentCollabError{Status: 503, Code: "UNAVAILABLE"}
	}
	return nil
}
func (c *AgentClient) ResolveHost(ctx domain.Context, ref domain.SceneRef, sessionID uuid.UUID, principalID string) (domain.PresentationRuntimeDecision, error) {
	var out struct {
		Active   bool  `json:"active"`
		Revision int64 `json:"revision"`
	}
	err := c.presentationDecision(ctx, map[string]any{"kind": "host", "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "sessionId": sessionID.String(), "principalId": principalID}, &out)
	if err != nil {
		return domain.PresentationRuntimeDecision{}, err
	}
	if !out.Active {
		return domain.PresentationRuntimeDecision{}, domain.ErrPresentationEnded
	}
	if out.Revision < 1 {
		return domain.PresentationRuntimeDecision{}, domain.ErrInvalidPresentation
	}
	return domain.PresentationRuntimeDecision{SessionID: sessionID, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, HostPrincipalID: principalID, Revision: out.Revision}, nil
}
func (c *AgentClient) ResolvePending(ctx domain.Context, ref domain.SceneRef, sessionID uuid.UUID, hostPrincipalID, connectionID string, revision int64) (domain.PresentationRuntimeDecision, error) {
	var out struct {
		SessionID   string `json:"sessionId"`
		WorkspaceID string `json:"workspaceId"`
		SceneID     string `json:"sceneId"`
		PrincipalID string `json:"principalId"`
		DisplayName string `json:"displayName"`
	}
	err := c.presentationDecision(ctx, map[string]any{"kind": "pending", "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "sessionId": sessionID.String(), "principalId": hostPrincipalID, "connectionId": connectionID, "expectedRevision": revision}, &out)
	if err != nil {
		return domain.PresentationRuntimeDecision{}, err
	}
	sid, se := uuid.Parse(out.SessionID)
	wid, we := uuid.Parse(out.WorkspaceID)
	scene, ce := uuid.Parse(out.SceneID)
	if se != nil || we != nil || ce != nil || sid != sessionID || wid != ref.WorkspaceID || scene != ref.SceneID || out.PrincipalID == "" {
		return domain.PresentationRuntimeDecision{}, domain.ErrInvalidPresentation
	}
	return domain.PresentationRuntimeDecision{SessionID: sid, WorkspaceID: wid, SceneID: scene, PrincipalID: out.PrincipalID, DisplayName: out.DisplayName}, nil
}
func (c *AgentClient) ResolveAudience(ctx domain.Context, ref domain.SceneRef, sessionID uuid.UUID, principalID string) (domain.PresentationRuntimeDecision, error) {
	var out struct {
		Active          bool   `json:"active"`
		HostPrincipalID string `json:"hostPrincipalId"`
	}
	err := c.presentationDecision(ctx, map[string]any{"kind": "audience", "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "sessionId": sessionID.String(), "principalId": principalID}, &out)
	if err != nil {
		return domain.PresentationRuntimeDecision{}, err
	}
	if !out.Active {
		return domain.PresentationRuntimeDecision{}, domain.ErrPresentationEnded
	}
	if host, e := uuid.Parse(out.HostPrincipalID); e != nil || host == uuid.Nil {
		return domain.PresentationRuntimeDecision{}, domain.ErrInvalidPresentation
	}
	return domain.PresentationRuntimeDecision{SessionID: sessionID, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, PrincipalID: principalID, HostPrincipalID: out.HostPrincipalID}, nil
}

var _ domain.PresentationRuntimeVerifier = (*AgentClient)(nil)
