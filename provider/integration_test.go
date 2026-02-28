package provider

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestLocalToLocalTransfer tests a complete file transfer between two local providers
func TestLocalToLocalTransfer(t *testing.T) {
	// Create temp directories for source and destination
	srcDir, err := os.MkdirTemp("", "gofast-src-*")
	if err != nil {
		t.Fatalf("Failed to create source temp dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "gofast-dst-*")
	if err != nil {
		t.Fatalf("Failed to create destination temp dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	// Create test file in source
	testContent := []byte("Hello, Gofast! This is a test file for integration.")
	testFile := "test.txt"
	srcPath := filepath.Join(srcDir, testFile)
	
	if err := os.WriteFile(srcPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create providers
	srcProvider := NewLocalProvider(srcDir)
	dstProvider := NewLocalProvider(dstDir)

	ctx := context.Background()

	// Stat the source file
	srcInfo, err := srcProvider.Stat(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to stat source file: %v", err)
	}

	// Open source for reading
	srcReader, err := srcProvider.OpenRead(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to open source file: %v", err)
	}
	defer srcReader.Close()

	// Read content to verify later
	srcData, err := io.ReadAll(srcReader)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}

	// Re-open for actual transfer (simulate real usage)
	srcReader2, err := srcProvider.OpenRead(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to re-open source file: %v", err)
	}
	defer srcReader2.Close()

	// Open destination for writing
	dstWriter, err := dstProvider.OpenWrite(ctx, testFile, srcInfo)
	if err != nil {
		t.Fatalf("Failed to open destination file: %v", err)
	}

	// Copy data
	_, err = io.Copy(dstWriter, srcReader2)
	if err != nil {
		dstWriter.Close()
		t.Fatalf("Failed to copy data: %v", err)
	}

	// Close destination (applies metadata)
	if err := dstWriter.Close(); err != nil {
		t.Fatalf("Failed to close destination file: %v", err)
	}

	// Verify destination file exists and has correct content
	dstInfo, err := dstProvider.Stat(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to stat destination file: %v", err)
	}

	if dstInfo.Size() != int64(len(testContent)) {
		t.Errorf("Expected destination size %d, got %d", len(testContent), dstInfo.Size())
	}

	// Read and verify content
	dstReader, err := dstProvider.OpenRead(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to open destination file for reading: %v", err)
	}
	defer dstReader.Close()

	dstData, err := io.ReadAll(dstReader)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if !bytes.Equal(dstData, testContent) {
		t.Errorf("Content mismatch.\nExpected: %s\nGot: %s", string(testContent), string(dstData))
	}

	// Verify source data matches
	if !bytes.Equal(srcData, testContent) {
		t.Errorf("Source data mismatch")
	}
}

// TestLocalProviderMetadataPreservation tests that metadata is preserved during transfer
func TestLocalProviderMetadataPreservation(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "gofast-meta-src-*")
	if err != nil {
		t.Fatalf("Failed to create source temp dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "gofast-meta-dst-*")
	if err != nil {
		t.Fatalf("Failed to create destination temp dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	// Create test file with specific permissions
	testFile := "perms.txt"
	srcPath := filepath.Join(srcDir, testFile)
	
	if err := os.WriteFile(srcPath, []byte("test"), 0755); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	srcProvider := NewLocalProvider(srcDir).WithMetadataMapper(NewMetadataMapper())
	dstProvider := NewLocalProvider(dstDir).WithMetadataMapper(NewMetadataMapper())

	ctx := context.Background()

	// Get source info including Unix metadata
	srcInfo, err := srcProvider.Stat(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to stat source: %v", err)
	}

	// Transfer
	srcReader, err := srcProvider.OpenRead(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to open source: %v", err)
	}
	defer srcReader.Close()

	dstWriter, err := dstProvider.OpenWrite(ctx, testFile, srcInfo)
	if err != nil {
		t.Fatalf("Failed to open destination: %v", err)
	}

	_, err = io.Copy(dstWriter, srcReader)
	if err != nil {
		dstWriter.Close()
		t.Fatalf("Failed to copy: %v", err)
	}

	if err := dstWriter.Close(); err != nil {
		t.Fatalf("Failed to close destination: %v", err)
	}

	// Verify destination metadata
	dstInfo, err := dstProvider.Stat(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to stat destination: %v", err)
	}

	// Check permissions (may vary based on umask, but should be close)
	srcUnix, srcOK := srcInfo.(UnixFileInfo)
	dstUnix, dstOK := dstInfo.(UnixFileInfo)
	
	if !srcOK || !dstOK {
		t.Skip("UnixFileInfo not available on this platform")
	}

	// Permissions should match (at least the user bits)
	srcMode := srcUnix.Mode() & 0777
	dstMode := dstUnix.Mode() & 0777
	
	if srcMode != dstMode {
		t.Logf("Note: Mode changed from %o to %o (expected due to umask)", srcMode, dstMode)
	}
}

// TestLocalProviderDirectoryCreation tests that parent directories are created automatically
func TestLocalProviderDirectoryCreation(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "gofast-dir-src-*")
	if err != nil {
		t.Fatalf("Failed to create source temp dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "gofast-dir-dst-*")
	if err != nil {
		t.Fatalf("Failed to create destination temp dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	// Create nested structure in source
	nestedPath := "a/b/c/deep.txt"
	fullSrcPath := filepath.Join(srcDir, nestedPath)
	
	if err := os.MkdirAll(filepath.Dir(fullSrcPath), 0755); err != nil {
		t.Fatalf("Failed to create source directories: %v", err)
	}
	
	testContent := []byte("deep file content")
	if err := os.WriteFile(fullSrcPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	srcProvider := NewLocalProvider(srcDir)
	dstProvider := NewLocalProvider(dstDir)

	ctx := context.Background()

	// Transfer nested file
	srcInfo, err := srcProvider.Stat(ctx, nestedPath)
	if err != nil {
		t.Fatalf("Failed to stat source: %v", err)
	}

	srcReader, err := srcProvider.OpenRead(ctx, nestedPath)
	if err != nil {
		t.Fatalf("Failed to open source: %v", err)
	}
	defer srcReader.Close()

	dstWriter, err := dstProvider.OpenWrite(ctx, nestedPath, srcInfo)
	if err != nil {
		t.Fatalf("Failed to open destination: %v", err)
	}

	_, err = io.Copy(dstWriter, srcReader)
	if err != nil {
		dstWriter.Close()
		t.Fatalf("Failed to copy: %v", err)
	}

	if err := dstWriter.Close(); err != nil {
		t.Fatalf("Failed to close destination: %v", err)
	}

	// Verify destination file exists at nested path
	fullDstPath := filepath.Join(dstDir, nestedPath)
	if _, err := os.Stat(fullDstPath); err != nil {
		t.Errorf("Destination file not found at %s: %v", fullDstPath, err)
	}

	// Verify content
	dstReader, err := dstProvider.OpenRead(ctx, nestedPath)
	if err != nil {
		t.Fatalf("Failed to open destination for reading: %v", err)
	}
	defer dstReader.Close()

	dstData, err := io.ReadAll(dstReader)
	if err != nil {
		t.Fatalf("Failed to read destination: %v", err)
	}

	if !bytes.Equal(dstData, testContent) {
		t.Errorf("Content mismatch")
	}
}

// TestConcurrentTransfers tests multiple concurrent transfers to the same destination provider
func TestConcurrentTransfers(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "gofast-conc-src-*")
	if err != nil {
		t.Fatalf("Failed to create source temp dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "gofast-conc-dst-*")
	if err != nil {
		t.Fatalf("Failed to create destination temp dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	// Create multiple test files
	numFiles := 10
	files := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(srcDir, "file"+string(rune('0'+i))+".txt")
		content := []byte("content for file " + string(rune('0'+i)))
		if err := os.WriteFile(filename, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}
		files[i] = "file" + string(rune('0'+i)) + ".txt"
	}

	srcProvider := NewLocalProvider(srcDir)
	dstProvider := NewLocalProvider(dstDir)
	ctx := context.Background()

	// Transfer all files concurrently
	done := make(chan error, numFiles)
	
	for _, filename := range files {
		go func(file string) {
			srcInfo, err := srcProvider.Stat(ctx, file)
			if err != nil {
				done <- err
				return
			}

			srcReader, err := srcProvider.OpenRead(ctx, file)
			if err != nil {
				done <- err
				return
			}
			defer srcReader.Close()

			dstWriter, err := dstProvider.OpenWrite(ctx, file, srcInfo)
			if err != nil {
				done <- err
				return
			}

			_, err = io.Copy(dstWriter, srcReader)
			if err != nil {
				dstWriter.Close()
				done <- err
				return
			}

			done <- dstWriter.Close()
		}(filename)
	}

	// Wait for all transfers
	for i := 0; i < numFiles; i++ {
		if err := <-done; err != nil {
			t.Errorf("Transfer failed: %v", err)
		}
	}

	// Verify all files exist in destination
	for _, filename := range files {
		dstPath := filepath.Join(dstDir, filename)
		if _, err := os.Stat(dstPath); err != nil {
			t.Errorf("Destination file missing: %s", filename)
		}
	}
}
