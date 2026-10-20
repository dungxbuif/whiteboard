package usecase

import (
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

type retentionAccess interface {
	ResolveWorkspace(domain.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error)
	WithWorkspaceLock(domain.Context, uuid.UUID, uuid.UUID, func(domain.Context) error) error
}
type retentionFeatures interface {
	Policy(domain.Context, domain.Principal, uuid.UUID) (domain.WorkspaceFeaturePolicy, error)
}

type RetentionService struct {
	repo     domain.RetentionRepository
	access   retentionAccess
	features retentionFeatures
	now      func() time.Time
}

func NewRetentionService(repo domain.RetentionRepository, access retentionAccess, features retentionFeatures) *RetentionService {
	return &RetentionService{repo: repo, access: access, features: features, now: func() time.Time { return time.Now().UTC() }}
}

func validRetentionPolicy(input domain.RetentionPolicyInput) bool {
	if input.ActivityDays < 1 || input.ActivityDays > 3650 || input.NotificationDays < 1 || input.NotificationDays > 3650 || input.AutomaticVersionLimit < 1 || input.AutomaticVersionLimit > 10000 || input.TrashHardPurge {
		return false
	}
	validDays := func(value *int) bool { return value == nil || *value >= 1 && *value <= 3650 }
	return validDays(input.ManualVersionDays) && validDays(input.CommentsDays)
}
func retentionPrincipal(p domain.Principal) (uuid.UUID, error) {
	if p.Kind != domain.PrincipalUser || p.UserID == nil || *p.UserID == uuid.Nil || p.ID != p.UserID.String() {
		return uuid.Nil, domain.ErrAccessDenied
	}
	return *p.UserID, nil
}
func (s *RetentionService) authorize(ctx domain.Context, p domain.Principal, w uuid.UUID) (uuid.UUID, error) {
	u, err := retentionPrincipal(p)
	if err != nil || w == uuid.Nil || s.repo == nil || s.access == nil || s.features == nil {
		return uuid.Nil, domain.ErrAccessDenied
	}
	decision, err := s.access.ResolveWorkspace(ctx, p, w)
	if err != nil {
		return uuid.Nil, err
	}
	if !decision.Owner {
		return uuid.Nil, domain.ErrAccessDenied
	}
	policy, err := s.features.Policy(ctx, p, w)
	if err != nil {
		return uuid.Nil, err
	}
	if !policy.RetentionEnabled {
		return uuid.Nil, domain.ErrFeatureDisabled
	}
	return u, nil
}
func policyDigest(input domain.RetentionPolicyInput) (string, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func (s *RetentionService) Policy(ctx domain.Context, p domain.Principal, w uuid.UUID) (domain.RetentionPolicy, error) {
	if _, err := s.authorize(ctx, p, w); err != nil {
		return domain.RetentionPolicy{}, err
	}
	return s.repo.CurrentPolicy(ctx, w)
}

func (s *RetentionService) PreviewPolicy(ctx domain.Context, p domain.Principal, w uuid.UUID, input domain.RetentionPolicyInput) (domain.RetentionPreview, error) {
	if !validRetentionPolicy(input) {
		return domain.RetentionPreview{}, domain.ErrInvalidRetention
	}
	user, err := s.authorize(ctx, p, w)
	if err != nil {
		return domain.RetentionPreview{}, err
	}
	var preview domain.RetentionPreview
	err = s.access.WithWorkspaceLock(ctx, w, user, func(txCtx domain.Context) error {
		if _, authorizeErr := s.authorize(txCtx, p, w); authorizeErr != nil {
			return authorizeErr
		}
		current, currentErr := s.repo.CurrentPolicy(txCtx, w)
		if currentErr != nil {
			return currentErr
		}
		now := s.now()
		counts, countErr := s.repo.PreviewCounts(txCtx, w, input, now)
		if countErr != nil {
			return countErr
		}
		hash, hashErr := policyDigest(input)
		if hashErr != nil {
			return hashErr
		}
		preview = domain.RetentionPreview{ID: uuid.New(), WorkspaceID: w, PolicyRevision: current.Revision, Policy: input, PolicyHash: hash, Counts: counts, EarliestDestructiveAt: now.Add(domain.MinimumRetentionActivationDelay), ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now}
		return s.repo.SavePreview(txCtx, preview)
	})
	return preview, err
}

func storedRetention(value any, status int) (domain.StoredFeatureResponse, error) {
	raw, err := json.Marshal(value)
	return domain.StoredFeatureResponse{Status: status, Body: raw}, err
}
func decodeRetentionPolicy(stored domain.StoredFeatureResponse) (domain.RetentionPolicy, error) {
	var value domain.RetentionPolicy
	if stored.Status < 200 || stored.Status > 299 || json.Unmarshal(stored.Body, &value) != nil || value.WorkspaceID == uuid.Nil {
		return value, domain.ErrInvalidRetention
	}
	return value, nil
}
func decodeRetentionHold(stored domain.StoredFeatureResponse) (domain.RetentionHold, error) {
	var value domain.RetentionHold
	if stored.Status < 200 || stored.Status > 299 || json.Unmarshal(stored.Body, &value) != nil || value.Reference.WorkspaceID == uuid.Nil {
		return value, domain.ErrInvalidRetention
	}
	return value, nil
}

func (s *RetentionService) Schedule(ctx domain.Context, p domain.Principal, w, previewID uuid.UUID, expected int64, requestID uuid.UUID) (domain.RetentionPolicy, error) {
	if w == uuid.Nil || previewID == uuid.Nil || expected < 0 || requestID == uuid.Nil {
		return domain.RetentionPolicy{}, domain.ErrInvalidRetention
	}
	user, err := s.authorize(ctx, p, w)
	if err != nil {
		return domain.RetentionPolicy{}, err
	}
	payload := struct {
		PreviewID        uuid.UUID `json:"previewId"`
		ExpectedRevision int64     `json:"expectedRevision"`
	}{previewID, expected}
	raw, _ := json.Marshal(payload)
	var stored domain.StoredFeatureResponse
	err = s.access.WithWorkspaceLock(ctx, w, user, func(txCtx domain.Context) error {
		if _, e := s.authorize(txCtx, p, w); e != nil {
			return e
		}
		var mutationErr error
		stored, mutationErr = s.repo.WithMutation(txCtx, p, w, requestID, "retention.schedule", raw, func(callCtx domain.Context) (domain.StoredFeatureResponse, error) {
			preview, e := s.repo.Preview(callCtx, w, previewID)
			if e != nil {
				return domain.StoredFeatureResponse{}, e
			}
			calculatedHash, hashErr := policyDigest(preview.Policy)
			if hashErr != nil || calculatedHash != preview.PolicyHash {
				return domain.StoredFeatureResponse{}, domain.ErrInvalidRetention
			}
			now := s.now()
			if !now.Before(preview.ExpiresAt) {
				return domain.StoredFeatureResponse{}, domain.ErrPreviewExpired
			}
			if preview.PolicyRevision != expected {
				return domain.StoredFeatureResponse{}, domain.ErrRevisionConflict
			}
			current, e := s.repo.CurrentPolicy(callCtx, w)
			if e != nil {
				return domain.StoredFeatureResponse{}, e
			}
			if current.Revision != expected {
				return domain.StoredFeatureResponse{}, domain.ErrRevisionConflict
			}
			value, e := s.repo.Schedule(callCtx, w, preview, expected, now)
			if e != nil {
				return domain.StoredFeatureResponse{}, e
			}
			return storedRetention(value, 200)
		})
		return mutationErr
	})
	if err != nil {
		return domain.RetentionPolicy{}, err
	}
	return decodeRetentionPolicy(stored)
}

func validHold(ref domain.RetentionReference, versionID *uuid.UUID) bool {
	switch ref.Kind {
	case "all", "scene", "activity", "notification", "version", "comment":
	default:
		return false
	}
	if ref.WorkspaceID == uuid.Nil || strings.TrimSpace(ref.ID) != ref.ID || len(ref.ID) < 1 || len(ref.ID) > 256 {
		return false
	}
	if ref.Kind == "all" && ref.ID != "*" {
		return false
	}
	if ref.Kind == "scene" {
		if id, err := uuid.Parse(ref.ID); err != nil || id == uuid.Nil {
			return false
		}
	}
	if versionID != nil && (*versionID == uuid.Nil || ref.Kind != "version") {
		return false
	}
	return true
}
func (s *RetentionService) SetHold(ctx domain.Context, p domain.Principal, ref domain.RetentionReference, versionID *uuid.UUID, held bool, requestID uuid.UUID) (domain.RetentionHold, error) {
	if !validHold(ref, versionID) || requestID == uuid.Nil {
		return domain.RetentionHold{}, domain.ErrInvalidRetention
	}
	user, err := s.authorize(ctx, p, ref.WorkspaceID)
	if err != nil {
		return domain.RetentionHold{}, err
	}
	payload := struct {
		Reference domain.RetentionReference `json:"reference"`
		VersionID *uuid.UUID                `json:"versionId,omitempty"`
		Held      bool                      `json:"held"`
	}{ref, versionID, held}
	raw, _ := json.Marshal(payload)
	var stored domain.StoredFeatureResponse
	err = s.access.WithWorkspaceLock(ctx, ref.WorkspaceID, user, func(txCtx domain.Context) error {
		if _, e := s.authorize(txCtx, p, ref.WorkspaceID); e != nil {
			return e
		}
		var mutationErr error
		stored, mutationErr = s.repo.WithMutation(txCtx, p, ref.WorkspaceID, requestID, "retention.hold", raw, func(callCtx domain.Context) (domain.StoredFeatureResponse, error) {
			value, e := s.repo.SetHold(callCtx, ref, versionID, held, s.now())
			if e != nil {
				return domain.StoredFeatureResponse{}, e
			}
			return storedRetention(value, 200)
		})
		return mutationErr
	})
	if err != nil {
		return domain.RetentionHold{}, err
	}
	return decodeRetentionHold(stored)
}

func (s *RetentionService) PurgeBatch(ctx domain.Context, now time.Time, limit int) (domain.PurgeResult, error) {
	if s.repo == nil || now.IsZero() || limit < 1 || limit > 1000 {
		return domain.PurgeResult{}, domain.ErrInvalidRetention
	}
	return s.repo.PurgeBatch(ctx, now.UTC(), limit)
}

func (s *RetentionService) JobHandler(ctx domain.Context, _ domain.FeatureJob) error {
	_, err := s.PurgeBatch(ctx, s.now(), 100)
	return err
}

var _ domain.RetentionUsecase = (*RetentionService)(nil)
