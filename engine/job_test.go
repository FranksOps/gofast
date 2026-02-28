package engine_test

import (
	"context"
	"testing"

	"github.com/franksops/gofast/engine"
)

func TestTransferJob(t *testing.T) {
	job := engine.TransferJob{
		SourcePath:      "/tmp/source.txt",
		DestinationPath: "/tmp/dest.txt",
		FileInfo:        nil,
		Ctx:             context.Background(),
	}

	if job.SourcePath != "/tmp/source.txt" {
		t.Errorf("Expected /tmp/source.txt, got %s", job.SourcePath)
	}
}

func TestJobChannel(t *testing.T) {
	ch := make(engine.JobChannel, 1)

	job := engine.TransferJob{
		SourcePath: "/tmp/foo.txt",
	}

	ch <- job
	received := <-ch

	if received.SourcePath != "/tmp/foo.txt" {
		t.Errorf("Expected /tmp/foo.txt, got %s", received.SourcePath)
	}
}
