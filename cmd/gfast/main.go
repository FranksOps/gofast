package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/franksops/gofast/engine"
	"github.com/franksops/gofast/provider"
	"github.com/franksops/gofast/store"
	"github.com/franksops/gofast/ui"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	defaultStreams    = 32
	defaultBufferSize = 1 * 1024 * 1024 // 1MB
)

func main() {
	// CLI flags
	var (
		source      string
		dest        string
		streams     int
		bufferSize  int
		stateDir    string
		noMetadata  bool
		checksum    bool
		tuiEnabled  bool
	)

	flag.StringVar(&source, "source", "", "Source path (local or s3://bucket/prefix)")
	flag.StringVar(&dest, "dest", "", "Destination path (local or s3://bucket/prefix)")
	flag.IntVar(&streams, "streams", defaultStreams, "Number of concurrent transfer streams")
	flag.IntVar(&bufferSize, "buffer-size", defaultBufferSize, "Buffer size in bytes for each stream")
	flag.StringVar(&stateDir, "state-dir", "./.gofast-state", "Directory to store state/checkpoint files")
	flag.BoolVar(&noMetadata, "no-metadata", false, "Disable metadata preservation (UID/GID/mode)")
	flag.BoolVar(&checksum, "checksum", false, "Enable streaming checksum verification (CRC64)")
	flag.BoolVar(&tuiEnabled, "tui", true, "Enable TUI (disable for headless operation)")
	flag.Parse()

	if source == "" || dest == "" {
		fmt.Println("Usage: gfast -source <src> -dest <dst> [options]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  gfast -source /data/old -dest /data/new -streams 64")
		fmt.Println("  gfast -source /data/local -dest s3://bucket/prefix -streams 32")
		os.Exit(1)
	}

	// Create state directory
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		log.Fatalf("Failed to create state directory: %v", err)
	}

	// Initialize state store
	stateStorePath := filepath.Join(stateDir, "state.db")
	stateStore, err := store.NewBoltStore(stateStorePath)
	if err != nil {
		log.Fatalf("Failed to initialize state store: %v", err)
	}
	defer stateStore.Close()

	// Initialize job tracker
	jobTracker := engine.NewJobTracker(stateStore, engine.DefaultCheckpointConfig)

	// Create source provider
	srcProvider, err := createProvider(source, !noMetadata)
	if err != nil {
		log.Fatalf("Failed to create source provider: %v", err)
	}

	// Create destination provider
	dstProvider, err := createProvider(dest, !noMetadata)
	if err != nil {
		log.Fatalf("Failed to create destination provider: %v", err)
	}

	// Create buffer pool
	bufferPool := engine.NewBufferPool(bufferSize)

	// Job channel for work distribution
	jobChan := make(engine.JobChannel, 1000)

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TUI state
	tuiState := &ui.UIState{
		ActiveStreams: make([]*ui.ActiveStream, 0),
		MaxWorkers:    streams,
		ActiveWorkers: streams,
		IsRunning:     true,
	}

	// Create TUI model
	var tuiModel ui.TUIModel
	var teaProgram *tea.Program

	if tuiEnabled {
		tuiModel = ui.NewTUIModel(tuiState)
		teaProgram = tea.NewProgram(tuiModel, tea.WithAltScreen())

		// Start TUI update loop
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Send update to TUI
					teaProgram.Send(ui.TUIUpdateMsg{State: tuiState})
				}
			}
		}()
	}

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

	// Worker pool
	workerPool := engine.NewWorkerPool(ctx, jobChan, func(ctx context.Context, job engine.TransferJob) error {
		return transferFile(ctx, job, srcProvider, dstProvider, jobTracker, bufferPool, checksum, tuiState)
	})
	workerPool.SetWorkerCount(streams)

	// Handle worker count changes from TUI
	if tuiEnabled {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Check for TUI messages (handled in update loop)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	}

	// Start walker
	walker := engine.NewWalker(srcProvider, jobChan)
	walkCtx, walkCancel := context.WithCancel(ctx)

	// Start walking in background
	go func() {
		defer walkCancel()
		defer close(jobChan)

		// Determine destination root
		destRoot := dest
		if filepath.Separator == '/' && len(dest) > 0 && dest[0] != '/' {
			// Relative path, keep as is
		}

		if err := walker.Walk(walkCtx, source, destRoot); err != nil {
			log.Printf("Walker error: %v", err)
		}
	}()

	// Wait for completion or signal
	done := make(chan struct{})
	go func() {
		<-sigChan
		cancel()
		close(done)
	}()

	// Wait for jobs to complete
	<-walkCtx.Done()
	workerPool.Stop()

	if tuiEnabled {
		tuiState.Done = true
		tuiState.IsRunning = false
		teaProgram.Send(ui.TUIUpdateMsg{State: tuiState})
		time.Sleep(200 * time.Millisecond)
		teaProgram.Quit()
	}

	fmt.Println("\nMigration complete.")
}

func createProvider(path string, withMetadata bool) (provider.Provider, error) {
	// Check if S3 path
	if len(path) >= 5 && path[:5] == "s3://" {
		ctx := context.Background()
		// Parse s3://bucket/prefix
		s3Path := path[5:] // Remove "s3://"
		bucket, prefix, _ := strings.Cut(s3Path, "/")
		return provider.NewS3Provider(ctx, bucket, prefix)
	}

	// Local provider
	localProvider := provider.NewLocalProvider("")
	if withMetadata {
		localProvider.WithMetadataMapper(provider.NewMetadataMapper())
	}
	return localProvider, nil
}

func transferFile(
	ctx context.Context,
	job engine.TransferJob,
	srcProvider provider.Provider,
	dstProvider provider.Provider,
	tracker *engine.JobTracker,
	bufferPool *engine.BufferPool,
	checksum bool,
	tuiState *ui.UIState,
) error {
	// Initialize job in store
	if err := tracker.InitJob(job); err != nil {
		return fmt.Errorf("failed to init job: %w", err)
	}

	// Mark as in progress
	if err := tracker.MarkInProgress(job.ID); err != nil {
		return fmt.Errorf("failed to mark job in progress: %w", err)
	}

	// Open source
	srcReader, err := srcProvider.OpenRead(ctx, job.SourcePath)
	if err != nil {
		tracker.MarkFailed(job.ID, err)
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcReader.Close()

	// Wrap with checksum if enabled
	var reader io.Reader = srcReader
	// TODO: Add CRC64/XXHash wrapper here

	// Open destination
	dstWriter, err := dstProvider.OpenWrite(ctx, job.DestinationPath, job.FileInfo)
	if err != nil {
		tracker.MarkFailed(job.ID, err)
		return fmt.Errorf("failed to open destination: %w", err)
	}

	// Wrap writer with tracking
	trackedWriter := tracker.NewTrackedWriter(dstWriter, job.ID, 0)

	// Perform transfer
	buf := bufferPool.Get()
	defer bufferPool.Put(buf)

	_, err = io.CopyBuffer(trackedWriter, reader, *buf)
	if err != nil {
		dstWriter.Close()
		tracker.MarkFailed(job.ID, err)
		return fmt.Errorf("transfer failed: %w", err)
	}

	// Close destination (applies metadata)
	if err := dstWriter.Close(); err != nil {
		tracker.MarkFailed(job.ID, err)
		return fmt.Errorf("failed to close destination: %w", err)
	}

	// Mark as completed
	if err := tracker.MarkCompleted(job.ID); err != nil {
		return fmt.Errorf("failed to mark job completed: %w", err)
	}

	// Update TUI state
	if tuiState != nil {
		tuiState.CompletedFiles++
		tuiState.CompletedBytes += job.FileInfo.Size()
	}

	return nil
}
