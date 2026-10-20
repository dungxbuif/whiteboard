package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTwoWorkersClaimOnce(t *testing.T) {
	db := accessTestDB(t, 16)
	jobID := uuid.New()
	mustExec(t, db, "INSERT INTO feature_jobs(id,kind,payload,available_at) VALUES (?,?, '{}'::jsonb,now())", jobID, "test")
	repo := NewJobRepository(db)
	now := time.Now().UTC()
	first, err := repo.Claim(context.Background(), now, 100, "worker-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.Claim(context.Background(), now, 100, "worker-b", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].ID != jobID || len(second) != 0 {
		t.Fatalf("claim mismatch first=%#v second=%#v", first, second)
	}
}

func TestExpiredLeaseResumes(t *testing.T) {
	db := accessTestDB(t, 16)
	jobID := uuid.New()
	now := time.Now().UTC()
	mustExec(t, db, "INSERT INTO feature_jobs(id,kind,payload,available_at,lease_owner,lease_expires_at) VALUES (?,?, '{}'::jsonb,?,?,?)", jobID, "test", now.Add(-time.Hour), "dead-worker", now.Add(-time.Minute))
	repo := NewJobRepository(db)
	claimed, err := repo.Claim(context.Background(), now, 1, "replacement", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].ID != jobID || claimed[0].LeaseOwner != "replacement" {
		t.Fatalf("expired lease not resumed: %#v", claimed)
	}
}
