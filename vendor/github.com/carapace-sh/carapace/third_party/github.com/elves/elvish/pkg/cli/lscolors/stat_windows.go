package lscolors

import "os"

func isMultiHardlink(_ os.FileInfo) bool {
	// Windows supports hardlinks, but it is not exposed directly. We omit the
	// implementation for now.
	// TODO: Maybe implement it?
	return false
}
