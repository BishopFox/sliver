//go:build linux

package e2e

import (
	"fmt"
	"os"
	"syscall"
)

func accessTimeUnix(info os.FileInfo) (int64, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("unexpected Linux stat type %T", info.Sys())
	}
	return int64(stat.Atim.Sec), nil
}
