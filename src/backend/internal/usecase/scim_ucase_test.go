package usecase

import (
	"context"
	"testing"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
)

type scimAccessStub struct{ owner uuid.UUID }

func (s scimAccessStub) RequireOwner(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (s scimAccessStub) WithWorkspaceLock(ctx context.Context, _ uuid.UUID, _ uuid.UUID, fn func(context.Context) error) error {
	return fn(ctx)
}

type scimRepoStub struct {
	created      domain.SCIMUser
	replaced     domain.SCIMUser
	filter       domain.SCIMFilter
	start, count int
	credential   domain.SCIMCredential
}

func (s *scimRepoStub) ResolveCredential(context.Context, []byte) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (s *scimRepoStub) CredentialStatus(context.Context, uuid.UUID) (domain.SCIMTokenStatus, error) {
	return domain.SCIMTokenStatus{}, nil
}
func (s *scimRepoStub) RotateCredential(_ context.Context, c domain.SCIMCredential) error {
	s.credential = c
	return nil
}
func (s *scimRepoStub) RevokeCredential(context.Context, uuid.UUID, time.Time) error { return nil }
func (s *scimRepoStub) CreateUser(_ context.Context, _ uuid.UUID, u domain.SCIMUser, _ time.Time) (domain.SCIMUser, error) {
	s.created = u
	u.ID = uuid.NewString()
	return u, nil
}
func (s *scimRepoStub) GetUser(context.Context, uuid.UUID, uuid.UUID) (domain.SCIMUser, error) {
	return domain.SCIMUser{ID: uuid.NewString(), UserName: "a@example.test", Active: true, Role: "viewer", Meta: domain.SCIMMeta{Version: `W/"2"`}}, nil
}
func (s *scimRepoStub) ListUsers(_ context.Context, _ uuid.UUID, f domain.SCIMFilter, start, count int) (domain.SCIMPage[domain.SCIMUser], error) {
	s.filter, s.start, s.count = f, start, count
	return domain.SCIMPage[domain.SCIMUser]{}, nil
}
func (s *scimRepoStub) ReplaceUser(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string, u domain.SCIMUser, _ time.Time) (domain.SCIMUser, error) {
	s.replaced = u
	return u, nil
}
func (s *scimRepoStub) DeleteUser(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (domain.SCIMUser, error) {
	return domain.SCIMUser{}, nil
}
func (s *scimRepoStub) AdoptUser(context.Context, uuid.UUID, uuid.UUID, time.Time) (domain.SCIMUser, error) {
	return domain.SCIMUser{}, nil
}
func (s *scimRepoStub) CreateGroup(context.Context, uuid.UUID, domain.SCIMGroup, time.Time) (domain.SCIMGroup, error) {
	return domain.SCIMGroup{}, nil
}
func (s *scimRepoStub) GetGroup(context.Context, uuid.UUID, uuid.UUID) (domain.SCIMGroup, error) {
	return domain.SCIMGroup{}, nil
}
func (s *scimRepoStub) ListGroups(context.Context, uuid.UUID, domain.SCIMFilter, int, int) (domain.SCIMPage[domain.SCIMGroup], error) {
	return domain.SCIMPage[domain.SCIMGroup]{}, nil
}
func (s *scimRepoStub) ReplaceGroup(context.Context, uuid.UUID, uuid.UUID, string, domain.SCIMGroup, time.Time) (domain.SCIMGroup, error) {
	return domain.SCIMGroup{}, nil
}
func (s *scimRepoStub) DeleteGroup(context.Context, uuid.UUID, uuid.UUID, string, time.Time) error {
	return nil
}

func TestSCIMPagingFiltersAndRoleValidation(t *testing.T) {
	repo := &scimRepoStub{}
	service := NewSCIMService(repo, scimAccessStub{})
	workspace := uuid.New()
	if _, err := service.ListUsers(context.Background(), workspace, `userName eq "A@Example.Test"`, 0, 1000); err != nil {
		t.Fatal(err)
	}
	if repo.start != 1 || repo.count != 100 || repo.filter.Attribute != "userName" || repo.filter.Value != "A@Example.Test" {
		t.Fatalf("paging/filter=%+v %d %d", repo.filter, repo.start, repo.count)
	}
	if _, err := service.ListUsers(context.Background(), workspace, `displayName co "x"`, 1, 10); err != domain.ErrSCIMInvalid {
		t.Fatalf("invalid filter=%v", err)
	}
	if _, err := service.CreateUser(context.Background(), workspace, domain.SCIMUser{UserName: "a@example.test", Active: true, Role: "owner"}); err != domain.ErrSCIMInvalid {
		t.Fatalf("owner role=%v", err)
	}
}

func TestSCIMPatchAddReplaceRemove(t *testing.T) {
	repo := &scimRepoStub{}
	service := NewSCIMService(repo, scimAccessStub{})
	id := uuid.New()
	workspace := uuid.New()
	patch := domain.SCIMPatchRequest{Schemas: []string{domain.SCIMPatchSchema}, Operations: []domain.SCIMPatchOperation{{Operation: "replace", Path: "active", Value: false}, {Operation: "replace", Path: "role", Value: "editor"}}}
	if _, err := service.PatchUser(context.Background(), workspace, id, `W/"2"`, patch); err != nil {
		t.Fatal(err)
	}
	if repo.replaced.Active || repo.replaced.Role != "editor" {
		t.Fatalf("patched=%+v", repo.replaced)
	}
	bad := domain.SCIMPatchRequest{Schemas: []string{domain.SCIMPatchSchema}, Operations: []domain.SCIMPatchOperation{{Operation: "replace", Path: "unknown", Value: true}}}
	if _, err := service.PatchUser(context.Background(), workspace, id, `W/"2"`, bad); err != domain.ErrSCIMInvalid {
		t.Fatalf("bad patch=%v", err)
	}
}

func TestSCIMTokenRotationHashesCredential(t *testing.T) {
	repo := &scimRepoStub{}
	service := NewSCIMService(repo, scimAccessStub{})
	user, workspace := uuid.New(), uuid.New()
	p := domain.UserPrincipal(user)
	result, err := service.RotateToken(context.Background(), p, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Token) < 40 || len(repo.credential.TokenHash) != 32 || repo.credential.WorkspaceID != workspace || repo.credential.CreatedBy != user {
		t.Fatalf("token=%+v credential=%+v", result, repo.credential)
	}
}
