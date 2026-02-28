package engine

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/franksops/gofast/provider"
)

type mockFileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) ModTime() time.Time { return m.modTime }

type mockProvider struct {
	files map[string]mockFileInfo
	dirs  map[string][]mockFileInfo
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		files: make(map[string]mockFileInfo),
		dirs:  make(map[string][]mockFileInfo),
	}
}

func (m *mockProvider) Stat(ctx context.Context, path string) (provider.FileInfo, error) {
	if info, ok := m.files[path]; ok {
		return info, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

func (m *mockProvider) List(ctx context.Context, path string) ([]provider.FileInfo, error) {
	if files, ok := m.dirs[path]; ok {
		// Convert to slice of interface
		res := make([]provider.FileInfo, len(files))
		for i, f := range files {
			res[i] = f
		}
		return res, nil
	}
	return nil, fmt.Errorf("directory not found: %s", path)
}

func (m *mockProvider) OpenRead(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockProvider) OpenWrite(ctx context.Context, path string, metadata provider.FileInfo) (io.WriteCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestWalker_Walk(t *testing.T) {
	mp := newMockProvider()

	// Setup mock filesystem structure:
	// /root
	// /root/file1.txt
	// /root/dir1
	// /root/dir1/file2.txt
	// /root/dir1/dir2
	// /root/dir1/dir2/file3.txt

	mp.files["/root"] = mockFileInfo{name: "root", isDir: true}
	
	mp.dirs["/root"] = []mockFileInfo{
		{name: "file1.txt", isDir: false},
		{name: "dir1", isDir: true},
	}
	mp.dirs["/root/dir1"] = []mockFileInfo{
		{name: "file2.txt", isDir: false},
		{name: "dir2", isDir: true},
	}
	mp.dirs["/root/dir1/dir2"] = []mockFileInfo{
		{name: "file3.txt", isDir: false},
	}

	jobChan := make(JobChannel, 10)
	walker := NewWalker(mp, jobChan)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Run walk in a goroutine so it can push to the channel without blocking forever if logic is wrong
	errCh := make(chan error, 1)
	go func() {
		errCh <- walker.Walk(ctx, "/root", "/dest")
		close(jobChan)
	}()

	var receivedFiles []string
	for job := range jobChan {
		receivedFiles = append(receivedFiles, job.SourcePath)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	expectedFiles := []string{
		"/root/file1.txt",
		"/root/dir1/file2.txt",
		"/root/dir1/dir2/file3.txt",
	}

	if len(receivedFiles) != len(expectedFiles) {
		t.Fatalf("Expected %d files, got %d", len(expectedFiles), len(receivedFiles))
	}

	// We can't guarantee order with the stack, so check membership
	for _, expected := range expectedFiles {
		found := false
		for _, received := range receivedFiles {
			if expected == received {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %s not found in jobs", expected)
		}
	}
}

func TestWalker_Walk_SingleFile(t *testing.T) {
	mp := newMockProvider()
	mp.files["/root/file1.txt"] = mockFileInfo{name: "file1.txt", isDir: false}

	jobChan := make(JobChannel, 1)
	walker := NewWalker(mp, jobChan)

	ctx := context.Background()
	err := walker.Walk(ctx, "/root/file1.txt", "/dest/file1.txt")
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	select {
	case job := <-jobChan:
		if job.SourcePath != "/root/file1.txt" {
			t.Errorf("Expected /root/file1.txt, got %s", job.SourcePath)
		}
	default:
		t.Fatal("Expected a job on the channel")
	}
}
