// Copyright (c) 2025 Toni Spets
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exhttp

import (
	"errors"
	"net"
	"syscall"

	"golang.org/x/net/http2"
)

func IsNetworkError(err error) bool {
	if errno := syscall.Errno(0); errors.As(err, &errno) {
		// common errnos for network related operations
		return errno == syscall.ENETDOWN ||
			errno == syscall.ENETUNREACH ||
			errno == syscall.ENETRESET ||
			errno == syscall.ECONNABORTED ||
			errno == syscall.ECONNRESET ||
			errno == syscall.ENOBUFS ||
			errno == syscall.ETIMEDOUT ||
			errno == syscall.ECONNREFUSED ||
			errno == syscall.EHOSTDOWN ||
			errno == syscall.EHOSTUNREACH ||
			errno == syscall.EPIPE
	} else if netError := net.Error(nil); errors.As(err, &netError) {
		return true
	} else if errors.As(err, &http2.StreamError{}) {
		return true
	}

	return false
}
