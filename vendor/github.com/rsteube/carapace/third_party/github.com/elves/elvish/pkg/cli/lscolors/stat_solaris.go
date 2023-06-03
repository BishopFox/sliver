package lscolors

import (
	"os"
	"syscall"
)

// Taken from Illumos header file.
const sIFDOOR = 0xD000

func isDoor(info os.FileInfo) bool {
	return info.Sys().(*syscall.Stat_t).Mode&sIFDOOR == sIFDOOR
}
