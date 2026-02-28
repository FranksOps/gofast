package provider

import (
	"context"
	"io"
	"time"
)

// FileInfo represents the standard metadata for a file or a directory
// across different storage abstractions.
type FileInfo interface {
	Name() string
	Size() int64
	IsDir() bool
	ModTime() time.Time
}

// Provider represents a storage backend abstraction.
// A typical Provider might be local storage, S3, FTP, etc.
type Provider interface {
	// Stat returns the FileInfo for the given path.
	Stat(ctx context.Context, path string) (FileInfo, error)

	// List returns the contents of the given directory.
	List(ctx context.Context, path string) ([]FileInfo, error)

	// OpenRead opens a file for streaming reads.
	OpenRead(ctx context.Context, path string) (io.ReadCloser, error)

	// OpenWrite opens a file for streaming writes, applying metadata if supported.
	OpenWrite(ctx context.Context, path string, metadata FileInfo) (io.WriteCloser, error)
}
