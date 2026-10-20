package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"strings"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type scimAccess interface {
	RequireOwner(context.Context, uuid.UUID, uuid.UUID) error
	WithWorkspaceLock(context.Context, uuid.UUID, uuid.UUID, func(context.Context) error) error
}

type SCIMService struct {
	repo   domain.SCIMRepository
	access scimAccess
	now    func() time.Time
}

func NewSCIMService(repo domain.SCIMRepository, access scimAccess, clocks ...func() time.Time) *SCIMService {
	now := time.Now
	if len(clocks) > 0 && clocks[0] != nil {
		now = clocks[0]
	}
	return &SCIMService{repo: repo, access: access, now: now}
}

func scimUserID(p domain.Principal) (uuid.UUID, error) {
	if p.Kind != domain.PrincipalUser || p.UserID == nil || *p.UserID == uuid.Nil {
		return uuid.Nil, domain.ErrAccessDenied
	}
	return *p.UserID, nil
}
func (s *SCIMService) ownerLocked(ctx context.Context, p domain.Principal, w uuid.UUID, fn func(context.Context) error) error {
	u, e := scimUserID(p)
	if e != nil {
		return e
	}
	if e = s.access.RequireOwner(ctx, u, w); e != nil {
		return e
	}
	return s.access.WithWorkspaceLock(ctx, w, u, func(tx context.Context) error {
		if e = s.access.RequireOwner(tx, u, w); e != nil {
			return e
		}
		return fn(tx)
	})
}
func (s *SCIMService) RotateToken(ctx context.Context, p domain.Principal, w uuid.UUID) (domain.SCIMTokenResult, error) {
	var raw [domain.SCIMTokenBytes]byte
	if _, e := rand.Read(raw[:]); e != nil {
		return domain.SCIMTokenResult{}, e
	}
	token := "scim_" + base64.RawURLEncoding.EncodeToString(raw[:])
	sum := sha256.Sum256([]byte(token))
	now := s.now().UTC()
	u, _ := scimUserID(p)
	result := domain.SCIMTokenResult{Token: token, Prefix: token[:12], CreatedAt: now}
	e := s.ownerLocked(ctx, p, w, func(tx context.Context) error {
		return s.repo.RotateCredential(tx, domain.SCIMCredential{WorkspaceID: w, TokenHash: sum[:], Prefix: result.Prefix, CreatedBy: u, CreatedAt: now})
	})
	if e != nil {
		return domain.SCIMTokenResult{}, e
	}
	return result, nil
}
func (s *SCIMService) TokenStatus(ctx context.Context, p domain.Principal, w uuid.UUID) (domain.SCIMTokenStatus, error) {
	u, e := scimUserID(p)
	if e != nil {
		return domain.SCIMTokenStatus{}, e
	}
	if e = s.access.RequireOwner(ctx, u, w); e != nil {
		return domain.SCIMTokenStatus{}, e
	}
	return s.repo.CredentialStatus(ctx, w)
}
func (s *SCIMService) RevokeToken(ctx context.Context, p domain.Principal, w uuid.UUID) error {
	return s.ownerLocked(ctx, p, w, func(tx context.Context) error { return s.repo.RevokeCredential(tx, w, s.now().UTC()) })
}

func validManagedRole(role string) bool {
	return role == string(domain.WorkspaceRoleEditor) || role == string(domain.WorkspaceRoleViewer)
}
func normalizeUser(in domain.SCIMUser) (domain.SCIMUser, error) {
	in.UserName = strings.ToLower(strings.TrimSpace(in.UserName))
	in.ExternalID = strings.TrimSpace(in.ExternalID)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if in.WorkspaceAccess != nil && strings.TrimSpace(in.WorkspaceAccess.Role) != "" {
		in.Role = in.WorkspaceAccess.Role
	}
	in.Role = strings.ToLower(strings.TrimSpace(in.Role))
	if in.Role == "" {
		in.Role = string(domain.WorkspaceRoleViewer)
	}
	if in.UserName == "" || !strings.Contains(in.UserName, "@") || !validManagedRole(in.Role) {
		return domain.SCIMUser{}, domain.ErrSCIMInvalid
	}
	in.Schemas = []string{domain.SCIMUserSchema, domain.SCIMWorkspaceAccessSchema}
	in.WorkspaceAccess = &domain.SCIMWorkspaceAccess{Role: in.Role, RetainedManualAccess: in.RetainedManualAccess}
	if len(in.Emails) == 0 {
		in.Emails = []domain.SCIMEmail{{Value: in.UserName, Type: "work", Primary: true}}
	}
	return in, nil
}
func normalizeGroup(in domain.SCIMGroup) (domain.SCIMGroup, error) {
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.ExternalID = strings.TrimSpace(in.ExternalID)
	if in.WorkspaceAccess != nil && strings.TrimSpace(in.WorkspaceAccess.Role) != "" {
		in.Role = in.WorkspaceAccess.Role
	}
	in.Role = strings.ToLower(strings.TrimSpace(in.Role))
	if in.Role == "" {
		in.Role = string(domain.WorkspaceRoleViewer)
	}
	if in.DisplayName == "" || !validManagedRole(in.Role) {
		return domain.SCIMGroup{}, domain.ErrSCIMInvalid
	}
	in.Schemas = []string{domain.SCIMGroupSchema, domain.SCIMWorkspaceAccessSchema}
	in.WorkspaceAccess = &domain.SCIMWorkspaceAccess{Role: in.Role}
	return in, nil
}
func (s *SCIMService) CreateUser(ctx context.Context, w uuid.UUID, in domain.SCIMUser) (domain.SCIMUser, error) {
	u, e := normalizeUser(in)
	if e != nil {
		return domain.SCIMUser{}, e
	}
	return s.repo.CreateUser(ctx, w, u, s.now().UTC())
}
func (s *SCIMService) GetUser(ctx context.Context, w, id uuid.UUID) (domain.SCIMUser, error) {
	return s.repo.GetUser(ctx, w, id)
}

var scimFilterPattern = regexp.MustCompile(`(?i)^\s*(userName|externalId|displayName|id)\s+eq\s+"([^"]+)"\s*$`)

func parseSCIMFilter(raw string, group bool) (domain.SCIMFilter, error) {
	if strings.TrimSpace(raw) == "" {
		return domain.SCIMFilter{}, nil
	}
	m := scimFilterPattern.FindStringSubmatch(raw)
	if m == nil {
		return domain.SCIMFilter{}, domain.ErrSCIMInvalid
	}
	attribute := strings.ToLower(m[1])
	if group && attribute == "username" || !group && attribute == "displayname" {
		return domain.SCIMFilter{}, domain.ErrSCIMInvalid
	}
	canonical := map[string]string{"username": "userName", "displayname": "displayName", "externalid": "externalId", "id": "id"}
	attribute, ok := canonical[attribute]
	if !ok {
		return domain.SCIMFilter{}, domain.ErrSCIMInvalid
	}
	return domain.SCIMFilter{Attribute: attribute, Value: m[2]}, nil
}
func pageBounds(start, count int) (int, int) {
	if start < 1 {
		start = 1
	}
	if count < 0 {
		count = 0
	}
	if count > domain.SCIMMaxPageSize {
		count = domain.SCIMMaxPageSize
	}
	return start, count
}
func (s *SCIMService) ListUsers(ctx context.Context, w uuid.UUID, filter string, start, count int) (domain.SCIMPage[domain.SCIMUser], error) {
	f, e := parseSCIMFilter(filter, false)
	if e != nil {
		return domain.SCIMPage[domain.SCIMUser]{}, e
	}
	start, count = pageBounds(start, count)
	return s.repo.ListUsers(ctx, w, f, start, count)
}
func (s *SCIMService) ReplaceUser(ctx context.Context, w, id uuid.UUID, etag string, in domain.SCIMUser) (domain.SCIMUser, error) {
	u, e := normalizeUser(in)
	if e != nil {
		return domain.SCIMUser{}, e
	}
	return s.repo.ReplaceUser(ctx, w, id, etag, u, s.now().UTC())
}
func (s *SCIMService) DeleteUser(ctx context.Context, w, id uuid.UUID, etag string) (domain.SCIMUser, error) {
	return s.repo.DeleteUser(ctx, w, id, etag, s.now().UTC())
}
func (s *SCIMService) AdoptUser(ctx context.Context, p domain.Principal, w, id uuid.UUID) (out domain.SCIMUser, e error) {
	e = s.ownerLocked(ctx, p, w, func(tx context.Context) error {
		out, e = s.repo.AdoptUser(tx, w, id, s.now().UTC())
		return e
	})
	return out, e
}

func patchString(value any) (string, bool) { v, ok := value.(string); return strings.TrimSpace(v), ok }
func patchBool(value any) (bool, bool)     { v, ok := value.(bool); return v, ok }
func (s *SCIMService) PatchUser(ctx context.Context, w, id uuid.UUID, etag string, patch domain.SCIMPatchRequest) (domain.SCIMUser, error) {
	if len(patch.Schemas) != 1 || patch.Schemas[0] != domain.SCIMPatchSchema || len(patch.Operations) == 0 {
		return domain.SCIMUser{}, domain.ErrSCIMInvalid
	}
	u, e := s.repo.GetUser(ctx, w, id)
	if e != nil {
		return domain.SCIMUser{}, e
	}
	for _, op := range patch.Operations {
		if strings.TrimSpace(op.Path) == "" {
			values, ok := op.Value.(map[string]any)
			if !ok || strings.ToLower(op.Operation) == "remove" || len(values) == 0 {
				return domain.SCIMUser{}, domain.ErrSCIMInvalid
			}
			for path, value := range values {
				var err error
				u, err = applyUserPatch(u, strings.ToLower(op.Operation), path, value)
				if err != nil {
					return domain.SCIMUser{}, err
				}
			}
			continue
		}
		var err error
		u, err = applyUserPatch(u, strings.ToLower(op.Operation), op.Path, op.Value)
		if err != nil {
			return domain.SCIMUser{}, err
		}
	}
	return s.ReplaceUser(ctx, w, id, etag, u)
}

func applyUserPatch(u domain.SCIMUser, kind, rawPath string, value any) (domain.SCIMUser, error) {
	kind = strings.ToLower(kind)
	path := strings.ToLower(strings.TrimSpace(rawPath))
	if kind != "add" && kind != "replace" && kind != "remove" {
		return domain.SCIMUser{}, domain.ErrSCIMInvalid
	}
	if path == strings.ToLower(domain.SCIMWorkspaceAccessSchema) {
		attributes, ok := value.(map[string]any)
		if !ok || len(attributes) != 1 {
			return domain.SCIMUser{}, domain.ErrSCIMInvalid
		}
		role, ok := attributes["role"]
		if !ok {
			return domain.SCIMUser{}, domain.ErrSCIMInvalid
		}
		return applyUserPatch(u, kind, domain.SCIMWorkspaceAccessSchema+":role", role)
	}
	switch path {
	case "active":
		v, ok := patchBool(value)
		if !ok || kind == "remove" {
			return domain.SCIMUser{}, domain.ErrSCIMInvalid
		}
		u.Active = v
	case "username":
		v, ok := patchString(value)
		if !ok || kind == "remove" {
			return domain.SCIMUser{}, domain.ErrSCIMInvalid
		}
		u.UserName = v
	case "displayname":
		if kind == "remove" {
			u.DisplayName = ""
		} else if v, ok := patchString(value); ok {
			u.DisplayName = v
		} else {
			return domain.SCIMUser{}, domain.ErrSCIMInvalid
		}
	case "externalid":
		if kind == "remove" {
			u.ExternalID = ""
		} else if v, ok := patchString(value); ok {
			u.ExternalID = v
		} else {
			return domain.SCIMUser{}, domain.ErrSCIMInvalid
		}
	case strings.ToLower(domain.SCIMWorkspaceAccessSchema + ":role"):
		v, ok := patchString(value)
		if !ok || kind == "remove" {
			return domain.SCIMUser{}, domain.ErrSCIMInvalid
		}
		u.Role = v
	default:
		return domain.SCIMUser{}, domain.ErrSCIMInvalid
	}
	return u, nil
}

func (s *SCIMService) CreateGroup(ctx context.Context, w uuid.UUID, in domain.SCIMGroup) (domain.SCIMGroup, error) {
	g, e := normalizeGroup(in)
	if e != nil {
		return domain.SCIMGroup{}, e
	}
	return s.repo.CreateGroup(ctx, w, g, s.now().UTC())
}
func (s *SCIMService) GetGroup(ctx context.Context, w, id uuid.UUID) (domain.SCIMGroup, error) {
	return s.repo.GetGroup(ctx, w, id)
}
func (s *SCIMService) ListGroups(ctx context.Context, w uuid.UUID, filter string, start, count int) (domain.SCIMPage[domain.SCIMGroup], error) {
	f, e := parseSCIMFilter(filter, true)
	if e != nil {
		return domain.SCIMPage[domain.SCIMGroup]{}, e
	}
	start, count = pageBounds(start, count)
	return s.repo.ListGroups(ctx, w, f, start, count)
}
func (s *SCIMService) ReplaceGroup(ctx context.Context, w, id uuid.UUID, etag string, in domain.SCIMGroup) (domain.SCIMGroup, error) {
	g, e := normalizeGroup(in)
	if e != nil {
		return domain.SCIMGroup{}, e
	}
	return s.repo.ReplaceGroup(ctx, w, id, etag, g, s.now().UTC())
}
func (s *SCIMService) DeleteGroup(ctx context.Context, w, id uuid.UUID, etag string) error {
	return s.repo.DeleteGroup(ctx, w, id, etag, s.now().UTC())
}
func patchMembers(value any) ([]domain.SCIMGroupMember, bool) {
	raw, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]domain.SCIMGroupMember, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := m["value"].(string)
		if !ok {
			return nil, false
		}
		if _, e := uuid.Parse(v); e != nil {
			return nil, false
		}
		out = append(out, domain.SCIMGroupMember{Value: v})
	}
	return out, true
}
func (s *SCIMService) PatchGroup(ctx context.Context, w, id uuid.UUID, etag string, patch domain.SCIMPatchRequest) (domain.SCIMGroup, error) {
	if len(patch.Schemas) != 1 || patch.Schemas[0] != domain.SCIMPatchSchema || len(patch.Operations) == 0 {
		return domain.SCIMGroup{}, domain.ErrSCIMInvalid
	}
	g, e := s.repo.GetGroup(ctx, w, id)
	if e != nil {
		return domain.SCIMGroup{}, e
	}
	for _, op := range patch.Operations {
		if strings.TrimSpace(op.Path) == "" {
			values, ok := op.Value.(map[string]any)
			if !ok || strings.ToLower(op.Operation) == "remove" || len(values) == 0 {
				return domain.SCIMGroup{}, domain.ErrSCIMInvalid
			}
			for path, value := range values {
				g, e = applyGroupPatch(g, op.Operation, path, value)
				if e != nil {
					return domain.SCIMGroup{}, e
				}
			}
			continue
		}
		g, e = applyGroupPatch(g, op.Operation, op.Path, op.Value)
		if e != nil {
			return domain.SCIMGroup{}, e
		}
	}
	return s.ReplaceGroup(ctx, w, id, etag, g)
}

var scimMemberFilterPath = regexp.MustCompile(`(?i)^members\s*\[\s*value\s+eq\s+"([0-9a-f-]+)"\s*\]$`)

func applyGroupPatch(g domain.SCIMGroup, kind, rawPath string, value any) (domain.SCIMGroup, error) {
	kind = strings.ToLower(kind)
	path := strings.ToLower(strings.TrimSpace(rawPath))
	if kind != "add" && kind != "replace" && kind != "remove" {
		return domain.SCIMGroup{}, domain.ErrSCIMInvalid
	}
	if path == strings.ToLower(domain.SCIMWorkspaceAccessSchema) {
		attributes, ok := value.(map[string]any)
		if !ok || len(attributes) != 1 {
			return domain.SCIMGroup{}, domain.ErrSCIMInvalid
		}
		role, ok := attributes["role"]
		if !ok {
			return domain.SCIMGroup{}, domain.ErrSCIMInvalid
		}
		return applyGroupPatch(g, kind, domain.SCIMWorkspaceAccessSchema+":role", role)
	}
	if match := scimMemberFilterPath.FindStringSubmatch(strings.TrimSpace(rawPath)); match != nil {
		if kind != "remove" {
			return domain.SCIMGroup{}, domain.ErrSCIMInvalid
		}
		id, err := uuid.Parse(match[1])
		if err != nil {
			return domain.SCIMGroup{}, domain.ErrSCIMInvalid
		}
		filtered := g.Members[:0]
		for _, member := range g.Members {
			if member.Value != id.String() {
				filtered = append(filtered, member)
			}
		}
		g.Members = filtered
		return g, nil
	}
	switch path {
	case "displayname":
		if kind == "remove" {
			return domain.SCIMGroup{}, domain.ErrSCIMInvalid
		}
		v, ok := patchString(value)
		if !ok {
			return domain.SCIMGroup{}, domain.ErrSCIMInvalid
		}
		g.DisplayName = v
	case "externalid":
		if kind == "remove" {
			g.ExternalID = ""
		} else if v, ok := patchString(value); ok {
			g.ExternalID = v
		} else {
			return domain.SCIMGroup{}, domain.ErrSCIMInvalid
		}
	case strings.ToLower(domain.SCIMWorkspaceAccessSchema + ":role"):
		if kind == "remove" {
			return domain.SCIMGroup{}, domain.ErrSCIMInvalid
		}
		v, ok := patchString(value)
		if !ok {
			return domain.SCIMGroup{}, domain.ErrSCIMInvalid
		}
		g.Role = v
	case "members":
		if kind == "remove" {
			g.Members = nil
		} else if v, ok := patchMembers(value); ok {
			if kind == "add" {
				g.Members = append(g.Members, v...)
			} else {
				g.Members = v
			}
		} else {
			return domain.SCIMGroup{}, domain.ErrSCIMInvalid
		}
	default:
		return domain.SCIMGroup{}, domain.ErrSCIMInvalid
	}
	return g, nil
}

var _ domain.SCIMUsecase = (*SCIMService)(nil)
