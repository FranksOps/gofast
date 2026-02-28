package engine

import (
	"context"

	"github.com/franksops/gofast/provider"
)

// TransferJob represents a single file transfer operation from a source
// provider to a destination provider.
type TransferJob struct {
	// SourcePath is the file path to read from the source provider.
	SourcePath string

	// DestinationPath is the file path to write to the destination provider.
	DestinationPath string

	// FileInfo holds the metadata of the source file to be preserved or
	// checked at the destination.
	FileInfo provider.FileInfo

	// Ctx allows cancellation or timeout settings for this specific job.
	Ctx context.Context
}

// JobChannel is a channel used to queue and dispatch TransferJobs to workers
// in the worker pool.
type JobChannel chan TransferJob
