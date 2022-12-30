//go:build openbsd || netbsd
// +build openbsd netbsd

package sysutil

import (
	"syscall"
)

type timeval syscall.Timeval
