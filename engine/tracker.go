package engine

import (
	"io"
	"sync"
	"time"

	"github.com/franksops/gofast/store"
)

// CheckpointConfig defines the criteria for when to save a job's state
type CheckpointConfig struct {
	// BytesInterval triggers a save after this many bytes have been transferred
	BytesInterval int64
	// TimeInterval triggers a save after this much time has passed
	TimeInterval time.Duration
}

// DefaultCheckpointConfig provides reasonable defaults for checkpointing
var DefaultCheckpointConfig = CheckpointConfig{
	BytesInterval: 10 * 1024 * 1024, // 10 MB
	TimeInterval:  5 * time.Second,
}

// JobTracker wraps a store to provide job tracking and checkpointing capabilities
type JobTracker struct {
	store  store.Store
	config CheckpointConfig
}

// NewJobTracker creates a new JobTracker
func NewJobTracker(store store.Store, config CheckpointConfig) *JobTracker {
	return &JobTracker{
		store:  store,
		config: config,
	}
}

// InitJob initializes a job in the store and returns a tracker for that job
func (jt *JobTracker) InitJob(job TransferJob) error {
	totalBytes := int64(0)
	if job.FileInfo != nil {
		totalBytes = job.FileInfo.Size()
	}

	record := &store.JobRecord{
		ID:               job.ID,
		SourcePath:       job.SourcePath,
		DestinationPath:  job.DestinationPath,
		State:            store.StatePending,
		BytesTransferred: 0,
		TotalBytes:       totalBytes,
	}

	return jt.store.SaveJob(record)
}

// MarkInProgress updates a job's state to InProgress
func (jt *JobTracker) MarkInProgress(jobID string) error {
	record, err := jt.store.GetJob(jobID)
	if err != nil {
		return err
	}
	record.State = store.StateInProgress
	return jt.store.SaveJob(record)
}

// MarkCompleted updates a job's state to Completed
func (jt *JobTracker) MarkCompleted(jobID string) error {
	record, err := jt.store.GetJob(jobID)
	if err != nil {
		return err
	}
	record.State = store.StateCompleted
	record.BytesTransferred = record.TotalBytes // Ensure it matches
	return jt.store.SaveJob(record)
}

// MarkFailed updates a job's state to Failed with an error message
func (jt *JobTracker) MarkFailed(jobID string, err error) error {
	record, getErr := jt.store.GetJob(jobID)
	if getErr != nil {
		return getErr
	}
	record.State = store.StateFailed
	if err != nil {
		record.Error = err.Error()
	}
	return jt.store.SaveJob(record)
}

// TrackedWriter wraps an io.Writer to track bytes written and checkpoint progress
type TrackedWriter struct {
	io.Writer
	tracker *JobTracker
	jobID   string

	mu              sync.Mutex
	bytesWritten    int64
	lastCheckpoint  int64
	lastCheckpointT time.Time
}

// NewTrackedWriter creates a new TrackedWriter
func (jt *JobTracker) NewTrackedWriter(w io.Writer, jobID string, startBytes int64) *TrackedWriter {
	return &TrackedWriter{
		Writer:          w,
		tracker:         jt,
		jobID:           jobID,
		bytesWritten:    startBytes,
		lastCheckpoint:  startBytes,
		lastCheckpointT: time.Now(),
	}
}

// Write implements io.Writer and checkpoints progress
func (tw *TrackedWriter) Write(p []byte) (int, error) {
	n, err := tw.Writer.Write(p)
	if n > 0 {
		tw.mu.Lock()
		tw.bytesWritten += int64(n)

		needsCheckpoint := false
		if tw.bytesWritten-tw.lastCheckpoint >= tw.tracker.config.BytesInterval {
			needsCheckpoint = true
		} else if time.Since(tw.lastCheckpointT) >= tw.tracker.config.TimeInterval {
			needsCheckpoint = true
		}

		currentBytes := tw.bytesWritten
		tw.mu.Unlock()

		if needsCheckpoint {
			tw.checkpoint(currentBytes)
		}
	}
	return n, err
}

func (tw *TrackedWriter) checkpoint(bytes int64) {
	// We don't want a write failure to block everything, but we should try to save
	record, err := tw.tracker.store.GetJob(tw.jobID)
	if err == nil {
		record.BytesTransferred = bytes
		// Ignore save error as it's just a checkpoint
		_ = tw.tracker.store.SaveJob(record)

		tw.mu.Lock()
		tw.lastCheckpoint = bytes
		tw.lastCheckpointT = time.Now()
		tw.mu.Unlock()
	}
}

// BytesWritten returns the total number of bytes written
func (tw *TrackedWriter) BytesWritten() int64 {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return tw.bytesWritten
}
