package usecase

import (
	"bytes"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const versionResponseBytes = 32 * 1024

type VersionService struct {
	repo     domain.SceneVersionRepository
	features domain.FeatureService
	caps     domain.CapabilityService
	remote   domain.AgentCollabRepository
	assets   domain.ScopedAssetViewer
	events   versionEventService
}

func (s *VersionService) SetAssetViewer(assets domain.ScopedAssetViewer) { s.assets = assets }
func (s *VersionService) SetEventService(events versionEventService)     { s.events = events }

type versionEventService interface {
	AppendEvent(domain.Context, domain.EventInput) (domain.WorkspaceEvent, error)
}

func NewVersionService(repo domain.SceneVersionRepository, features domain.FeatureService, caps domain.CapabilityService, remote domain.AgentCollabRepository) *VersionService {
	return &VersionService{repo: repo, features: features, caps: caps, remote: remote}
}

func validVersionLabel(label *string) bool {
	return label == nil || utf8.ValidString(*label) && strings.TrimSpace(*label) == *label && utf8.RuneCountInString(*label) >= 1 && utf8.RuneCountInString(*label) <= 255
}

func validSequence(value string) bool {
	n, err := strconv.ParseInt(value, 10, 64)
	return err == nil && n >= 0 && strconv.FormatInt(n, 10) == value
}

func canonicalVersionPayload(input domain.CreateVersionInput) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(input); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte{'\n'}), nil
}

func (s *VersionService) Create(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, input domain.CreateVersionInput) (domain.SceneVersion, error) {
	if principal.Kind != domain.PrincipalUser || principal.UserID == nil || input.RequestID == uuid.Nil || !validSequence(input.ExpectedSequence) || !validVersionLabel(input.Label) || s.features == nil || s.caps == nil {
		return domain.SceneVersion{}, domain.ErrInvalidVersionInput
	}
	if err := s.features.Require(ctx, principal, ref, domain.FeatureVersionCreate); err != nil {
		return domain.SceneVersion{}, err
	}
	if s.remote == nil {
		return domain.SceneVersion{}, &domain.AgentCollabError{Status: 503, Code: "UNAVAILABLE"}
	}
	payload, err := canonicalVersionPayload(input)
	if err != nil {
		return domain.SceneVersion{}, err
	}
	digest := sha256.Sum256(payload)
	grant, err := s.caps.AuthorizeCommand(ctx, principal, domain.SceneMutation{WorkspaceID: ref.WorkspaceID.String(), SceneID: ref.SceneID.String(), Action: domain.ActionVersionCreate, RequestID: input.RequestID.String(), PayloadHash: hex.EncodeToString(digest[:]), ExpectedSequence: input.ExpectedSequence})
	if err != nil {
		return domain.SceneVersion{}, err
	}
	raw, err := s.remote.Call(ctx, "versions", ref, map[string]any{"grant": grant.Token, "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "input": input}, versionResponseBytes)
	if err != nil {
		return domain.SceneVersion{}, err
	}
	var result domain.SceneVersion
	if json.Unmarshal(raw, &result) != nil || result.ID == uuid.Nil || result.DocumentID == uuid.Nil || result.Sequence != input.ExpectedSequence {
		return domain.SceneVersion{}, domain.ErrInvalidVersionInput
	}
	result.WorkspaceID, result.SceneID = ref.WorkspaceID, ref.SceneID
	return result, nil
}

func versionCursor(raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return nil, domain.ErrInvalidVersionInput
	}
	return &id, nil
}

func (s *VersionService) List(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, cursor string, limit int) (domain.VersionPage, error) {
	out := domain.VersionPage{Items: []domain.SceneVersion{}}
	if s.repo == nil || s.features == nil || limit < 1 || limit > 50 {
		return out, domain.ErrInvalidVersionInput
	}
	if err := s.features.Require(ctx, principal, ref, domain.FeatureVersionRead); err != nil {
		return out, err
	}
	c, err := versionCursor(cursor)
	if err != nil {
		return out, err
	}
	rows, err := s.repo.ListVersions(ctx, ref, c, limit+1)
	if err != nil {
		return out, err
	}
	if len(rows) > limit {
		out.NextCursor = rows[limit-1].ID.String()
		rows = rows[:limit]
	}
	out.Items = rows
	return out, nil
}

func (s *VersionService) Get(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, id uuid.UUID) (domain.SceneVersion, error) {
	if id == uuid.Nil || s.repo == nil || s.features == nil {
		return domain.SceneVersion{}, domain.ErrInvalidVersionInput
	}
	if err := s.features.Require(ctx, principal, ref, domain.FeatureVersionRead); err != nil {
		return domain.SceneVersion{}, err
	}
	version, err := s.repo.Version(ctx, ref, id)
	if err != nil {
		return domain.SceneVersion{}, err
	}
	var content struct {
		Files map[string]json.RawMessage `json:"files"`
	}
	if json.Unmarshal(version.Content, &content) != nil {
		return domain.SceneVersion{}, domain.ErrInvalidVersionInput
	}
	if len(content.Files) == 0 {
		return version, nil
	}
	if s.assets == nil {
		return domain.SceneVersion{}, domain.ErrInaccessible
	}
	files := make([]string, 0, len(content.Files))
	for file := range content.Files {
		files = append(files, file)
	}
	version.Assets, err = s.assets.PresignScopedView(ctx, ref, files, time.Now().Add(5*time.Minute))
	if err != nil {
		return domain.SceneVersion{}, err
	}
	if len(version.Assets) != len(files) {
		return domain.SceneVersion{}, domain.ErrInaccessible
	}
	return version, nil
}

func (s *VersionService) SetMetadata(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, id uuid.UUID, input domain.VersionMetadataInput) (domain.SceneVersion, error) {
	if id == uuid.Nil || input.RequestID == uuid.Nil || input.ExpectedRevision < 1 || input.Label == nil && input.Pinned == nil || !validVersionLabel(input.Label) || s.repo == nil || s.features == nil || s.events == nil {
		return domain.SceneVersion{}, domain.ErrInvalidVersionInput
	}
	payload, err := json.Marshal(struct {
		VersionID uuid.UUID
		Input     domain.VersionMetadataInput
	}{id, input})
	if err != nil {
		return domain.SceneVersion{}, err
	}
	stored, err := s.features.Mutate(ctx, principal, ref, domain.FeatureVersionCreate, input.RequestID, payload, func(txCtx domain.Context) (domain.StoredFeatureResponse, error) {
		updated, e := s.repo.SetVersionMetadata(txCtx, ref, id, input.ExpectedRevision, input.Label, input.Pinned)
		if e != nil {
			return domain.StoredFeatureResponse{}, e
		}
		if _, e = s.events.AppendEvent(txCtx, domain.EventInput{WorkspaceID: ref.WorkspaceID, SceneID: &ref.SceneID, ActorKind: domain.EventActorUser, ActorID: principal.ID, Kind: "version.metadata_updated", SubjectID: id.String(), SafeMetadata: map[string]any{"versionId": id.String(), "metadataRevision": updated.MetadataRevision, "requestId": input.RequestID.String(), "labelChanged": input.Label != nil, "pinnedChanged": input.Pinned != nil}, Source: "scene_version_metadata", SourceKey: id.String() + ":revision:" + strconv.FormatInt(updated.MetadataRevision, 10)}); e != nil {
			return domain.StoredFeatureResponse{}, e
		}
		body, e := json.Marshal(updated)
		return domain.StoredFeatureResponse{Status: 200, Body: body}, e
	})
	if err != nil {
		return domain.SceneVersion{}, err
	}
	return decodeStored[domain.SceneVersion](stored)
}

func elementIndex(raw json.RawMessage) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	var content struct {
		Elements []json.RawMessage          `json:"elements"`
		AppState map[string]json.RawMessage `json:"appState"`
	}
	if json.Unmarshal(raw, &content) != nil {
		return nil, nil, domain.ErrInvalidVersionInput
	}
	items := make(map[string]json.RawMessage, len(content.Elements))
	for _, element := range content.Elements {
		var identity struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(element, &identity) != nil || identity.ID == "" {
			return nil, nil, domain.ErrInvalidVersionInput
		}
		items[identity.ID] = element
	}
	return items, content.AppState, nil
}

func equivalentJSON(left, right json.RawMessage) (bool, error) {
	var a, b any
	if json.Unmarshal(left, &a) != nil || json.Unmarshal(right, &b) != nil {
		return false, domain.ErrInvalidVersionInput
	}
	leftCanonical, leftErr := json.Marshal(a)
	rightCanonical, rightErr := json.Marshal(b)
	if leftErr != nil || rightErr != nil {
		return false, domain.ErrInvalidVersionInput
	}
	return bytes.Equal(leftCanonical, rightCanonical), nil
}

func semanticDiff(before, after json.RawMessage) (domain.SceneDiff, error) {
	result := domain.SceneDiff{AddedIDs: []string{}, RemovedIDs: []string{}, ChangedIDs: []string{}, ChangedAppStateFields: []string{}}
	a, appA, err := elementIndex(before)
	if err != nil {
		return result, err
	}
	b, appB, err := elementIndex(after)
	if err != nil {
		return result, err
	}
	for id, right := range b {
		if left, ok := a[id]; !ok {
			result.AddedIDs = append(result.AddedIDs, id)
		} else {
			equal, compareErr := equivalentJSON(left, right)
			if compareErr != nil {
				return result, compareErr
			}
			if !equal {
				result.ChangedIDs = append(result.ChangedIDs, id)
			}
		}
	}
	for id := range a {
		if _, ok := b[id]; !ok {
			result.RemovedIDs = append(result.RemovedIDs, id)
		}
	}
	keys := map[string]struct{}{}
	for key := range appA {
		keys[key] = struct{}{}
	}
	for key := range appB {
		keys[key] = struct{}{}
	}
	for key := range keys {
		left, leftOK := appA[key]
		right, rightOK := appB[key]
		equal := leftOK == rightOK
		if equal && leftOK {
			equal, err = equivalentJSON(left, right)
			if err != nil {
				return result, err
			}
		}
		if !equal {
			result.ChangedAppStateFields = append(result.ChangedAppStateFields, key)
		}
	}
	slices.Sort(result.AddedIDs)
	slices.Sort(result.RemovedIDs)
	slices.Sort(result.ChangedIDs)
	slices.Sort(result.ChangedAppStateFields)
	return result, nil
}

func (s *VersionService) Diff(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, id uuid.UUID, againstSequence string) (domain.SceneDiff, error) {
	if id == uuid.Nil || !validSequence(againstSequence) || s.repo == nil || s.features == nil || s.caps == nil {
		return domain.SceneDiff{}, domain.ErrInvalidVersionInput
	}
	if err := s.features.Require(ctx, principal, ref, domain.FeatureVersionRead); err != nil {
		return domain.SceneDiff{}, err
	}
	if s.remote == nil {
		return domain.SceneDiff{}, &domain.AgentCollabError{Status: 503, Code: "UNAVAILABLE"}
	}
	version, err := s.repo.Version(ctx, ref, id)
	if err != nil {
		return domain.SceneDiff{}, err
	}
	empty := sha256.Sum256(nil)
	grant, err := s.caps.AuthorizeCommand(ctx, principal, domain.SceneMutation{WorkspaceID: ref.WorkspaceID.String(), SceneID: ref.SceneID.String(), Action: domain.ActionSceneRead, RequestID: uuid.NewString(), PayloadHash: hex.EncodeToString(empty[:])})
	if err != nil {
		return domain.SceneDiff{}, err
	}
	raw, err := s.remote.Call(ctx, "projection", ref, map[string]any{"grant": grant.Token, "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String()}, 26<<20)
	if err != nil {
		return domain.SceneDiff{}, err
	}
	var current struct {
		Sequence string          `json:"sequence"`
		Content  json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &current) != nil || current.Sequence != againstSequence {
		return domain.SceneDiff{}, domain.ErrRevisionConflict
	}
	return semanticDiff(version.Content, current.Content)
}

func canonicalRestorePayload(ref domain.SceneRef, input domain.RestoreVersionInput) ([]byte, error) {
	return json.Marshal(struct {
		WorkspaceID      string    `json:"workspaceId"`
		SceneID          string    `json:"sceneId"`
		VersionID        uuid.UUID `json:"versionId"`
		RequestID        uuid.UUID `json:"requestId"`
		ExpectedSequence string    `json:"expectedSequence"`
	}{ref.WorkspaceID.String(), ref.SceneID.String(), input.VersionID, input.RequestID, input.ExpectedSequence})
}

func (s *VersionService) Restore(ctx domain.Context, principal domain.Principal, ref domain.SceneRef, input domain.RestoreVersionInput) (domain.RestoreVersionResult, error) {
	if principal.Kind != domain.PrincipalUser || principal.UserID == nil || input.VersionID == uuid.Nil || input.RequestID == uuid.Nil || !validSequence(input.ExpectedSequence) || s.features == nil || s.caps == nil {
		return domain.RestoreVersionResult{}, domain.ErrInvalidVersionInput
	}
	if err := s.features.Require(ctx, principal, ref, domain.FeatureVersionRestore); err != nil {
		return domain.RestoreVersionResult{}, err
	}
	if s.remote == nil {
		return domain.RestoreVersionResult{}, &domain.AgentCollabError{Status: 503, Code: "UNAVAILABLE"}
	}
	payload, err := canonicalRestorePayload(ref, input)
	if err != nil {
		return domain.RestoreVersionResult{}, err
	}
	digest := sha256.Sum256(payload)
	grant, err := s.caps.AuthorizeCommand(ctx, principal, domain.SceneMutation{WorkspaceID: ref.WorkspaceID.String(), SceneID: ref.SceneID.String(), Action: domain.ActionVersionRestore, RequestID: input.RequestID.String(), PayloadHash: hex.EncodeToString(digest[:]), ExpectedSequence: input.ExpectedSequence})
	if err != nil {
		return domain.RestoreVersionResult{}, err
	}
	raw, err := s.remote.Call(ctx, "version-restore", ref, map[string]any{"grant": grant.Token, "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "input": input}, versionResponseBytes)
	if err != nil {
		return domain.RestoreVersionResult{}, err
	}
	var out domain.RestoreVersionResult
	if json.Unmarshal(raw, &out) != nil || out.RequestID != input.RequestID || out.PreRestoreVersionID == uuid.Nil || !validSequence(out.Sequence) || out.StateVector == "" {
		return domain.RestoreVersionResult{}, domain.ErrInvalidVersionInput
	}
	return out, nil
}
