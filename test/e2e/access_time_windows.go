//go:build windows

package e2e

import (
	"fmt"
	"os"
	"syscall"
)

func accessTimeUnix(info os.FileInfo) (int64, error) {
	stat, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return 0, fmt.Errorf("unexpected Windows stat type %T", info.Sys())
	}
	return stat.LastAccessTime.Nanoseconds() / 1_000_000_000, nil
}
