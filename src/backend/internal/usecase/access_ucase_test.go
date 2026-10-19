package usecase

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"github.com/google/uuid"
	"testing"
)

type accessStateRepo struct {
	domain.AccessRepository
	state domain.AccessState
	err   error
}

func (r accessStateRepo) WorkspaceState(context.Context, uuid.UUID, uuid.UUID) (domain.AccessState, error) {
	return r.state, r.err
}
func (r accessStateRepo) SceneState(context.Context, uuid.UUID, domain.SceneRef) (domain.AccessState, error) {
	return r.state, r.err
}
func TestResolveAccessMatrix(t *testing.T) {
	user := uuid.New()
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	for _, tc := range []struct {
		name           string
		role, override domain.WorkspaceRole
		access         string
		owner          bool
		err            error
	}{
		{"owner edit/manage", domain.WorkspaceRoleOwner, "", "edit", true, nil},
		{"viewer read only", domain.WorkspaceRoleViewer, "", "view", false, nil},
		{"editor cannot manage", domain.WorkspaceRoleEditor, "", "edit", false, nil},
		{"scene downgrade wins", domain.WorkspaceRoleEditor, domain.WorkspaceRoleViewer, "view", false, nil},
		{"scene upgrade wins", domain.WorkspaceRoleViewer, domain.WorkspaceRoleEditor, "edit", false, nil},
		{"owner remains owner", domain.WorkspaceRoleOwner, domain.WorkspaceRoleViewer, "edit", true, nil},
		{"removed member override denies", "", domain.WorkspaceRoleEditor, "", false, domain.ErrInaccessible},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, e := NewAccessService(accessStateRepo{state: domain.AccessState{Role: tc.role, Override: tc.override, ACLVersion: 42}, err: tc.err}, "../../configs/casbin_model.conf")
			if e != nil {
				t.Fatal(e)
			}
			d, e := service.Resolve(context.Background(), domain.UserPrincipal(user), ref)
			if tc.err != nil {
				if !errors.Is(e, tc.err) {
					t.Fatalf("got %v", e)
				}
				return
			}
			if e != nil || d.Access != tc.access || d.Owner != tc.owner || d.ACLVersion != 42 {
				t.Fatalf("decision=%+v err=%v", d, e)
			}
			e = service.RequireOwner(context.Background(), user, ref.WorkspaceID)
			if (e == nil) != tc.owner {
				t.Fatalf("owner requirement: %v", e)
			}
		})
	}
	service, _ := NewAccessService(accessStateRepo{state: domain.AccessState{Role: domain.WorkspaceRoleOwner, ACLVersion: 1}})
	for _, p := range []domain.Principal{{Kind: domain.PrincipalPublic, ID: user.String(), UserID: &user}, {Kind: domain.PrincipalAgent, ID: user.String(), UserID: &user}, {Kind: domain.PrincipalUser, ID: uuid.NewString(), UserID: &user}, {Kind: domain.PrincipalUser, ID: user.String()}} {
		if _, e := service.Resolve(context.Background(), p, ref); !errors.Is(e, domain.ErrAccessDenied) {
			t.Fatalf("unnormalized principal accepted: %+v %v", p, e)
		}
	}
}
