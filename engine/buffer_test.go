package engine

import (
	"testing"
)

func TestBufferPool_DefaultSize(t *testing.T) {
	bp := NewBufferPool(0)

	buf := bp.Get()
	if buf == nil {
		t.Fatalf("expected a valid buffer pointer, got nil")
	}

	if len(*buf) != DefaultBufferSize {
		t.Errorf("expected buffer size %d, got %d", DefaultBufferSize, len(*buf))
	}

	bp.Put(buf)
}

func TestBufferPool_CustomSize(t *testing.T) {
	customSize := 8192 // 8KB
	bp := NewBufferPool(customSize)

	buf1 := bp.Get()
	if len(*buf1) != customSize {
		t.Errorf("expected buffer size %d, got %d", customSize, len(*buf1))
	}

	// modify the buffer
	(*buf1)[0] = 42

	// put it back and retrieve
	bp.Put(buf1)
	buf2 := bp.Get()

	// the underlying array might be the same, verify the length is correct
	if len(*buf2) != customSize {
		t.Errorf("expected reused buffer size %d, got %d", customSize, len(*buf2))
	}

	bp.Put(buf2)
}
