// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package wingoes

import (
	"errors"
	"math"
	"time"

	"golang.org/x/sys/windows"
)

// Infinite is the [time.Duration] equivalent of [windows.INFINITE].
const Infinite = time.Duration(math.MaxInt64)

var (
	// ErrDurationOutOfRange means that a time.Duration is too large to be able
	// to be specified as a valid Win32 timeout value.
	ErrDurationOutOfRange = errors.New("duration is out of timeout range")
)

// DurationToTimeoutMilliseconds converts d into a timeout usable by Win32 APIs.
func DurationToTimeoutMilliseconds(d time.Duration) (uint32, error) {
	if d == Infinite {
		return windows.INFINITE, nil
	}

	millis := d.Milliseconds()
	if millis >= windows.INFINITE {
		return 0, ErrDurationOutOfRange
	}
	return uint32(millis), nil
}
