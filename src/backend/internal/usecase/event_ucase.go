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
var eventMetadataToken = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]{0,127}$`)
var forbiddenEventMetadataKeys = map[string]struct{}{
	"authorization": {}, "body": {}, "candidate": {}, "comment": {},
	"content": {}, "credential": {}, "ice": {}, "password": {},
	"sdp": {}, "secret": {}, "text": {}, "token": {},
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

func safeEventValue(value any) bool {
	switch current := value.(type) {
	case nil, bool, string, float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, json.Number:
		return true
	case []any:
		for _, item := range current {
			if !safeEventValue(item) {
				return false
			}
		}
		return true
	case map[string]any:
		for key, item := range current {
			if _, denied := forbiddenEventMetadataKeys[strings.ToLower(key)]; denied || !eventMetadataToken.MatchString(key) || !safeEventValue(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
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
	if !safeEventValue(input.SafeMetadata) {
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
			if !workspace.Owner && strings.HasPrefix(event.Kind, "security.") {
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
