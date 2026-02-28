package provider

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalProvider_Stat(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "local-provider-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempBase)

	p := NewLocalProvider(tempBase)
	ctx := context.Background()

	testFile := "test-stat.txt"
	testContent := []byte("hello stat")

	if err := os.WriteFile(filepath.Join(tempBase, testFile), testContent, 0644); err != nil {
		t.Fatal(err)
	}

	info, err := p.Stat(ctx, testFile)
	if err != nil {
		t.Errorf("Stat failed: %v", err)
	}

	if info.Name() != testFile {
		t.Errorf("expected %q, got %q", testFile, info.Name())
	}
	if info.Size() != int64(len(testContent)) {
		t.Errorf("expected size %d, got %d", len(testContent), info.Size())
	}
	if info.IsDir() {
		t.Errorf("expected isDir to be false")
	}
}

func TestLocalProvider_List(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "local-provider-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempBase)

	// Create a subdirectory
	testDir := "subdir"
	if err := os.MkdirAll(filepath.Join(tempBase, testDir), 0755); err != nil {
		t.Fatal(err)
	}

	// Create some files inside the subdirectory
	file1 := "file1.txt"
	file2 := "file2.txt"
	if err := os.WriteFile(filepath.Join(tempBase, testDir, file1), []byte("f1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempBase, testDir, file2), []byte("f2"), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewLocalProvider(tempBase)
	ctx := context.Background()

	infos, err := p.List(ctx, testDir)
	if err != nil {
		t.Errorf("List failed: %v", err)
	}

	if len(infos) != 2 {
		t.Errorf("expected 2 items, got %d", len(infos))
	}

	foundF1, foundF2 := false, false
	for _, info := range infos {
		if info.Name() == file1 {
			foundF1 = true
		}
		if info.Name() == file2 {
			foundF2 = true
		}
	}
	if !foundF1 || !foundF2 {
		t.Errorf("expected to find file1 and file2")
	}
}

func TestLocalProvider_OpenRead(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "local-provider-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempBase)

	testFile := "test-read.txt"
	testContent := []byte("hello read")
	if err := os.WriteFile(filepath.Join(tempBase, testFile), testContent, 0644); err != nil {
		t.Fatal(err)
	}

	p := NewLocalProvider(tempBase)
	ctx := context.Background()

	rc, err := p.OpenRead(ctx, testFile)
	if err != nil {
		t.Errorf("OpenRead failed: %v", err)
	}

	content, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Errorf("ReadAll failed: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("expected content %q, got %q", testContent, content)
	}
}

type dummyFileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

func (d *dummyFileInfo) Name() string       { return d.name }
func (d *dummyFileInfo) Size() int64        { return d.size }
func (d *dummyFileInfo) IsDir() bool        { return d.isDir }
func (d *dummyFileInfo) ModTime() time.Time { return d.modTime }

func TestLocalProvider_OpenWrite(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "local-provider-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempBase)

	p := NewLocalProvider(tempBase)
	ctx := context.Background()

	testFile := "nested/test-write.txt"
	testContent := []byte("hello write")
	testModTime := time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC)

	metadata := &dummyFileInfo{
		name:    "test-write.txt",
		size:    int64(len(testContent)),
		isDir:   false,
		modTime: testModTime,
	}

	wc, err := p.OpenWrite(ctx, testFile, metadata)
	if err != nil {
		t.Fatalf("OpenWrite failed: %v", err)
	}

	n, err := wc.Write(testContent)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(testContent) {
		t.Errorf("expected to write %d bytes, wrote %d", len(testContent), n)
	}

	if err := wc.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify the file exists and is correct
	fullPath := filepath.Join(tempBase, testFile)
	readContent, err := os.ReadFile(fullPath)
	if err != nil {
		t.Errorf("ReadFile failed: %v", err)
	}
	if string(readContent) != string(testContent) {
		t.Errorf("expected content %q, got %q", testContent, readContent)
	}

	// Verify metadata (modTime)
	stat, err := os.Stat(fullPath)
	if err != nil {
		t.Errorf("Stat failed: %v", err)
	}
	if !stat.ModTime().Equal(testModTime) {
		t.Errorf("expected mod time %v, got %v", testModTime, stat.ModTime())
	}
}
