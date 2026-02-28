package engine

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/franksops/gofast/provider"
)

// Walker traverses a directory iteratively to push TransferJobs to a channel.
// It avoids deep recursion to prevent stack overflows on very deep directory structures.
type Walker struct {
	SourceProvider provider.Provider
	JobChan        JobChannel
}

// NewWalker creates a new iterative directory walker.
func NewWalker(src provider.Provider, jobChan JobChannel) *Walker {
	return &Walker{
		SourceProvider: src,
		JobChan:        jobChan,
	}
}

// Walk start an iterative (stack-based) walk of the root directory.
func (w *Walker) Walk(ctx context.Context, sourcePath string, destPath string) error {
	// Let's get information about the source path first.
	stat, err := w.SourceProvider.Stat(ctx, sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source %s: %w", sourcePath, err)
	}

	// If the root itself is just a file, we send one job and return.
	if !stat.IsDir() {
		job := TransferJob{
			ID:              sourcePath, // A UUID generator would be better here in a full app
			SourcePath:      sourcePath,
			DestinationPath: destPath,
			FileInfo:        stat,
			Ctx:             ctx,
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case w.JobChan <- job:
			return nil
		}
	}

	// For a directory, initialize a stack for the iterative walk.
	// We'll store paths relative to the sourcePath to easily compute destination paths.
	type walkItem struct {
		relPath string
	}

	stack := []walkItem{{relPath: ""}}

	for len(stack) > 0 {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Pop item
		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		currentSourcePath := sourcePath
		if curr.relPath != "" {
			currentSourcePath = filepath.Join(sourcePath, curr.relPath)
		}

		entries, err := w.SourceProvider.List(ctx, currentSourcePath)
		if err != nil {
			// In production, might log and continue, or fail fast based on config.
			return fmt.Errorf("failed to list directory %s: %w", currentSourcePath, err)
		}

		for _, entry := range entries {
			entryRelPath := entry.Name()
			if curr.relPath != "" {
				entryRelPath = filepath.Join(curr.relPath, entry.Name())
			}

			if entry.IsDir() {
				// Push subdirectory onto stack to process later
				stack = append(stack, walkItem{relPath: entryRelPath})
			} else {
				// It's a file, generate a job
				job := TransferJob{
					ID:              filepath.Join(sourcePath, entryRelPath), 
					SourcePath:      filepath.Join(sourcePath, entryRelPath),
					DestinationPath: filepath.Join(destPath, entryRelPath),
					FileInfo:        entry,
					Ctx:             ctx,
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case w.JobChan <- job:
					// Enqueued
				}
			}
		}
	}

	return nil
}
