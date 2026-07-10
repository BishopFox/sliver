package leaky

import "testing"

func TestGetReturnsCorrectlySizedBuffer(t *testing.T) {
	const bufSize = 4096
	lb := NewLeakyBuf(2, bufSize)

	b := lb.Get()
	if len(b) != bufSize {
		t.Fatalf("Get returned a buffer of size %d, want %d", len(b), bufSize)
	}
}

func TestPutThenGetReusesBuffer(t *testing.T) {
	const bufSize = 8
	lb := NewLeakyBuf(1, bufSize)

	original := lb.Get()
	original[0] = 0x42
	lb.Put(original)

	reused := lb.Get()
	if &reused[0] != &original[0] {
		t.Fatalf("expected Get to return the pooled buffer after Put")
	}
	if reused[0] != 0x42 {
		t.Fatalf("expected pooled buffer to retain its contents")
	}
}

func TestPutBeyondCapacityDoesNotBlock(t *testing.T) {
	const bufSize = 8
	lb := NewLeakyBuf(1, bufSize)

	// Putting more buffers than the pool can hold must silently drop the
	// overflow rather than block.
	lb.Put(make([]byte, bufSize))
	lb.Put(make([]byte, bufSize))
	lb.Put(make([]byte, bufSize))
}

func TestPutWrongSizePanics(t *testing.T) {
	const bufSize = 8
	lb := NewLeakyBuf(1, bufSize)

	defer func() {
		if recover() == nil {
			t.Fatalf("expected Put with a wrong-sized buffer to panic")
		}
	}()
	lb.Put(make([]byte, bufSize+1))
}

func TestGetOnEmptyPoolAllocates(t *testing.T) {
	const bufSize = 16
	lb := NewLeakyBuf(4, bufSize)

	// Nothing has been Put yet, so each Get must allocate a fresh buffer of
	// the configured size.
	for i := 0; i < 3; i++ {
		if b := lb.Get(); len(b) != bufSize {
			t.Fatalf("Get on an empty pool returned size %d, want %d", len(b), bufSize)
		}
	}
}
