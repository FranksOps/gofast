package engine

import (
	"sync"
)

// DefaultBufferSize is the default size of byte buffers allocated for file transfers.
// 1MB is generally a good balance for modern fast I/O operations (network/disk).
const DefaultBufferSize = 1 * 1024 * 1024

// BufferPool manages reusable byte buffers to minimize GC overhead during
// multi-terabyte transfers.
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new BufferPool that allocates buffers of the specified size.
// If size is <= 0, DefaultBufferSize is used.
func NewBufferPool(size int) *BufferPool {
	if size <= 0 {
		size = DefaultBufferSize
	}
	return &BufferPool{
		pool: sync.Pool{
			New: func() any {
				b := make([]byte, size)
				return &b
			},
		},
	}
}

// Get retrieves a reusable byte buffer from the pool.
// The caller should defer calling Put on this buffer once finished.
func (bp *BufferPool) Get() *[]byte {
	return bp.pool.Get().(*[]byte)
}

// Put returns the byte buffer to the pool so it can be reused.
// The caller should not hold onto or read/write to the buffer after calling Put.
func (bp *BufferPool) Put(b *[]byte) {
	// A basic sanity check to avoid returning nil pointers.
	if b != nil {
		bp.pool.Put(b)
	}
}
