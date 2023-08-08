// Package close allows experimental.CloseNotifier without introducing a
// package cycle.
package close

import "context"

// NotifierKey is a context.Context Value key. Its associated value should be a
// Notifier.
type NotifierKey struct{}

type Notifier interface {
	CloseNotify(ctx context.Context, exitCode uint32)
}
