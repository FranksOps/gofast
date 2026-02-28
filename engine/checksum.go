package engine

import (
	"hash"
	"hash/crc64"
	"io"
	"sync"
)

// ChecksumWriter wraps an io.Writer to compute a checksum while writing.
type ChecksumWriter struct {
	w    io.Writer
	hash hash.Hash64
	n    int64
}

// Hash64 is the hash.Hash64 interface for 64-bit hashes
type Hash64 interface {
	hash.Hash64
}

// NewChecksumWriter creates a new ChecksumWriter that wraps the given writer
// and computes a CRC64 checksum of the data written.
func NewChecksumWriter(w io.Writer) *ChecksumWriter {
	return &ChecksumWriter{
		w:    w,
		hash: crc64.New(crc64.MakeTable(crc64.ISO)),
	}
}

// Write writes data to the underlying writer and updates the checksum.
func (cw *ChecksumWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	if n > 0 {
		cw.n += int64(n)
		cw.hash.Write(p[:n])
	}
	return n, err
}

// Checksum returns the current checksum value.
func (cw *ChecksumWriter) Checksum() uint64 {
	return cw.hash.Sum64()
}

// BytesWritten returns the total number of bytes written.
func (cw *ChecksumWriter) BytesWritten() int64 {
	return cw.n
}

// ChecksumReader wraps an io.Reader to compute a checksum while reading.
type ChecksumReader struct {
	r    io.Reader
	hash hash.Hash64
	n    int64
}

// NewChecksumReader creates a new ChecksumReader that wraps the given reader
// and computes a CRC64 checksum of the data read.
func NewChecksumReader(r io.Reader) *ChecksumReader {
	return &ChecksumReader{
		r:    r,
		hash: crc64.New(crc64.MakeTable(crc64.ISO)),
	}
}

// Read reads data from the underlying reader and updates the checksum.
func (cr *ChecksumReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	if n > 0 {
		cr.n += int64(n)
		cr.hash.Write(p[:n])
	}
	return n, err
}

// Checksum returns the current checksum value.
func (cr *ChecksumReader) Checksum() uint64 {
	return cr.hash.Sum64()
}

// BytesRead returns the total number of bytes read.
func (cr *ChecksumReader) BytesRead() int64 {
	return cr.n
}

// ChecksumPool manages reusable checksum hashers to reduce allocations.
type ChecksumPool struct {
	pool sync.Pool
}

// NewChecksumPool creates a new ChecksumPool.
func NewChecksumPool() *ChecksumPool {
	return &ChecksumPool{
		pool: sync.Pool{
			New: func() any {
				return crc64.New(crc64.MakeTable(crc64.ISO))
			},
		},
	}
}

// Get retrieves a hasher from the pool.
func (cp *ChecksumPool) Get() hash.Hash64 {
	return cp.pool.Get().(hash.Hash64)
}

// Put returns a hasher to the pool after resetting it.
func (cp *ChecksumPool) Put(h hash.Hash64) {
	h.Reset()
	cp.pool.Put(h)
}

// VerifyChecksum compares the checksum of data against an expected value.
func VerifyChecksum(actual, expected uint64) bool {
	return actual == expected
}
