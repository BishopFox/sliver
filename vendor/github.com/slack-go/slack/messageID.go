package slack

import "sync/atomic"

// IDGenerator provides an interface for generating integer ID values.
type IDGenerator interface {
	Next() int
}

// NewSafeID returns a new instance of an IDGenerator which is safe for
// concurrent use by multiple goroutines.
func NewSafeID(startID int) IDGenerator {
	return &safeID{
		nextID: int64(startID),
	}
}

type safeID struct {
	nextID int64
}

// make sure safeID implements the IDGenerator interface.
var _ IDGenerator = (*safeID)(nil)

// Next implements IDGenerator.Next.
func (s *safeID) Next() int {
	id := atomic.AddInt64(&s.nextID, 1)

	return int(id)
}
