//go:build !solaris
// +build !solaris

package lscolors

import "os"

func isDoor(info os.FileInfo) bool {
	// Doors are only supported on Solaris.
	return false
}
