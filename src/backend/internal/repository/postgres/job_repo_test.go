package postgres

import (
	"context"
	"sync"
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
	start := make(chan struct{})
	results := make(chan []uuid.UUID, 2)
	errors := make(chan error, 2)
	var workers sync.WaitGroup
	for _, workerID := range []string{"worker-a", "worker-b"} {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			claimed, err := repo.Claim(context.Background(), now, 100, workerID, time.Minute)
			if err != nil {
				errors <- err
				return
			}
			ids := make([]uuid.UUID, len(claimed))
			for index := range claimed {
				ids[index] = claimed[index].ID
			}
			results <- ids
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
	var claimed []uuid.UUID
	for ids := range results {
		claimed = append(claimed, ids...)
	}
	if len(claimed) != 1 || claimed[0] != jobID {
		t.Fatalf("concurrent workers claimed job more than once: %v", claimed)
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
