package postgres

import (
	"context"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func featureFixture(t *testing.T) (*FeatureRepository, *AccessRepository, uuid.UUID, uuid.UUID, domain.SceneRef) {
	t.Helper()
	db := accessTestDB(t, 14)
	owner, viewer, workspace := accessFixture(t, db)
	scene := uuid.New()
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer')", workspace, viewer)
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Feature')", scene, workspace)
	return NewFeatureRepository(db), NewAccessRepository(db), owner, viewer, domain.SceneRef{WorkspaceID: workspace, SceneID: scene}
}

func TestMetadataRetryBinding(t *testing.T) {
	repo, access, owner, _, ref := featureFixture(t)
	requestID := uuid.New()
	principal := domain.UserPrincipal(owner)
	callbackCalls := 0
	first := domain.StoredFeatureResponse{Status: 201, Body: []byte(`{"id":"first"}`)}
	var got domain.StoredFeatureResponse
	err := access.WithWorkspaceLock(context.Background(), ref.WorkspaceID, owner, func(ctx context.Context) error {
		var err error
		got, err = repo.WithMutation(ctx, principal, ref, domain.FeatureCommentWrite, requestID, strings.Repeat("a", domain.FeatureRequestHashLength), func(context.Context) (domain.StoredFeatureResponse, error) {
			callbackCalls++
			return first, nil
		})
		return err
	})
	if err != nil || got.Status != first.Status || string(got.Body) != string(first.Body) {
		t.Fatalf("first mutation: got=%+v err=%v", got, err)
	}
	err = access.WithWorkspaceLock(context.Background(), ref.WorkspaceID, owner, func(ctx context.Context) error {
		var err error
		got, err = repo.WithMutation(ctx, principal, ref, domain.FeatureCommentWrite, requestID, strings.Repeat("a", domain.FeatureRequestHashLength), func(context.Context) (domain.StoredFeatureResponse, error) {
			callbackCalls++
			return domain.StoredFeatureResponse{}, errors.New("must not run")
		})
		return err
	})
	if err != nil || callbackCalls != 1 || string(got.Body) != string(first.Body) {
		t.Fatalf("exact retry: calls=%d got=%+v err=%v", callbackCalls, got, err)
	}
	err = access.WithWorkspaceLock(context.Background(), ref.WorkspaceID, owner, func(ctx context.Context) error {
		_, err := repo.WithMutation(ctx, principal, ref, domain.FeatureCommentWrite, requestID, strings.Repeat("b", domain.FeatureRequestHashLength), func(context.Context) (domain.StoredFeatureResponse, error) {
			return first, nil
		})
		return err
	})
	if !errors.Is(err, domain.ErrRequestIDReused) {
		t.Fatalf("changed retry accepted: %v", err)
	}
	bindings := []struct {
		name      string
		principal domain.Principal
		ref       domain.SceneRef
		action    domain.FeatureAction
	}{
		{"principal", domain.UserPrincipal(uuid.New()), ref, domain.FeatureCommentWrite},
		{"action", principal, ref, domain.FeatureVersionCreate},
		{"scene", principal, domain.SceneRef{WorkspaceID: ref.WorkspaceID, SceneID: uuid.New()}, domain.FeatureCommentWrite},
	}
	for _, binding := range bindings {
		t.Run("rejects changed "+binding.name, func(t *testing.T) {
			err := access.WithWorkspaceLock(context.Background(), ref.WorkspaceID, owner, func(ctx context.Context) error {
				_, callErr := repo.WithMutation(ctx, binding.principal, binding.ref, binding.action, requestID, strings.Repeat("a", domain.FeatureRequestHashLength), func(context.Context) (domain.StoredFeatureResponse, error) {
					t.Fatal("callback ran for mismatched retry binding")
					return first, nil
				})
				return callErr
			})
			if !errors.Is(err, domain.ErrRequestIDReused) {
				t.Fatalf("changed %s accepted: %v", binding.name, err)
			}
		})
	}
}

func TestFeatureMutationCallbackRollback(t *testing.T) {
	repo, access, owner, _, ref := featureFixture(t)
	sentinel := errors.New("feature callback failed")
	err := access.WithWorkspaceLock(context.Background(), ref.WorkspaceID, owner, func(ctx context.Context) error {
		_, callErr := repo.WithMutation(ctx, domain.UserPrincipal(owner), ref, domain.FeatureCommentWrite, uuid.New(), strings.Repeat("a", domain.FeatureRequestHashLength), func(callCtx context.Context) (domain.StoredFeatureResponse, error) {
			enabled := true
			if _, updateErr := repo.UpdatePolicy(callCtx, ref.WorkspaceID, 1, domain.WorkspaceFeaturePolicyPatch{CommentsEnabled: &enabled}); updateErr != nil {
				return domain.StoredFeatureResponse{}, updateErr
			}
			return domain.StoredFeatureResponse{}, sentinel
		})
		return callErr
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("callback error lost: %v", err)
	}
	policy, err := repo.Policy(context.Background(), ref.WorkspaceID)
	if err != nil || policy.Revision != 1 || policy.CommentsEnabled {
		t.Fatalf("callback state committed: policy=%+v err=%v", policy, err)
	}
	var receipts int64
	if err := repo.db.Table("feature_mutation_receipts").Where("workspace_id=?", ref.WorkspaceID).Count(&receipts).Error; err != nil || receipts != 0 {
		t.Fatalf("callback receipt committed: count=%d err=%v", receipts, err)
	}
}

func TestFeaturePolicyMatrixPostgres(t *testing.T) {
	repo, access, owner, viewer, ref := featureFixture(t)
	editor := uuid.New()
	mustExec(t, repo.db, "INSERT INTO users(id,email,name) VALUES (?,?,'Editor')", editor, editor.String()+"@example.test")
	mustExec(t, repo.db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'editor')", ref.WorkspaceID, editor)
	mustExec(t, repo.db, `UPDATE workspace_feature_policies SET comments_enabled=true,activity_enabled=true,versions_enabled=true,libraries_enabled=true,collections_enabled=true,presentation_enabled=true,screen_share_enabled=true WHERE workspace_id=?`, ref.WorkspaceID)
	accessService, err := usecase.NewAccessService(access)
	if err != nil {
		t.Fatal(err)
	}
	service := usecase.NewFeatureService(repo, accessService)
	roles := []struct {
		name      string
		principal domain.Principal
		allowed   map[domain.FeatureAction]bool
	}{
		{"owner", domain.UserPrincipal(owner), map[domain.FeatureAction]bool{
			domain.FeatureCommentRead: true, domain.FeatureCommentWrite: true, domain.FeatureCommentModerate: true, domain.FeatureActivityRead: true,
			domain.FeatureVersionRead: true, domain.FeatureVersionCreate: true, domain.FeatureVersionRestore: true, domain.FeatureLibraryRead: true,
			domain.FeatureLibraryWrite: true, domain.FeatureCollectionManage: true, domain.FeaturePresentationHost: true, domain.FeatureScreenPublish: true,
			domain.FeatureEnterpriseManage: true,
		}},
		{"editor", domain.UserPrincipal(editor), map[domain.FeatureAction]bool{
			domain.FeatureCommentRead: true, domain.FeatureCommentWrite: true, domain.FeatureActivityRead: true, domain.FeatureVersionRead: true,
			domain.FeatureVersionCreate: true, domain.FeatureVersionRestore: true, domain.FeatureLibraryRead: true, domain.FeatureCollectionManage: true,
			domain.FeaturePresentationHost: true, domain.FeatureScreenPublish: true,
		}},
		{"viewer", domain.UserPrincipal(viewer), map[domain.FeatureAction]bool{
			domain.FeatureCommentRead: true, domain.FeatureCommentWrite: true, domain.FeatureActivityRead: true, domain.FeatureVersionRead: true, domain.FeatureLibraryRead: true,
		}},
	}
	actions := []domain.FeatureAction{domain.FeatureCommentRead, domain.FeatureCommentWrite, domain.FeatureCommentModerate, domain.FeatureActivityRead, domain.FeatureVersionRead, domain.FeatureVersionCreate, domain.FeatureVersionRestore, domain.FeatureLibraryRead, domain.FeatureLibraryWrite, domain.FeatureCollectionManage, domain.FeaturePresentationHost, domain.FeatureScreenPublish, domain.FeatureEnterpriseManage}
	for _, role := range roles {
		for _, action := range actions {
			err := service.Require(context.Background(), role.principal, ref, action)
			if (err == nil) != role.allowed[action] {
				t.Fatalf("role=%s action=%s allowed=%v err=%v", role.name, action, role.allowed[action], err)
			}
		}
	}
}

func TestFeatureRevokeDuringMutation(t *testing.T) {
	repo, access, owner, viewer, ref := featureFixture(t)
	requestID := uuid.New()
	principal := domain.UserPrincipal(viewer)
	err := access.WithWorkspaceLock(context.Background(), ref.WorkspaceID, owner, func(ctx context.Context) error {
		if err := access.ChangeMember(ctx, ref.WorkspaceID, viewer, nil); err != nil {
			return err
		}
		accessService, serviceErr := usecase.NewAccessService(access)
		if serviceErr != nil {
			return serviceErr
		}
		service := usecase.NewFeatureService(repo, accessService)
		_, err := service.Mutate(ctx, principal, ref, domain.FeatureCommentWrite, requestID, []byte(`{"body":"late"}`), func(context.Context) (domain.StoredFeatureResponse, error) {
			t.Fatal("callback ran after revoke")
			return domain.StoredFeatureResponse{}, nil
		})
		return err
	})
	if !errors.Is(err, domain.ErrInaccessible) && !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("revoked mutation accepted: %v", err)
	}
}

func TestFeaturePolicyRetryKeepsDecimalRevision(t *testing.T) {
	repo, access, owner, _, ref := featureFixture(t)
	accessService, err := usecase.NewAccessService(access)
	if err != nil {
		t.Fatal(err)
	}
	service := usecase.NewFeatureService(repo, accessService)
	requestID := uuid.New()
	enabled := true
	patch := domain.WorkspaceFeaturePolicyPatch{CommentsEnabled: &enabled}
	first, err := service.UpdatePolicy(context.Background(), domain.UserPrincipal(owner), ref.WorkspaceID, 1, requestID, patch)
	if err != nil || first.Revision != 2 || !first.CommentsEnabled {
		t.Fatalf("first update=%+v err=%v", first, err)
	}
	retry, err := service.UpdatePolicy(context.Background(), domain.UserPrincipal(owner), ref.WorkspaceID, 1, requestID, patch)
	if err != nil || retry.Revision != 2 || !retry.CommentsEnabled {
		t.Fatalf("retry update=%+v err=%v", retry, err)
	}
}

func TestEnterpriseFeatureMigrationUpDown(t *testing.T) {
	db := accessTestDB(t, 13)
	up, err := os.ReadFile("../../../migrations/000014_enterprise_feature_policy.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(string(up)).Error; err != nil {
		t.Fatal(err)
	}
	_, _, workspace := accessFixture(t, db)
	var revision int64
	if err := db.Raw("SELECT revision FROM workspace_feature_policies WHERE workspace_id=?", workspace).Scan(&revision).Error; err != nil || revision != 1 {
		t.Fatalf("new workspace policy revision=%d err=%v", revision, err)
	}
	if !db.Migrator().HasColumn("share_links", "allow_comments") {
		t.Fatal("share-link comment policy column missing")
	}
	down, err := os.ReadFile("../../../migrations/000014_enterprise_feature_policy.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(string(down)).Error; err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable("workspace_feature_policies") || db.Migrator().HasTable("feature_mutation_receipts") || db.Migrator().HasColumn("share_links", "allow_comments") {
		t.Fatal("feature migration rollback incomplete")
	}
}

func TestPublicCommentPolicyUsesCurrentShareRevision(t *testing.T) {
	shareRepo, links, rooms, accessService, owner, ref := publicLinkFixture(t)
	mustExec(t, shareRepo.db, "UPDATE workspace_feature_policies SET comments_enabled=true WHERE workspace_id=?", ref.WorkspaceID)
	featureService := usecase.NewFeatureService(NewFeatureRepository(shareRepo.db), accessService)
	allowedLink, err := links.Create(context.Background(), owner, ref, domain.CreateShareLink{Access: domain.AccessView, AllowComments: true})
	if err != nil {
		t.Fatal(err)
	}
	allowedSession, err := rooms.ExchangePublicSession(context.Background(), allowedLink.Token, "", "Commenter")
	if err != nil {
		t.Fatal(err)
	}
	allowedIdentity, err := rooms.ParseSceneSession(allowedSession.Token)
	if err != nil {
		t.Fatal(err)
	}
	if err := featureService.Require(context.Background(), allowedIdentity.Principal, ref, domain.FeatureCommentWrite); err != nil {
		t.Fatalf("comment-enabled view link denied: %v", err)
	}
	if err := featureService.Require(context.Background(), allowedIdentity.Principal, ref, domain.FeatureVersionRestore); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("comment-enabled link gained version restore: %v", err)
	}
	viewLink, err := links.Create(context.Background(), owner, ref, domain.CreateShareLink{Access: domain.AccessView})
	if err != nil {
		t.Fatal(err)
	}
	viewSession, err := rooms.ExchangePublicSession(context.Background(), viewLink.Token, "", "Viewer")
	if err != nil {
		t.Fatal(err)
	}
	viewIdentity, err := rooms.ParseSceneSession(viewSession.Token)
	if err != nil {
		t.Fatal(err)
	}
	if err := featureService.Require(context.Background(), viewIdentity.Principal, ref, domain.FeatureCommentWrite); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("default public comments unexpectedly allowed: %v", err)
	}
}
