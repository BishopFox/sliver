package javaser

import "fmt"

// ProtocolError reports structurally invalid or deliberately unsupported Java
// serialization data. Offset is the number of bytes consumed when detected.
type ProtocolError struct {
	Offset  int64
	Message string
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("java serialization protocol error at byte %d: %s", e.Offset, e.Message)
}

// LimitError reports a configured decoding or encoding bound.
type LimitError struct {
	Resource string
	Limit    int64
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("java serialization %s limit exceeded (%d)", e.Resource, e.Limit)
}
