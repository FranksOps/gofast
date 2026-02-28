package engine

import (
	"bytes"
	"testing"
	"time"

	"github.com/franksops/gofast/store"
)

type MockStore struct {
	Jobs map[string]*store.JobRecord
}

func (m *MockStore) SaveJob(job *store.JobRecord) error {
	m.Jobs[job.ID] = job
	return nil
}

func (m *MockStore) GetJob(id string) (*store.JobRecord, error) {
	job, ok := m.Jobs[id]
	if !ok {
		return nil, store.ErrJobNotFound
	}
	return job, nil
}

func (m *MockStore) Close() error { return nil }

func TestJobTracker(t *testing.T) {
	mockStore := &MockStore{Jobs: make(map[string]*store.JobRecord)}
	config := DefaultCheckpointConfig
	tracker := NewJobTracker(mockStore, config)

	job := TransferJob{
		ID:              "test-job",
		SourcePath:      "src",
		DestinationPath: "dst",
	}

	err := tracker.InitJob(job)
	if err != nil {
		t.Fatalf("Failed to init job: %v", err)
	}

	record, err := mockStore.GetJob("test-job")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if record.State != store.StatePending {
		t.Errorf("Expected state %s, got %s", store.StatePending, record.State)
	}

	err = tracker.MarkInProgress("test-job")
	if err != nil {
		t.Fatalf("Failed to mark in progress: %v", err)
	}
	if record.State != store.StateInProgress {
		t.Errorf("Expected state %s, got %s", store.StateInProgress, record.State)
	}

	err = tracker.MarkCompleted("test-job")
	if err != nil {
		t.Fatalf("Failed to mark completed: %v", err)
	}
	if record.State != store.StateCompleted {
		t.Errorf("Expected state %s, got %s", store.StateCompleted, record.State)
	}
}

func TestTrackedWriter_Checkpointing(t *testing.T) {
	mockStore := &MockStore{Jobs: make(map[string]*store.JobRecord)}

	// Fast checkpointing config
	config := CheckpointConfig{
		BytesInterval: 10,
		TimeInterval:  time.Millisecond,
	}

	tracker := NewJobTracker(mockStore, config)

	err := tracker.InitJob(TransferJob{ID: "job2"})
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	_ = tracker.MarkInProgress("job2")

	buf := new(bytes.Buffer)
	tw := tracker.NewTrackedWriter(buf, "job2", 0)

	// Write 5 bytes, shouldn't trigger checkpoint (interval=10)
	n, err := tw.Write([]byte("12345"))
	if err != nil || n != 5 {
		t.Fatalf("Write failed: n=%d err=%v", n, err)
	}

	record, _ := mockStore.GetJob("job2")
	if record.BytesTransferred != 0 {
		t.Errorf("Expected 0 bytes transferred (no checkpoint), got %d", record.BytesTransferred)
	}

	// Write 6 more bytes (total 11) - should trigger checkpoint based on bytes
	n, err = tw.Write([]byte("678901"))
	if err != nil || n != 6 {
		t.Fatalf("Write failed: n=%d err=%v", n, err)
	}

	record, _ = mockStore.GetJob("job2")
	if record.BytesTransferred != 11 {
		t.Errorf("Expected 11 bytes transferred due to checkpoint, got %d", record.BytesTransferred)
	}
}
