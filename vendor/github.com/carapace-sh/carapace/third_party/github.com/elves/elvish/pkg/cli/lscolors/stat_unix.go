//go:build !windows && !plan9
// +build !windows,!plan9

package lscolors

import (
	"os"
	"syscall"
)

func isMultiHardlink(info os.FileInfo) bool {
	return info.Sys().(*syscall.Stat_t).Nlink > 1
}
