//go:build !solaris

package lscolors

import "os"

func isDoor(_ os.FileInfo) bool {
	// Doors are only supported on Solaris.
	return false
}
