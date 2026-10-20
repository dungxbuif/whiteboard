package postgres

import (
	"bytes"
	"context"
	"devtool/internal/domain"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func domainRepositoryWorkspace(t *testing.T, db *gorm.DB, label string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	owner, workspace := uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO users(id,email,name) VALUES (?,?,?)", owner, owner.String()+"@example.test", label)
	mustExec(t, db, "INSERT INTO workspaces(id,name,owner_id) VALUES (?,?,?)", workspace, label, owner)
	return owner, workspace
}

func pendingDomain(workspace uuid.UUID, name string, now time.Time) domain.WorkspaceDomain {
	return domain.WorkspaceDomain{ID: uuid.New(), WorkspaceID: workspace, Domain: name, Status: domain.DomainPending, ChallengeHash: bytes.Repeat([]byte{7}, 32), ChallengeExpiresAt: now.Add(24 * time.Hour), Revision: 1, CreatedAt: now, UpdatedAt: now}
}

func createDomainLocked(t *testing.T, access *AccessRepository, repo *DomainRepository, owner uuid.UUID, value domain.WorkspaceDomain) (domain.WorkspaceDomain, error) {
	t.Helper()
	var created domain.WorkspaceDomain
	err := access.WithWorkspaceLock(context.Background(), value.WorkspaceID, owner, func(ctx domain.Context) error {
		var inner error
		created, inner = repo.Create(ctx, value)
		return inner
	})
	return created, err
}

func TestDomainRepositoryExpiredPendingClaimReclaimedAtomically(t *testing.T) {
	db := accessTestDB(t, 24)
	owner1, workspace1 := domainRepositoryWorkspace(t, db, "First")
	owner2, workspace2 := domainRepositoryWorkspace(t, db, "Second")
	owner3, workspace3 := domainRepositoryWorkspace(t, db, "Third")
	now := time.Now().UTC().Add(-48 * time.Hour)
	repo := NewDomainRepository(db)
	access := NewAccessRepository(db)
	active, err := createDomainLocked(t, access, repo, owner1, pendingDomain(workspace1, "active.example", time.Now().UTC()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = createDomainLocked(t, access, repo, owner2, pendingDomain(workspace2, active.Domain, time.Now().UTC().Add(72*time.Hour))); !errors.Is(err, domain.ErrDomainConflict) {
		t.Fatalf("active pending takeover=%v", err)
	}
	first, err := createDomainLocked(t, access, repo, owner1, pendingDomain(workspace1, "example.com", now))
	if err != nil {
		t.Fatal(err)
	}
	secondValue := pendingDomain(workspace2, "example.com", time.Now().UTC())
	second, err := createDomainLocked(t, access, repo, owner2, secondValue)
	if err != nil {
		t.Fatalf("expired reclaim: %v", err)
	}
	if second.WorkspaceID != workspace2 || second.ID != secondValue.ID || second.ID == first.ID {
		t.Fatalf("reclaimed=%+v", second)
	}
	if _, err = repo.Get(context.Background(), workspace1, first.ID); !errors.Is(err, domain.ErrInaccessible) {
		t.Fatalf("former owner still resolves claim: %v", err)
	}
	err = access.WithWorkspaceLock(context.Background(), workspace2, owner2, func(ctx domain.Context) error {
		current, e := repo.Get(ctx, workspace2, second.ID)
		if e != nil {
			return e
		}
		_, e = repo.Transition(ctx, workspace2, second.ID, current.Revision, domain.DomainVerified, time.Now().UTC())
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = createDomainLocked(t, access, repo, owner3, pendingDomain(workspace3, "example.com", time.Now().UTC().Add(72*time.Hour))); !errors.Is(err, domain.ErrDomainConflict) {
		t.Fatalf("verified takeover=%v", err)
	}
	err = access.WithWorkspaceLock(context.Background(), workspace2, owner2, func(ctx domain.Context) error {
		current, e := repo.Get(ctx, workspace2, second.ID)
		if e != nil {
			return e
		}
		_, e = repo.Transition(ctx, workspace2, second.ID, current.Revision, domain.DomainCaptured, time.Now().UTC())
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = createDomainLocked(t, access, repo, owner3, pendingDomain(workspace3, "example.com", time.Now().UTC().Add(96*time.Hour))); !errors.Is(err, domain.ErrDomainConflict) {
		t.Fatalf("captured takeover=%v", err)
	}
}

func TestDomainRepositoryExpiredClaimRaceHasOneWinner(t *testing.T) {
	db := accessTestDB(t, 24)
	owner1, workspace1 := domainRepositoryWorkspace(t, db, "Original")
	owner2, workspace2 := domainRepositoryWorkspace(t, db, "Two")
	owner3, workspace3 := domainRepositoryWorkspace(t, db, "Three")
	repo := NewDomainRepository(db)
	access := NewAccessRepository(db)
	_, err := createDomainLocked(t, access, repo, owner1, pendingDomain(workspace1, "race.example", time.Now().UTC().Add(-48*time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	type attempt struct {
		owner, workspace uuid.UUID
		db               *gorm.DB
	}
	attempts := []attempt{{owner2, workspace2, secondCapabilityDB(t, db)}, {owner3, workspace3, secondCapabilityDB(t, db)}}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for _, item := range attempts {
		wait.Add(1)
		go func(item attempt) {
			defer wait.Done()
			<-start
			candidate := pendingDomain(item.workspace, "race.example", time.Now().UTC())
			_, e := createDomainLocked(t, NewAccessRepository(item.db), NewDomainRepository(item.db), item.owner, candidate)
			results <- e
		}(item)
	}
	close(start)
	wait.Wait()
	close(results)
	successes, conflicts := 0, 0
	for result := range results {
		if result == nil {
			successes++
		} else if errors.Is(result, domain.ErrDomainConflict) {
			conflicts++
		} else {
			t.Fatalf("race error=%v", result)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	var count int64
	if err = db.Table("workspace_domains").Where("domain=?", "race.example").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("rows=%d err=%v", count, err)
	}
}

func TestWorkspaceDomainMigrationUpDown(t *testing.T) {
	db := accessTestDB(t, 23)
	up, err := os.ReadFile("../../../migrations/000024_workspace_domains.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(up)).Error; err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable("workspace_domains") || !db.Migrator().HasTable("workspace_domain_requests") {
		t.Fatal("domain tables missing")
	}
	down, err := os.ReadFile("../../../migrations/000024_workspace_domains.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec(string(down)).Error; err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable("workspace_domains") || db.Migrator().HasTable("workspace_domain_requests") {
		t.Fatal("domain tables survived down")
	}
}
