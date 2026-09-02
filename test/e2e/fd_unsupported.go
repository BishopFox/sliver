//go:build !darwin && !linux

package e2e

import (
	"context"
	"errors"
)

func readBoundedE2EFileDescriptor(_ context.Context, _ int, _ int) ([]byte, error) {
	return nil, errors.New("runtime descriptors are supported only on Linux and macOS drivers")
}
