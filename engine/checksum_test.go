package engine

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestChecksumWriter(t *testing.T) {
	data := []byte("hello world")
	
	var buf bytes.Buffer
	cw := NewChecksumWriter(&buf)
	
	n, err := cw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, got %d", len(data), n)
	}
	
	if buf.String() != string(data) {
		t.Errorf("Expected buffer to contain %q, got %q", data, buf.String())
	}
	
	// Verify checksum is non-zero
	checksum := cw.Checksum()
	if checksum == 0 {
		t.Error("Expected non-zero checksum")
	}
	
	if cw.BytesWritten() != int64(len(data)) {
		t.Errorf("Expected %d bytes written, got %d", len(data), cw.BytesWritten())
	}
}

func TestChecksumReader(t *testing.T) {
	data := []byte("hello world")
	
	cr := NewChecksumReader(bytes.NewReader(data))
	
	readData := make([]byte, len(data))
	n, err := cr.Read(readData)
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}
	
	if n != len(data) {
		t.Errorf("Expected to read %d bytes, got %d", len(data), n)
	}
	
	if !bytes.Equal(readData, data) {
		t.Errorf("Expected read data to match %q, got %q", data, readData)
	}
	
	// Verify checksum is non-zero
	checksum := cr.Checksum()
	if checksum == 0 {
		t.Error("Expected non-zero checksum")
	}
	
	if cr.BytesRead() != int64(len(data)) {
		t.Errorf("Expected %d bytes read, got %d", len(data), cr.BytesRead())
	}
}

func TestChecksumConsistency(t *testing.T) {
	data := []byte("test data for checksum consistency")
	
	// Write with ChecksumWriter
	var buf bytes.Buffer
	cw := NewChecksumWriter(&buf)
	_, err := cw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	writeChecksum := cw.Checksum()
	
	// Read with ChecksumReader
	cr := NewChecksumReader(bytes.NewReader(data))
	_, err = io.ReadAll(cr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	readChecksum := cr.Checksum()
	
	// Checksums should match
	if writeChecksum != readChecksum {
		t.Errorf("Checksum mismatch: write=%d, read=%d", writeChecksum, readChecksum)
	}
}

func TestChecksumPool(t *testing.T) {
	pool := NewChecksumPool()
	
	// Get hasher from pool
	h1 := pool.Get()
	h1.Write([]byte("test"))
	checksum1 := h1.Sum64()
	
	// Put back
	pool.Put(h1)
	
	// Get again (might be the same instance)
	h2 := pool.Get()
	
	// After reset, should produce same checksum for same data
	h2.Write([]byte("test"))
	checksum2 := h2.Sum64()
	
	if checksum1 != checksum2 {
		t.Errorf("Expected same checksum after pool reuse: %d vs %d", checksum1, checksum2)
	}
	
	pool.Put(h2)
}

func TestVerifyChecksum(t *testing.T) {
	tests := []struct {
		name     string
		actual   uint64
		expected uint64
		want     bool
	}{
		{"matching", 12345, 12345, true},
		{"mismatch", 12345, 54321, false},
		{"zero", 0, 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyChecksum(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("VerifyChecksum(%d, %d) = %v, want %v", tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}

func TestChecksumWriterMultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	cw := NewChecksumWriter(&buf)
	
	parts := []string{"hello", " ", "world", "!"}
	expected := strings.Join(parts, "")
	
	for _, part := range parts {
		_, err := cw.Write([]byte(part))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}
	
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
	
	// Checksum should be non-zero
	if cw.Checksum() == 0 {
		t.Error("Expected non-zero checksum")
	}
}
