package usecase

import (
	"bytes"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const maxLibraryBytes = 25 * 1024 * 1024

type libraryFileJSON struct {
	ID            string `json:"id"`
	MimeType      string `json:"mimeType"`
	AssetID       string `json:"assetId"`
	Created       int64  `json:"created"`
	LastRetrieved *int64 `json:"lastRetrieved,omitempty"`
	Version       *int64 `json:"version,omitempty"`
}
type libraryItemJSON struct {
	ID                string                     `json:"id"`
	Status            string                     `json:"status"`
	Created           int64                      `json:"created"`
	Name              string                     `json:"name,omitempty"`
	Elements          []json.RawMessage          `json:"elements"`
	Files             map[string]libraryFileJSON `json:"files"`
	PresentationOrder []string                   `json:"presentationOrder,omitempty"`
}
type libraryEnvelopeJSON struct {
	Type         string            `json:"type"`
	Version      int               `json:"version"`
	Source       string            `json:"source,omitempty"`
	LibraryItems []libraryItemJSON `json:"libraryItems"`
}

var libraryBaseFields = map[string]bool{"id": true, "type": true, "x": true, "y": true, "width": true, "height": true, "angle": true, "strokeColor": true, "backgroundColor": true, "fillStyle": true, "strokeWidth": true, "strokeStyle": true, "roughness": true, "opacity": true, "roundness": true, "seed": true, "version": true, "versionNonce": true, "index": true, "isDeleted": true, "groupIds": true, "frameId": true, "boundElements": true, "updated": true, "link": true, "locked": true, "customData": true}

func validateLibraryElements(item libraryItemJSON) error {
	elements, types := map[string]map[string]any{}, map[string]string{}
	extra := map[string][]string{"text": {"fontSize", "fontFamily", "text", "textAlign", "verticalAlign", "containerId", "originalText", "autoResize", "lineHeight"}, "line": {"points", "lastCommittedPoint", "startBinding", "endBinding", "startArrowhead", "endArrowhead"}, "arrow": {"points", "lastCommittedPoint", "startBinding", "endBinding", "startArrowhead", "endArrowhead", "elbowed", "fixedSegments", "startIsSpecial", "endIsSpecial"}, "freedraw": {"points", "pressures", "simulatePressure", "lastCommittedPoint"}, "image": {"fileId", "status", "scale", "crop"}, "frame": {"name"}, "magicframe": {"name"}, "rectangle": {}, "diamond": {}, "ellipse": {}}
	requiredBase := []string{"id", "type", "x", "y", "width", "height", "angle", "strokeColor", "backgroundColor", "fillStyle", "strokeWidth", "strokeStyle", "roughness", "opacity", "roundness", "seed", "version", "versionNonce", "index", "isDeleted", "groupIds", "frameId", "boundElements", "updated", "link", "locked"}
	requiredExtra := map[string][]string{"text": extra["text"], "line": extra["line"], "arrow": {"points", "lastCommittedPoint", "startBinding", "endBinding", "startArrowhead", "endArrowhead"}, "freedraw": extra["freedraw"], "image": extra["image"], "frame": extra["frame"], "magicframe": extra["magicframe"]}
	for _, raw := range item.Elements {
		var element map[string]any
		if json.Unmarshal(raw, &element) != nil {
			return domain.ErrInvalidLibrary
		}
		id, idOK := element["id"].(string)
		kind, typeOK := element["type"].(string)
		typeFields, known := extra[kind]
		if !idOK || id == "" || !typeOK || !known || elements[id] != nil {
			return domain.ErrInvalidLibrary
		}
		allowed := map[string]bool{}
		for key := range libraryBaseFields {
			allowed[key] = true
		}
		for _, key := range typeFields {
			allowed[key] = true
		}
		for key := range element {
			if !allowed[key] {
				return domain.ErrInvalidLibrary
			}
		}
		for _, key := range append(requiredBase, requiredExtra[kind]...) {
			if _, exists := element[key]; !exists {
				return domain.ErrInvalidLibrary
			}
		}
		if link, ok := element["link"].(string); ok && link != "" {
			parsed, err := url.Parse(link)
			if err != nil || parsed.User != nil || parsed.IsAbs() && !slices.Contains([]string{"http", "https", "mailto"}, parsed.Scheme) {
				return domain.ErrInvalidLibrary
			}
		}
		elements[id], types[id] = element, kind
	}
	ref := func(value any) (string, bool) { id, ok := value.(string); return id, ok && elements[id] != nil }
	for id, element := range elements {
		if frame := element["frameId"]; frame != nil {
			target, ok := ref(frame)
			if !ok || target == id || !slices.Contains([]string{"frame", "magicframe"}, types[target]) {
				return domain.ErrInvalidLibrary
			}
		}
		if types[id] == "text" && element["containerId"] != nil {
			target, ok := ref(element["containerId"])
			if !ok || target == id || !slices.Contains([]string{"rectangle", "diamond", "ellipse", "arrow"}, types[target]) {
				return domain.ErrInvalidLibrary
			}
		}
		if bound, ok := element["boundElements"].([]any); ok {
			for _, rawBound := range bound {
				entry, valid := rawBound.(map[string]any)
				target, targetValid := ref(entry["id"])
				boundType, typeValid := entry["type"].(string)
				if !valid || !targetValid || !typeValid || types[target] != boundType || target == id {
					return domain.ErrInvalidLibrary
				}
			}
		}
		if types[id] == "arrow" || types[id] == "line" {
			for _, name := range []string{"startBinding", "endBinding"} {
				if binding, ok := element[name].(map[string]any); ok {
					target, valid := ref(binding["elementId"])
					if !valid || target == id {
						return domain.ErrInvalidLibrary
					}
				}
			}
		}
		if types[id] == "image" && element["fileId"] != nil {
			fileID, ok := element["fileId"].(string)
			if !ok || item.Files[fileID].ID == "" {
				return domain.ErrInvalidLibrary
			}
		}
	}
	seen := map[string]bool{}
	for _, id := range item.PresentationOrder {
		if seen[id] || !slices.Contains([]string{"frame", "magicframe"}, types[id]) {
			return domain.ErrInvalidLibrary
		}
		seen[id] = true
	}
	return nil
}

type libraryAccess interface {
	Resolve(domain.Context, domain.Principal, domain.SceneRef) (domain.AccessDecision, error)
	ResolveWorkspace(domain.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error)
	WithWorkspaceLock(domain.Context, uuid.UUID, uuid.UUID, func(domain.Context) error) error
}
type libraryFeatures interface {
	Policy(domain.Context, domain.Principal, uuid.UUID) (domain.WorkspaceFeaturePolicy, error)
}

type LibraryService struct {
	repo     domain.LibraryRepository
	features libraryFeatures
	access   libraryAccess
	caps     domain.CapabilityService
	remote   domain.AgentCollabRepository
	stager   domain.LibraryAssetStager
}

func NewLibraryService(repo domain.LibraryRepository, features libraryFeatures, access libraryAccess, caps domain.CapabilityService, remote domain.AgentCollabRepository) *LibraryService {
	return &LibraryService{repo: repo, features: features, access: access, caps: caps, remote: remote}
}
func (s *LibraryService) SetAssetStager(stager domain.LibraryAssetStager) { s.stager = stager }

func libraryUser(principal domain.Principal) (uuid.UUID, error) {
	if principal.Kind != domain.PrincipalUser || principal.UserID == nil || principal.ID != principal.UserID.String() {
		return uuid.Nil, domain.ErrAccessDenied
	}
	return *principal.UserID, nil
}
func validLibraryName(name string) bool {
	return utf8.ValidString(name) && strings.TrimSpace(name) == name && utf8.RuneCountInString(name) > 0 && utf8.RuneCountInString(name) <= 100
}
func validLibrarySequence(value string) bool {
	n, err := strconv.ParseInt(value, 10, 64)
	return err == nil && n >= 0 && strconv.FormatInt(n, 10) == value
}

func validateLibraryContent(raw json.RawMessage) error {
	if len(raw) == 0 || len(raw) > maxLibraryBytes || !json.Valid(raw) {
		return domain.ErrInvalidLibrary
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value libraryEnvelopeJSON
	if decoder.Decode(&value) != nil || value.Type != "excalidrawlib" || value.Version != 2 || len(value.LibraryItems) < 1 || len(value.LibraryItems) > 500 {
		return domain.ErrInvalidLibrary
	}
	if value.Source != "" {
		if strings.Contains(value.Source, "@") || !strings.HasPrefix(value.Source, "https://") {
			return domain.ErrInvalidLibrary
		}
	}
	items := map[string]bool{}
	for _, item := range value.LibraryItems {
		if item.ID == "" || items[item.ID] || item.Status != "published" && item.Status != "unpublished" || item.Created < 0 {
			return domain.ErrInvalidLibrary
		}
		items[item.ID] = true
		if validateLibraryElements(item) != nil {
			return domain.ErrInvalidLibrary
		}
		for key, file := range item.Files {
			if key != file.ID || file.AssetID == "" {
				return domain.ErrInvalidLibrary
			}
			switch file.MimeType {
			case "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif":
			default:
				return domain.ErrInvalidLibrary
			}
		}
	}
	return nil
}

func libraryAssetReferences(raw json.RawMessage) (map[string]uuid.UUID, error) {
	var value struct {
		LibraryItems []struct {
			Files map[string]struct {
				AssetID string `json:"assetId"`
			} `json:"files"`
		} `json:"libraryItems"`
	}
	if json.Unmarshal(raw, &value) != nil {
		return nil, domain.ErrInvalidLibrary
	}
	result := map[string]uuid.UUID{}
	for _, item := range value.LibraryItems {
		for fileID, file := range item.Files {
			assetID, err := uuid.Parse(file.AssetID)
			if err != nil || assetID == uuid.Nil {
				return nil, domain.ErrInvalidLibrary
			}
			if previous, exists := result[fileID]; exists && previous != assetID {
				return nil, domain.ErrInvalidLibrary
			}
			result[fileID] = assetID
		}
	}
	return result, nil
}

func (s *LibraryService) authorize(ctx domain.Context, principal domain.Principal, workspaceID uuid.UUID, owner bool) (uuid.UUID, domain.AccessDecision, error) {
	user, err := libraryUser(principal)
	if err != nil || workspaceID == uuid.Nil || s.repo == nil || s.features == nil || s.access == nil {
		return uuid.Nil, domain.AccessDecision{}, domain.ErrAccessDenied
	}
	decision, err := s.access.ResolveWorkspace(ctx, principal, workspaceID)
	if err != nil {
		return uuid.Nil, domain.AccessDecision{}, err
	}
	policy, err := s.features.Policy(ctx, principal, workspaceID)
	if err != nil {
		return uuid.Nil, domain.AccessDecision{}, err
	}
	if !policy.LibrariesEnabled {
		return uuid.Nil, domain.AccessDecision{}, domain.ErrFeatureDisabled
	}
	if owner && !decision.Owner {
		return uuid.Nil, domain.AccessDecision{}, domain.ErrAccessDenied
	}
	return user, decision, nil
}

func (s *LibraryService) store(ctx domain.Context, principal domain.Principal, workspaceID, itemID uuid.UUID, expected int64, name string, content json.RawMessage, pins []domain.LibraryAssetPin) (domain.LibraryItem, domain.LibraryRevision, error) {
	if !validLibraryName(name) || validateLibraryContent(content) != nil || len(pins) > 100 {
		return domain.LibraryItem{}, domain.LibraryRevision{}, domain.ErrInvalidLibrary
	}
	references, err := libraryAssetReferences(content)
	if err != nil || len(references) != len(pins) {
		return domain.LibraryItem{}, domain.LibraryRevision{}, domain.ErrInvalidLibrary
	}
	seenPins := map[string]bool{}
	for _, pin := range pins {
		if seenPins[pin.FileID] || references[pin.FileID] != pin.AssetID {
			return domain.LibraryItem{}, domain.LibraryRevision{}, domain.ErrInvalidLibrary
		}
		seenPins[pin.FileID] = true
	}
	user, _, err := s.authorize(ctx, principal, workspaceID, true)
	if err != nil {
		return domain.LibraryItem{}, domain.LibraryRevision{}, err
	}
	for _, pin := range pins {
		if pin.SourceSceneID == uuid.Nil || pin.AssetID == uuid.Nil || pin.FileID == "" {
			return domain.LibraryItem{}, domain.LibraryRevision{}, domain.ErrInvalidLibrary
		}
		if _, e := s.access.Resolve(ctx, principal, domain.SceneRef{WorkspaceID: workspaceID, SceneID: pin.SourceSceneID}); e != nil {
			return domain.LibraryItem{}, domain.LibraryRevision{}, e
		}
	}
	var out domain.LibraryItem
	var revision domain.LibraryRevision
	err = s.access.WithWorkspaceLock(ctx, workspaceID, user, func(tx domain.Context) error {
		if _, _, e := s.authorize(tx, principal, workspaceID, true); e != nil {
			return e
		}
		if expected > 0 {
			current, e := s.repo.Item(tx, workspaceID, itemID)
			if e != nil {
				return e
			}
			if current.CurrentRevision != expected || current.OwnerID != user {
				return domain.ErrRevisionConflict
			}
		}
		digest := sha256.Sum256(content)
		out = domain.LibraryItem{ID: itemID, WorkspaceID: workspaceID, OwnerID: user, Name: name, CurrentRevision: expected + 1}
		revision = domain.LibraryRevision{LibraryItemID: itemID, WorkspaceID: workspaceID, Revision: expected + 1, Content: append(json.RawMessage(nil), content...), ContentHash: hex.EncodeToString(digest[:]), ByteSize: int64(len(content))}
		var e error
		out, revision, e = s.repo.CreateRevision(tx, out, revision, pins)
		return e
	})
	return out, revision, err
}
func (s *LibraryService) Import(ctx domain.Context, p domain.Principal, w uuid.UUID, in domain.LibraryImportInput) (domain.LibraryItem, domain.LibraryRevision, error) {
	if in.RequestID == uuid.Nil {
		return domain.LibraryItem{}, domain.LibraryRevision{}, domain.ErrInvalidLibrary
	}
	return s.store(ctx, p, w, uuid.New(), 0, in.Name, in.Content, in.Assets)
}
func (s *LibraryService) Publish(ctx domain.Context, p domain.Principal, w, id uuid.UUID, in domain.LibraryPublishInput) (domain.LibraryItem, domain.LibraryRevision, error) {
	if id == uuid.Nil || in.RequestID == uuid.Nil || in.ExpectedRevision < 1 {
		return domain.LibraryItem{}, domain.LibraryRevision{}, domain.ErrInvalidLibrary
	}
	return s.store(ctx, p, w, id, in.ExpectedRevision, in.Name, in.Content, in.Assets)
}
func (s *LibraryService) List(ctx domain.Context, p domain.Principal, w uuid.UUID, q string, limit int) (domain.LibraryPage, error) {
	page := domain.LibraryPage{Items: []domain.LibraryItem{}}
	if _, _, err := s.authorize(ctx, p, w, false); err != nil {
		return page, err
	}
	if limit < 1 || limit > 50 || utf8.RuneCountInString(q) > 100 {
		return page, domain.ErrInvalidLibrary
	}
	items, err := s.repo.List(ctx, w, q, limit)
	page.Items = items
	return page, err
}
func (s *LibraryService) GetRevision(ctx domain.Context, p domain.Principal, w, id uuid.UUID, revision int64) (domain.LibraryRevision, error) {
	if _, _, err := s.authorize(ctx, p, w, false); err != nil {
		return domain.LibraryRevision{}, err
	}
	if id == uuid.Nil || revision < 1 {
		return domain.LibraryRevision{}, domain.ErrInvalidLibrary
	}
	return s.repo.Revision(ctx, w, id, revision)
}
func (s *LibraryService) Export(ctx domain.Context, p domain.Principal, w, id uuid.UUID, revision int64) (json.RawMessage, error) {
	value, err := s.GetRevision(ctx, p, w, id, revision)
	return value.Content, err
}
func (s *LibraryService) Insert(ctx domain.Context, p domain.Principal, ref domain.SceneRef, in domain.LibraryInsertInput) (domain.LibraryInsertResult, error) {
	if in.LibraryItemID == uuid.Nil || in.Revision < 1 || in.RequestID == uuid.Nil || !validLibrarySequence(in.ExpectedSequence) || s.caps == nil || s.remote == nil {
		return domain.LibraryInsertResult{}, domain.ErrInvalidLibrary
	}
	if _, _, err := s.authorize(ctx, p, ref.WorkspaceID, false); err != nil {
		return domain.LibraryInsertResult{}, err
	}
	decision, err := s.access.Resolve(ctx, p, ref)
	if err != nil {
		return domain.LibraryInsertResult{}, err
	}
	if decision.Access != domain.AccessEdit && decision.Access != domain.AccessManage {
		return domain.LibraryInsertResult{}, domain.ErrAccessDenied
	}
	revision, err := s.repo.Revision(ctx, ref.WorkspaceID, in.LibraryItemID, in.Revision)
	if err != nil {
		return domain.LibraryInsertResult{}, err
	}
	content := append(json.RawMessage(nil), revision.Content...)
	pins, err := s.repo.RevisionAssets(ctx, ref.WorkspaceID, in.LibraryItemID, in.Revision)
	if err != nil {
		return domain.LibraryInsertResult{}, err
	}
	if len(pins) > 0 {
		if s.stager == nil {
			return domain.LibraryInsertResult{}, &domain.AgentCollabError{Status: 503, Code: "LIBRARY_ASSET_STAGER_UNAVAILABLE"}
		}
		for _, pin := range pins {
			if _, sourceErr := s.access.Resolve(ctx, p, domain.SceneRef{WorkspaceID: ref.WorkspaceID, SceneID: pin.SourceSceneID}); sourceErr != nil {
				return domain.LibraryInsertResult{}, sourceErr
			}
		}
		replacements, stageErr := s.stager.Stage(ctx, p, ref, in.RequestID, pins)
		if stageErr != nil {
			return domain.LibraryInsertResult{}, stageErr
		}
		var library map[string]any
		if json.Unmarshal(content, &library) != nil {
			return domain.LibraryInsertResult{}, domain.ErrInvalidLibrary
		}
		rawItems, ok := library["libraryItems"].([]any)
		if !ok {
			return domain.LibraryInsertResult{}, domain.ErrInvalidLibrary
		}
		for _, rawItem := range rawItems {
			entry, ok := rawItem.(map[string]any)
			if !ok {
				return domain.LibraryInsertResult{}, domain.ErrInvalidLibrary
			}
			files, ok := entry["files"].(map[string]any)
			if !ok {
				return domain.LibraryInsertResult{}, domain.ErrInvalidLibrary
			}
			for fileID, rawFile := range files {
				replacement, exists := replacements[fileID]
				file, valid := rawFile.(map[string]any)
				if !exists || !valid {
					return domain.LibraryInsertResult{}, domain.ErrInvalidLibrary
				}
				file["assetId"] = replacement.String()
			}
		}
		content, err = json.Marshal(library)
		if err != nil {
			return domain.LibraryInsertResult{}, err
		}
	}
	payload, err := json.Marshal(struct {
		WorkspaceID      uuid.UUID       `json:"workspaceId"`
		SceneID          uuid.UUID       `json:"sceneId"`
		RequestID        uuid.UUID       `json:"requestId"`
		ExpectedSequence string          `json:"expectedSequence"`
		NativeItemID     string          `json:"nativeItemId,omitempty"`
		Content          json.RawMessage `json:"content"`
	}{ref.WorkspaceID, ref.SceneID, in.RequestID, in.ExpectedSequence, in.NativeItemID, content})
	if err != nil {
		return domain.LibraryInsertResult{}, err
	}
	digest := sha256.Sum256(payload)
	grant, err := s.caps.AuthorizeCommand(ctx, p, domain.SceneMutation{WorkspaceID: ref.WorkspaceID.String(), SceneID: ref.SceneID.String(), Action: domain.ActionLibraryInsert, RequestID: in.RequestID.String(), PayloadHash: hex.EncodeToString(digest[:]), ExpectedSequence: in.ExpectedSequence})
	if err != nil {
		return domain.LibraryInsertResult{}, err
	}
	raw, err := s.remote.Call(ctx, "library-insertions", ref, map[string]any{"grant": grant.Token, "insertionBase64": base64.StdEncoding.EncodeToString(payload)}, 64*1024)
	if err != nil {
		return domain.LibraryInsertResult{}, err
	}
	var result domain.LibraryInsertResult
	if json.Unmarshal(raw, &result) != nil || result.RequestID != in.RequestID || result.Sequence == "" || len(result.ElementIDs) > 10000 {
		return domain.LibraryInsertResult{}, domain.ErrInvalidLibrary
	}
	return result, nil
}
