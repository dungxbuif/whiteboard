package usecase

import (
	"devtool/internal/domain"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const (
	maxEventMetadataBytes = 4096
	maxEventPageSize      = 50
	maxEventScanPageSize  = 100
)

var eventToken = regexp.MustCompile(`^[a-z0-9][a-z0-9_.:-]{0,127}$`)
var eventMetadataValueToken = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)
var eventMetadataUUIDKeys = map[string]struct{}{
	"threadId": {}, "commentId": {}, "versionId": {}, "collectionId": {}, "libraryItemId": {}, "requestId": {},
}
var eventMetadataDecimalKeys = map[string]struct{}{
	"revision": {}, "sequence": {}, "aclVersion": {}, "latencyMs": {}, "count": {},
}
var eventMetadataTokenKeys = map[string]struct{}{
	"action": {}, "eventType": {}, "tool": {}, "scope": {}, "outcome": {}, "failureClass": {}, "status": {}, "role": {},
}

type eventAccess interface {
	ResolveWorkspace(domain.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error)
	Resolve(domain.Context, domain.Principal, domain.SceneRef) (domain.AccessDecision, error)
}

type EventService struct {
	repo   domain.EventRepository
	access eventAccess
}

func NewEventService(repo domain.EventRepository, access eventAccess) *EventService {
	return &EventService{repo: repo, access: access}
}

func safeEventMetadata(metadata map[string]any) bool {
	for key, value := range metadata {
		if _, ok := eventMetadataUUIDKeys[key]; ok {
			text, stringOK := value.(string)
			if !stringOK || uuid.Validate(text) != nil {
				return false
			}
			continue
		}
		if _, ok := eventMetadataDecimalKeys[key]; ok {
			switch number := value.(type) {
			case string:
				if !decimalCursor(number) {
					return false
				}
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
			default:
				return false
			}
			continue
		}
		if _, ok := eventMetadataTokenKeys[key]; ok {
			text, stringOK := value.(string)
			if !stringOK || !eventMetadataValueToken.MatchString(text) || strings.Contains(strings.ToLower(text), "secret") {
				return false
			}
			continue
		}
		return false
	}
	return true
}

func validateEventInput(input domain.EventInput) error {
	if input.WorkspaceID == uuid.Nil || input.ActorID == "" || !eventToken.MatchString(input.Kind) || !eventToken.MatchString(input.Source) || input.SourceKey == "" || len(input.SourceKey) > 256 {
		return domain.ErrEventInput
	}
	if input.SceneID != nil && *input.SceneID == uuid.Nil {
		return domain.ErrEventInput
	}
	switch input.ActorKind {
	case domain.EventActorUser, domain.EventActorPublic, domain.EventActorAgent, domain.EventActorSystem:
	default:
		return domain.ErrEventInput
	}
	if input.SafeMetadata == nil {
		input.SafeMetadata = map[string]any{}
	}
	if !safeEventMetadata(input.SafeMetadata) {
		return domain.ErrUnsafeEventMetadata
	}
	encoded, err := json.Marshal(input.SafeMetadata)
	if err != nil || len(encoded) > maxEventMetadataBytes {
		return domain.ErrUnsafeEventMetadata
	}
	return nil
}

func (s *EventService) AppendEvent(ctx domain.Context, input domain.EventInput) (domain.WorkspaceEvent, error) {
	if err := validateEventInput(input); err != nil {
		return domain.WorkspaceEvent{}, err
	}
	return s.repo.AppendEvent(ctx, input)
}

func (s *EventService) ProjectAuditBatch(ctx domain.Context, limit int) (int, error) {
	if limit < 1 || limit > maxEventScanPageSize {
		return 0, domain.ErrEventInput
	}
	return s.repo.ProjectAuditBatch(ctx, limit)
}

func (s *EventService) ListWorkspace(ctx domain.Context, principal domain.Principal, workspaceID uuid.UUID, cursor string, limit int) (domain.WorkspaceEventPage, error) {
	if principal.Kind != domain.PrincipalUser || principal.UserID == nil || limit < 1 || limit > maxEventPageSize || (cursor != "" && !decimalCursor(cursor)) {
		return domain.WorkspaceEventPage{}, domain.ErrAccessDenied
	}
	workspace, err := s.access.ResolveWorkspace(ctx, principal, workspaceID)
	if err != nil {
		return domain.WorkspaceEventPage{}, err
	}
	items := make([]domain.WorkspaceEvent, 0, limit)
	scanCursor, nextCursor := cursor, ""
	for len(items) < limit {
		rows, listErr := s.repo.ListWorkspaceEvents(ctx, workspaceID, scanCursor, maxEventScanPageSize)
		if listErr != nil {
			return domain.WorkspaceEventPage{}, listErr
		}
		if len(rows) == 0 {
			break
		}
		for _, event := range rows {
			scanCursor = event.ID
			if !workspace.Owner && (strings.HasPrefix(event.Kind, "security.") || strings.HasPrefix(event.Kind, "acl.")) {
				continue
			}
			if event.SceneID != nil {
				if _, resolveErr := s.access.Resolve(ctx, principal, domain.SceneRef{WorkspaceID: workspaceID, SceneID: *event.SceneID}); resolveErr != nil {
					continue
				}
			}
			items = append(items, event)
			nextCursor = event.ID
			if len(items) == limit {
				break
			}
		}
		if len(rows) < maxEventScanPageSize || len(items) == limit {
			break
		}
	}
	return domain.WorkspaceEventPage{Items: items, NextCursor: nextCursor}, nil
}

func decimalCursor(value string) bool {
	if value == "" {
		return true
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return value != "0"
}
