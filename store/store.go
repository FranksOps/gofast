package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.etcd.io/bbolt"
)

var (
	// ErrJobNotFound is returned when a job is not found in the state store.
	ErrJobNotFound = errors.New("job not found")
)

var (
	jobsBucket = []byte("jobs")
)

// JobState represents the current state of a file transfer.
type JobState string

const (
	StatePending    JobState = "Pending"
	StateInProgress JobState = "InProgress"
	StateCompleted  JobState = "Completed"
	StateFailed     JobState = "Failed"
)

// JobRecord represents the state of a job in the store.
type JobRecord struct {
	ID               string   `json:"id"`
	SourcePath       string   `json:"source_path"`
	DestinationPath  string   `json:"destination_path"`
	State            JobState `json:"state"`
	BytesTransferred int64    `json:"bytes_transferred"`
	TotalBytes       int64    `json:"total_bytes"`
	Error            string   `json:"error,omitempty"`
}

// Store define the interface for tracking file status.
type Store interface {
	SaveJob(job *JobRecord) error
	GetJob(id string) (*JobRecord, error)
	Close() error
}

// BoltStore is a Store implementation backed by bbolt.
type BoltStore struct {
	db *bbolt.DB
}

// NewBoltStore creates a new BoltStore at the given path.
func NewBoltStore(path string) (*BoltStore, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt database: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(jobsBucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create jobs bucket: %w", err)
	}

	return &BoltStore{db: db}, nil
}

// SaveJob saves a job to the state store.
func (s *BoltStore) SaveJob(job *JobRecord) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(jobsBucket)

		data, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("failed to marshal job: %w", err)
		}

		err = b.Put([]byte(job.ID), data)
		if err != nil {
			return fmt.Errorf("failed to put job: %w", err)
		}

		return nil
	})
}

// GetJob retrieves a job from the state store.
func (s *BoltStore) GetJob(id string) (*JobRecord, error) {
	var job JobRecord
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(jobsBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return ErrJobNotFound
		}

		if err := json.Unmarshal(data, &job); err != nil {
			return fmt.Errorf("failed to unmarshal job: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &job, nil
}

// Close closes the underlying store.
func (s *BoltStore) Close() error {
	return s.db.Close()
}
