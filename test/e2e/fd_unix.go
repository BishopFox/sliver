//go:build darwin || linux

package e2e

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sys/unix"
)

//nolint:gocyclo // Descriptor polling keeps deadline, EINTR, HUP, and bounded-read handling in one ownership loop.
func readBoundedE2EFileDescriptor(ctx context.Context, descriptor int, maximum int) ([]byte, error) {
	descriptorFlags, err := unix.FcntlInt(uintptr(descriptor), unix.F_GETFD, 0)
	if err != nil {
		return nil, err
	}
	// The caller retains ownership of the supplied descriptor, but it may have
	// arrived from a shell without FD_CLOEXEC. Mark the original before any later
	// server, implant, or application child is spawned so seekable runtime-only
	// inputs cannot be rewound and recovered there.
	if _, err := unix.FcntlInt(uintptr(descriptor), unix.F_SETFD, descriptorFlags|unix.FD_CLOEXEC); err != nil {
		return nil, err
	}
	duplicatedDescriptor, err := unix.FcntlInt(uintptr(descriptor), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer func() { _ = unix.Close(duplicatedDescriptor) }()
	pollDescriptors := []unix.PollFd{{Fd: int32(duplicatedDescriptor), Events: unix.POLLIN | unix.POLLHUP}}
	payload := make([]byte, 0, maximum)
	buffer := make([]byte, 4096)
	for len(payload) < maximum {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pollTimeout := 100 * time.Millisecond
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				return nil, context.DeadlineExceeded
			}
			if remaining < pollTimeout {
				pollTimeout = remaining
			}
		}
		pollDescriptors[0].Revents = 0
		ready, err := unix.Poll(pollDescriptors, int(pollTimeout.Milliseconds()))
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return nil, err
		}
		if ready == 0 {
			continue
		}
		if pollDescriptors[0].Revents&(unix.POLLIN|unix.POLLHUP) == 0 {
			continue
		}
		remaining := maximum - len(payload)
		if remaining < len(buffer) {
			buffer = buffer[:remaining]
		}
		read, err := unix.Read(duplicatedDescriptor, buffer)
		if read > 0 {
			payload = append(payload, buffer[:read]...)
		}
		if err != nil {
			if errors.Is(err, unix.EINTR) || errors.Is(err, unix.EAGAIN) {
				continue
			}
			return nil, err
		}
		if read == 0 {
			return payload, nil
		}
	}
	return payload, nil
}
