package usecase

import (
	"devtool/internal/domain"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const maxActivityPageSize = 50

type activityAccess interface {
	ResolveWorkspace(domain.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error)
	Resolve(domain.Context, domain.Principal, domain.SceneRef) (domain.AccessDecision, error)
}

type ActivityService struct {
	repo     domain.ActivityRepository
	features domain.FeatureService
	access   activityAccess
}

func NewActivityService(repo domain.ActivityRepository, features domain.FeatureService, access activityAccess) *ActivityService {
	return &ActivityService{repo: repo, features: features, access: access}
}

type activityCursorEnvelope struct {
	OccurredAt string `json:"occurredAt"`
	ID         string `json:"id"`
}

func encodeActivityCursor(item domain.ActivityItem) string {
	payload, _ := json.Marshal(activityCursorEnvelope{OccurredAt: item.OccurredAt.UTC().Format(time.RFC3339Nano), ID: item.ID})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeActivityCursor(raw string) (*domain.ActivityCursor, error) {
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, domain.ErrInvalidActivity
	}
	var value activityCursorEnvelope
	if json.Unmarshal(payload, &value) != nil || value.OccurredAt == "" || value.ID == "" {
		return nil, domain.ErrInvalidActivity
	}
	when, err := time.Parse(time.RFC3339Nano, value.OccurredAt)
	id, idErr := strconv.ParseInt(value.ID, 10, 64)
	if err != nil || idErr != nil || id < 1 {
		return nil, domain.ErrInvalidActivity
	}
	canonical, _ := json.Marshal(activityCursorEnvelope{OccurredAt: when.UTC().Format(time.RFC3339Nano), ID: strconv.FormatInt(id, 10)})
	if base64.RawURLEncoding.EncodeToString(canonical) != raw {
		return nil, domain.ErrInvalidActivity
	}
	return &domain.ActivityCursor{OccurredAt: when.UTC(), ID: value.ID}, nil
}

func (s *ActivityService) List(ctx domain.Context, principal domain.Principal, workspaceID uuid.UUID, sceneID *uuid.UUID, cursor string, limit int) (domain.ActivityPage, error) {
	page := domain.ActivityPage{Items: []domain.ActivityItem{}}
	if principal.Kind != domain.PrincipalUser || principal.UserID == nil || *principal.UserID == uuid.Nil || principal.ID != principal.UserID.String() || workspaceID == uuid.Nil || sceneID != nil && *sceneID == uuid.Nil || limit < 1 || limit > maxActivityPageSize || s.repo == nil || s.features == nil || s.access == nil {
		return page, domain.ErrAccessDenied
	}
	if !principal.CredentialExpiresAt.IsZero() && !time.Now().Before(principal.CredentialExpiresAt) {
		return page, domain.ErrInvalidSceneSession
	}
	decoded, err := decodeActivityCursor(cursor)
	if err != nil {
		return page, err
	}
	decision, err := s.access.ResolveWorkspace(ctx, principal, workspaceID)
	if err != nil {
		return page, err
	}
	policy, err := s.features.Policy(ctx, principal, workspaceID)
	if err != nil {
		return page, err
	}
	if !policy.ActivityEnabled {
		return page, domain.ErrFeatureDisabled
	}
	if sceneID != nil {
		if err = s.features.Require(ctx, principal, domain.SceneRef{WorkspaceID: workspaceID, SceneID: *sceneID}, domain.FeatureActivityRead); err != nil {
			return page, err
		}
	}
	rows, err := s.repo.ListActivity(ctx, *principal.UserID, workspaceID, sceneID, decision.Owner, decoded, limit+1)
	if err != nil {
		return page, err
	}
	if len(rows) > limit {
		page.NextCursor = encodeActivityCursor(rows[limit-1])
		rows = rows[:limit]
	}
	page.Items = rows
	retainedSince, err := s.repo.ActivityRetainedSince(ctx, *principal.UserID, workspaceID, sceneID, decision.Owner)
	if !retainedSince.IsZero() {
		page.RetainedSince = &retainedSince
	}
	return page, err
}

var _ domain.ActivityUsecase = (*ActivityService)(nil)
