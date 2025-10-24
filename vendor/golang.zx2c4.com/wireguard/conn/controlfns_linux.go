/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2025 WireGuard LLC. All Rights Reserved.
 */

package conn

import (
	"fmt"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

// Taken from go/src/internal/syscall/unix/kernel_version_linux.go
func kernelVersion() (major, minor int) {
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return
	}

	var (
		values    [2]int
		value, vi int
	)
	for _, c := range uname.Release {
		if '0' <= c && c <= '9' {
			value = (value * 10) + int(c-'0')
		} else {
			// Note that we're assuming N.N.N here.
			// If we see anything else, we are likely to mis-parse it.
			values[vi] = value
			vi++
			if vi >= len(values) {
				break
			}
			value = 0
		}
	}

	return values[0], values[1]
}

func init() {
	controlFns = append(controlFns,

		// Attempt to set the socket buffer size beyond net.core.{r,w}mem_max by
		// using SO_*BUFFORCE. This requires CAP_NET_ADMIN, and is allowed here to
		// fail silently - the result of failure is lower performance on very fast
		// links or high latency links.
		func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				// Set up to *mem_max
				_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_RCVBUF, socketBufferSize)
				_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_SNDBUF, socketBufferSize)
				// Set beyond *mem_max if CAP_NET_ADMIN
				_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_RCVBUFFORCE, socketBufferSize)
				_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_SNDBUFFORCE, socketBufferSize)
			})
		},

		// Enable receiving of the packet information (IP_PKTINFO for IPv4,
		// IPV6_PKTINFO for IPv6) that is used to implement sticky socket support.
		func(network, address string, c syscall.RawConn) error {
			var err error
			switch network {
			case "udp4":
				if runtime.GOOS != "android" {
					c.Control(func(fd uintptr) {
						err = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_PKTINFO, 1)
					})
				}
			case "udp6":
				c.Control(func(fd uintptr) {
					if runtime.GOOS != "android" {
						err = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_RECVPKTINFO, 1)
						if err != nil {
							return
						}
					}
					err = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_V6ONLY, 1)
				})
			default:
				err = fmt.Errorf("unhandled network: %s: %w", network, unix.EINVAL)
			}
			return err
		},

		// Attempt to enable UDP_GRO
		func(network, address string, c syscall.RawConn) error {
			// Kernels below 5.12 are missing 98184612aca0 ("net:
			// udp: Add support for getsockopt(..., ..., UDP_GRO,
			// ..., ...);"), which means we can't read this back
			// later. We could pipe the return value through to
			// the rest of the code, but UDP_GRO is kind of buggy
			// anyway, so just gate this here.
			major, minor := kernelVersion()
			if major < 5 || (major == 5 && minor < 12) {
				return nil
			}

			c.Control(func(fd uintptr) {
				_ = unix.SetsockoptInt(int(fd), unix.IPPROTO_UDP, unix.UDP_GRO, 1)
			})
			return nil
		},
	)
}
