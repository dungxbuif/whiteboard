package postgres

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"devtool/internal/domain"
	"devtool/internal/usecase"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSCIMMigrationUpDown(t *testing.T) {
	db := accessTestDB(t, 26)
	up, e := os.ReadFile("../../../migrations/000027_workspace_scim.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	if e = db.Exec(string(up)).Error; e != nil {
		t.Fatal(e)
	}
	for _, table := range []string{"workspace_scim_credentials", "workspace_scim_users", "workspace_scim_groups", "workspace_scim_group_members", "workspace_membership_sources"} {
		var name *string
		if e = db.Raw("SELECT to_regclass(?)", table).Scan(&name).Error; e != nil || name == nil {
			t.Fatalf("missing %s: %v", table, e)
		}
	}
	down, _ := os.ReadFile("../../../migrations/000027_workspace_scim.down.sql")
	if e = db.Exec(string(down)).Error; e != nil {
		t.Fatal(e)
	}
}

func TestSCIMDeprovisionRevokesOnlyManagedWorkspaceAndPreservesManual(t *testing.T) {
	db := accessTestDB(t, 27)
	owner, workspace := domainRepositoryWorkspace(t, db, "SCIM")
	other := uuid.New()
	mustExec(t, db, "INSERT INTO workspaces(id,name,owner_id) VALUES (?,'Other',?)", other, owner)
	managed, manual := uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO users(id,email,name) VALUES (?,?, 'Managed'),(?,?, 'Manual')", managed, "managed@example.test", manual, "manual@example.test")
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer'),(?,?,'viewer')", workspace, manual, other, managed)
	repo := NewSCIMRepository(db)
	now := time.Now().UTC()
	created, e := repo.CreateUser(context.Background(), workspace, domain.SCIMUser{UserName: "managed@example.test", DisplayName: "Managed", Active: true, Role: "viewer", ExternalID: "m-1"}, now)
	if e != nil {
		t.Fatal(e)
	}
	retried, e := repo.CreateUser(context.Background(), workspace, domain.SCIMUser{UserName: "managed@example.test", DisplayName: "Managed", Active: true, Role: "viewer", ExternalID: "m-1"}, now)
	if e != nil || retried.ID != created.ID {
		t.Fatalf("idempotent retry=%+v err=%v", retried, e)
	}
	scene := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Active room')", scene, workspace)
	access, e := usecase.NewAccessService(NewAccessRepository(db), "../../../configs/casbin_model.conf")
	if e != nil {
		t.Fatal(e)
	}
	rooms, e := usecase.NewRoomService(NewShareLinkRepository(db), access, []byte("0123456789abcdef0123456789abcdef"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = rooms.MintRoomToken(context.Background(), domain.UserPrincipal(managed), domain.SceneRef{WorkspaceID: workspace, SceneID: scene}); e != nil {
		t.Fatalf("active room before deprovision: %v", e)
	}
	manualSCIM, e := repo.CreateUser(context.Background(), workspace, domain.SCIMUser{UserName: "manual@example.test", DisplayName: "Manual", Active: true, Role: "viewer", ExternalID: "m-2"}, now)
	if e != nil || !manualSCIM.RetainedManualAccess {
		t.Fatalf("manual=%+v err=%v", manualSCIM, e)
	}
	connection := uuid.New()
	mustExec(t, db, "INSERT INTO workspace_sso_connections(id,workspace_id,issuer,client_id,client_secret_cipher,mode,public_links_enabled) VALUES (?,?,?,?,?,'optional',false)", connection, workspace, "https://idp.example", "client", []byte("cipher"))
	session := domain.AuthSession{ID: uuid.New(), UserID: managed, Method: domain.AuthMethodOIDC, ConnectionID: &connection, WorkspaceID: &workspace, AuthenticatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if e = NewAuthSessionRepository(db).CreateSession(context.Background(), session); e != nil {
		t.Fatal(e)
	}
	deprovisioned, e := repo.DeleteUser(context.Background(), workspace, uuid.MustParse(created.ID), created.Meta.Version, now.Add(time.Minute))
	if e != nil || deprovisioned.Active {
		t.Fatalf("deleted=%+v err=%v", deprovisioned, e)
	}
	var target, otherCount, manualCount int64
	db.Table("workspace_members").Where("workspace_id=? AND user_id=?", workspace, managed).Count(&target)
	db.Table("workspace_members").Where("workspace_id=? AND user_id=?", other, managed).Count(&otherCount)
	if target != 0 || otherCount != 1 {
		t.Fatalf("target=%d other=%d", target, otherCount)
	}
	var aclVersion int64
	db.Raw("SELECT acl_version FROM workspaces WHERE id=?", workspace).Scan(&aclVersion)
	if aclVersion < 3 {
		t.Fatalf("deprovision did not invalidate current authority: acl_version=%d", aclVersion)
	}
	if _, e = rooms.MintRoomToken(context.Background(), domain.UserPrincipal(managed), domain.SceneRef{WorkspaceID: workspace, SceneID: scene}); e == nil {
		t.Fatal("deprovisioned user minted a new active-room token")
	}
	got, e := NewAuthSessionRepository(db).GetSession(context.Background(), session.ID)
	if e != nil || got.RevokedAt == nil {
		t.Fatalf("session=%+v err=%v", got, e)
	}
	if _, e = repo.DeleteUser(context.Background(), workspace, uuid.MustParse(manualSCIM.ID), manualSCIM.Meta.Version, now.Add(time.Minute)); e != nil {
		t.Fatal(e)
	}
	db.Table("workspace_members").Where("workspace_id=? AND user_id=?", workspace, manual).Count(&manualCount)
	if manualCount != 1 {
		t.Fatalf("manual access removed=%d", manualCount)
	}
}

func TestSCIMExplicitAdoptionMakesManualMembershipManaged(t *testing.T) {
	db := accessTestDB(t, 27)
	_, workspace := domainRepositoryWorkspace(t, db, "SCIM adoption")
	userID := uuid.New()
	mustExec(t, db, "INSERT INTO users(id,email,name) VALUES (?,?, 'Manual')", userID, "adopt@example.test")
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer')", workspace, userID)
	repo := NewSCIMRepository(db)
	now := time.Now().UTC()
	user, e := repo.CreateUser(context.Background(), workspace, domain.SCIMUser{UserName: "adopt@example.test", Active: true, Role: "editor", ExternalID: "adopt-1"}, now)
	if e != nil || !user.RetainedManualAccess {
		t.Fatalf("retained=%+v err=%v", user, e)
	}
	user, e = repo.AdoptUser(context.Background(), workspace, uuid.MustParse(user.ID), now.Add(time.Second))
	if e != nil || user.RetainedManualAccess {
		t.Fatalf("adopted=%+v err=%v", user, e)
	}
	user.Role = "viewer"
	user, e = repo.ReplaceUser(context.Background(), workspace, uuid.MustParse(user.ID), user.Meta.Version, user, now.Add(2*time.Second))
	if e != nil {
		t.Fatal(e)
	}
	var adoptedRole string
	db.Raw("SELECT role FROM workspace_members WHERE workspace_id=? AND user_id=?", workspace, userID).Scan(&adoptedRole)
	if adoptedRole != "viewer" {
		t.Fatalf("adopted downgrade role=%s", adoptedRole)
	}
	if _, e = repo.DeleteUser(context.Background(), workspace, uuid.MustParse(user.ID), user.Meta.Version, now.Add(time.Minute)); e != nil {
		t.Fatal(e)
	}
	var count int64
	db.Table("workspace_members").Where("workspace_id=? AND user_id=?", workspace, userID).Count(&count)
	if count != 0 {
		t.Fatalf("adopted membership retained=%d", count)
	}
}

func TestSCIMGroupStrongestRoleAndETag(t *testing.T) {
	db := accessTestDB(t, 27)
	_, workspace := domainRepositoryWorkspace(t, db, "SCIM group")
	repo := NewSCIMRepository(db)
	now := time.Now().UTC()
	user, e := repo.CreateUser(context.Background(), workspace, domain.SCIMUser{UserName: "group@example.test", Active: true, Role: "viewer"}, now)
	if e != nil {
		t.Fatal(e)
	}
	empty, e := repo.ListUsers(context.Background(), workspace, domain.SCIMFilter{}, 1, 0)
	if e != nil || empty.TotalResults != 1 || empty.ItemsPerPage != 0 || len(empty.Resources) != 0 {
		t.Fatalf("count zero page=%+v err=%v", empty, e)
	}
	group, e := repo.CreateGroup(context.Background(), workspace, domain.SCIMGroup{DisplayName: "Editors", Role: "editor", Members: []domain.SCIMGroupMember{{Value: user.ID}}}, now)
	if e != nil {
		t.Fatal(e)
	}
	var role string
	db.Raw("SELECT role FROM workspace_members m JOIN workspace_scim_users s ON s.user_id=m.user_id WHERE s.id=?", user.ID).Scan(&role)
	if role != "editor" {
		t.Fatalf("role=%s", role)
	}
	group.DisplayName = "Changed"
	if _, e = repo.ReplaceGroup(context.Background(), workspace, uuid.MustParse(group.ID), `W/"999"`, group, now); e != domain.ErrSCIMPrecondition {
		t.Fatalf("stale etag=%v", e)
	}
	group.Role = "owner"
	if _, e = repo.ReplaceGroup(context.Background(), workspace, uuid.MustParse(group.ID), group.Meta.Version, group, now); e == nil {
		t.Fatal("owner group accepted")
	}
	if e = repo.DeleteGroup(context.Background(), workspace, uuid.MustParse(group.ID), group.Meta.Version, now.Add(time.Second)); e != nil {
		t.Fatal(e)
	}
	db.Raw("SELECT role FROM workspace_members m JOIN workspace_scim_users s ON s.user_id=m.user_id WHERE s.id=?", user.ID).Scan(&role)
	if role != "viewer" {
		t.Fatalf("role after group delete=%s", role)
	}
	var groupSources int64
	db.Table("workspace_membership_sources").Where("workspace_id=? AND source_kind='group' AND source_id=?", workspace, group.ID).Count(&groupSources)
	if groupSources != 0 {
		t.Fatalf("orphan group sources=%d", groupSources)
	}
}

func TestSCIMLateManualAndInvitationProvenanceSurvivesDeprovision(t *testing.T) {
	db := accessTestDB(t, 27)
	owner, workspace := domainRepositoryWorkspace(t, db, "SCIM late manual")
	other := uuid.New()
	mustExec(t, db, "INSERT INTO workspaces(id,name,owner_id) VALUES (?,'Other',?)", other, owner)
	repo := NewSCIMRepository(db)
	now := time.Now().UTC()
	manual, e := repo.CreateUser(context.Background(), workspace, domain.SCIMUser{UserName: "late-manual@example.test", Active: true, Role: "viewer", ExternalID: "late-manual"}, now)
	if e != nil {
		t.Fatal(e)
	}
	var manualRow scimUserRow
	if e = db.Where("id=?", manual.ID).First(&manualRow).Error; e != nil {
		t.Fatal(e)
	}
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer')", other, manualRow.UserID)
	editor := domain.WorkspaceRoleEditor
	if e = NewAccessRepository(db).ChangeMember(context.Background(), workspace, manualRow.UserID, &editor); e != nil {
		t.Fatal(e)
	}
	if _, e = repo.DeleteUser(context.Background(), workspace, uuid.MustParse(manual.ID), manual.Meta.Version, now.Add(time.Minute)); e != nil {
		t.Fatal(e)
	}
	var role string
	db.Raw("SELECT role FROM workspace_members WHERE workspace_id=? AND user_id=?", workspace, manualRow.UserID).Scan(&role)
	if role != "editor" {
		t.Fatalf("late manual role=%s", role)
	}
	var otherCount int64
	db.Table("workspace_members").Where("workspace_id=? AND user_id=?", other, manualRow.UserID).Count(&otherCount)
	if otherCount != 1 {
		t.Fatalf("other workspace membership=%d", otherCount)
	}

	invited, e := repo.CreateUser(context.Background(), workspace, domain.SCIMUser{UserName: "late-invite@example.test", Active: true, Role: "viewer", ExternalID: "late-invite"}, now)
	if e != nil {
		t.Fatal(e)
	}
	var invitedRow scimUserRow
	if e = db.Where("id=?", invited.ID).First(&invitedRow).Error; e != nil {
		t.Fatal(e)
	}
	invitation := domain.Invitation{ID: uuid.New(), WorkspaceID: workspace, Email: invited.UserName, Role: domain.WorkspaceRoleEditor, InviterID: owner, TokenHash: strings.Repeat("a", 64), CreatedAt: now, ExpiresAt: now.Add(168 * time.Hour)}
	if e = db.Create(&invitation).Error; e != nil {
		t.Fatal(e)
	}
	if _, e = NewInvitationRepository(db).Join(context.Background(), &invitation, invitedRow.UserID, now.Add(time.Second)); e != nil {
		t.Fatal(e)
	}
	if _, e = repo.DeleteUser(context.Background(), workspace, uuid.MustParse(invited.ID), invited.Meta.Version, now.Add(time.Minute)); e != nil {
		t.Fatal(e)
	}
	role = ""
	db.Raw("SELECT role FROM workspace_members WHERE workspace_id=? AND user_id=?", workspace, invitedRow.UserID).Scan(&role)
	if role != "editor" {
		t.Fatalf("late invitation role=%s", role)
	}
}

func TestSCIMRepositoryDoesNotLogProvisionedPII(t *testing.T) {
	db := accessTestDB(t, 27)
	_, workspace := domainRepositoryWorkspace(t, db, "SCIM logging")
	var captured bytes.Buffer
	loggingDB := db.Session(&gorm.Session{Logger: logger.New(log.New(&captured, "", 0), logger.Config{LogLevel: logger.Info})})
	repo := NewSCIMRepository(loggingDB)
	pii := "private.person+scim@example.test"
	if _, e := repo.CreateUser(context.Background(), workspace, domain.SCIMUser{UserName: pii, DisplayName: "Private Person", Active: true, Role: "viewer", ExternalID: "employee-secret-42"}, time.Now().UTC()); e != nil {
		t.Fatal(e)
	}
	logged := captured.String()
	for _, secret := range []string{pii, "Private Person", "employee-secret-42"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("SCIM PII leaked to SQL log: %q", secret)
		}
	}
}
