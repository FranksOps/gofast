package engine_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/franksops/gofast/engine"
)

func TestWorkerPool_SetWorkerCount(t *testing.T) {
	ch := make(engine.JobChannel, 100)
	handler := func(ctx context.Context, job engine.TransferJob) error {
		return nil
	}

	pool := engine.NewWorkerPool(context.Background(), ch, handler)

	pool.SetWorkerCount(5)
	if count := pool.WorkerCount(); count != 5 {
		t.Errorf("Expected 5 workers, got %d", count)
	}

	pool.SetWorkerCount(2)
	if count := pool.WorkerCount(); count != 2 {
		t.Errorf("Expected 2 workers, got %d", count)
	}

	pool.SetWorkerCount(10)
	if count := pool.WorkerCount(); count != 10 {
		t.Errorf("Expected 10 workers, got %d", count)
	}

	pool.Stop()
}

func TestWorkerPool_Execution(t *testing.T) {
	ch := make(engine.JobChannel, 100)

	var mu sync.Mutex
	var processed int

	handler := func(ctx context.Context, job engine.TransferJob) error {
		mu.Lock()
		processed++
		mu.Unlock()
		time.Sleep(10 * time.Millisecond) // simulate work
		return nil
	}

	pool := engine.NewWorkerPool(context.Background(), ch, handler)
	pool.SetWorkerCount(3)

	for i := 0; i < 10; i++ {
		ch <- engine.TransferJob{SourcePath: "file.txt"}
	}

	// wait for jobs to complete (roughly)
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	if processed != 10 {
		t.Errorf("Expected 10 processed jobs, got %d", processed)
	}
	mu.Unlock()

	pool.Stop()
}
