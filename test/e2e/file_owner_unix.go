//go:build darwin || linux

package e2e

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

func fileOwnerIDs(info os.FileInfo) (string, string, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", "", fmt.Errorf("unexpected file metadata type %T", info.Sys())
	}
	return strconv.FormatUint(uint64(stat.Uid), 10), strconv.FormatUint(uint64(stat.Gid), 10), nil
}
