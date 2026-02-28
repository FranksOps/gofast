package store

import (
	"path/filepath"
	"testing"
)

func TestBoltStore_SaveAndGetJob(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewBoltStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	// Initial job
	job := &JobRecord{
		ID:               "job-123",
		SourcePath:       "/tmp/src.txt",
		DestinationPath:  "/tmp/dst.txt",
		State:            StatePending,
		BytesTransferred: 0,
		TotalBytes:       1024,
	}

	err = store.SaveJob(job)
	if err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	// Retrieve job
	retrievedJob, err := store.GetJob("job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrievedJob.ID != job.ID {
		t.Errorf("Expected job ID %s, got %s", job.ID, retrievedJob.ID)
	}
	if retrievedJob.State != job.State {
		t.Errorf("Expected job State %s, got %s", job.State, retrievedJob.State)
	}

	// Update job state
	job.State = StateInProgress
	job.BytesTransferred = 512
	err = store.SaveJob(job)
	if err != nil {
		t.Fatalf("Failed to update job: %v", err)
	}

	// Retrieve updated job
	retrievedJob, err = store.GetJob("job-123")
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	if retrievedJob.State != StateInProgress {
		t.Errorf("Expected updated job State %s, got %s", StateInProgress, retrievedJob.State)
	}
	if retrievedJob.BytesTransferred != 512 {
		t.Errorf("Expected updated bytes %d, got %d", 512, retrievedJob.BytesTransferred)
	}

	// Non-existent job
	_, err = store.GetJob("non-existent")
	if err != ErrJobNotFound {
		t.Errorf("Expected ErrJobNotFound, got %v", err)
	}
}

func TestBoltStore_Close(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_close.db")

	store, err := NewBoltStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Errorf("Failed to close BoltStore: %v", err)
	}

	// Try to get a job on closed store
	_, err = store.GetJob("job-123")
	if err == nil {
		t.Error("Expected error when accessing closed store, got nil")
	}
}
