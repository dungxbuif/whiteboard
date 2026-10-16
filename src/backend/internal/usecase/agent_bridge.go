package usecase

import (
	"bytes"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"io"
	"strconv"
	"unicode/utf8"
)

const agentCommandBytes = 1024 * 1024
const agentProjectionBytes = 256 * 1024
const agentExportBytes = 26 * 1024 * 1024

// AgentBridge signs local capability decisions, then releases their transactions
// before invoking the single durable Node writer.
type AgentBridge struct {
	agents *AgentService
	rooms  *RoomService
	caps   domain.CapabilityService
	remote domain.AgentCollabRepository
}

func NewAgentBridge(a *AgentService, r *RoomService, c domain.CapabilityService, remote domain.AgentCollabRepository) *AgentBridge {
	return &AgentBridge{a, r, c, remote}
}
func (b *AgentBridge) grant(ctx domain.Context, p domain.Principal, ref domain.SceneRef, action, id, hash, sequence string) (domain.SignedGrant, error) {
	room, e := b.rooms.MintRoomToken(ctx, p, ref)
	if e != nil {
		return domain.SignedGrant{}, e
	}
	identity, e := b.rooms.VerifyRoomToken(room.Token)
	if e != nil {
		return domain.SignedGrant{}, e
	}
	return b.caps.AuthorizeCommand(ctx, identity.Principal, domain.SceneMutation{WorkspaceID: ref.WorkspaceID.String(), SceneID: ref.SceneID.String(), Action: action, RequestID: id, PayloadHash: hash, ExpectedSequence: sequence})
}

type AgentCommandInput struct {
	MutationBase64 string `json:"mutationBase64" validate:"required"`
	Tool           string `json:"tool" validate:"required,oneof=generate_diagram patch_elements delete_elements"`
}

func (b *AgentBridge) Commands(ctx domain.Context, p domain.Principal, ref domain.SceneRef, in AgentCommandInput) (json.RawMessage, error) {
	result, e := b.agents.RunAuditedRemoteOperation(ctx, p, in.Tool, ref, func(ctx domain.Context) (any, error) {
		if validator.New().Struct(in) != nil || len(in.MutationBase64) > base64.StdEncoding.EncodedLen(agentCommandBytes) {
			return nil, domain.ErrAgentInput
		}
		raw, e := base64.StdEncoding.Strict().DecodeString(in.MutationBase64)
		if e != nil || base64.StdEncoding.EncodeToString(raw) != in.MutationBase64 || len(raw) > agentCommandBytes || !utf8.Valid(raw) {
			return nil, domain.ErrAgentInput
		}
		var command struct {
			WorkspaceID      string `json:"workspaceId"`
			SceneID          string `json:"sceneId"`
			RequestID        string `json:"requestId"`
			ExpectedSequence string `json:"expectedSequence"`
			Commands         []struct {
				Op       string          `json:"op"`
				Elements json.RawMessage `json:"elements,omitempty"`
				Patches  json.RawMessage `json:"patches,omitempty"`
				IDs      json.RawMessage `json:"ids,omitempty"`
			} `json:"commands"`
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&command) != nil || decoder.Decode(new(any)) != io.EOF || command.WorkspaceID != ref.WorkspaceID.String() || command.SceneID != ref.SceneID.String() || len(command.Commands) != 1 || command.ExpectedSequence == "" {
			return nil, domain.ErrAgentInput
		}
		op := map[string]string{"generate_diagram": "add", "patch_elements": "patch", "delete_elements": "delete"}[in.Tool]
		if command.Commands[0].Op != op {
			return nil, domain.ErrAgentInput
		}
		digest := sha256.Sum256(raw)
		grant, e := b.grant(ctx, p, ref, domain.ActionSceneCommand, command.RequestID, hex.EncodeToString(digest[:]), command.ExpectedSequence)
		if e != nil {
			return nil, e
		}
		return b.remote.Call(ctx, "commands", ref, map[string]any{"grant": grant.Token, "mutationBase64": in.MutationBase64}, agentProjectionBytes)
	})
	if e != nil {
		return nil, e
	}
	return result.(json.RawMessage), nil
}
func (b *AgentBridge) Projection(ctx domain.Context, p domain.Principal, ref domain.SceneRef, exporting bool) (json.RawMessage, error) {
	tool, action, operation, limit := "get_scene", domain.ActionSceneRead, "projection", int64(agentProjectionBytes)
	if exporting {
		tool, action, operation, limit = "export_scene", domain.ActionSceneExport, "export-projection", agentExportBytes
	}
	return b.read(ctx, p, ref, tool, action, operation, map[string]any{}, limit)
}
func (b *AgentBridge) Activity(ctx domain.Context, p domain.Principal, ref domain.SceneRef, cursor string, limit int) (json.RawMessage, error) {
	if cursor == "" {
		cursor = "0"
	}
	n, e := strconv.ParseInt(cursor, 10, 64)
	if e != nil || n < 0 || strconv.FormatInt(n, 10) != cursor || limit < 1 || limit > 50 {
		return nil, domain.ErrAgentInput
	}
	return b.read(ctx, p, ref, "get_scene_activity", domain.ActionSceneRead, "activity", map[string]any{"cursor": cursor, "limit": limit}, agentProjectionBytes)
}
func (b *AgentBridge) read(ctx domain.Context, p domain.Principal, ref domain.SceneRef, tool, action, operation string, envelope map[string]any, limit int64) (json.RawMessage, error) {
	result, e := b.agents.RunAuditedRemoteOperation(ctx, p, tool, ref, func(ctx domain.Context) (any, error) {
		digest := sha256.Sum256(nil)
		grant, e := b.grant(ctx, p, ref, action, uuid.NewString(), hex.EncodeToString(digest[:]), "")
		if e != nil {
			return nil, e
		}
		envelope["grant"], envelope["workspaceId"], envelope["sceneId"] = grant.Token, ref.WorkspaceID.String(), ref.SceneID.String()
		return b.remote.Call(ctx, operation, ref, envelope, limit)
	})
	if e != nil {
		return nil, e
	}
	return result.(json.RawMessage), nil
}
