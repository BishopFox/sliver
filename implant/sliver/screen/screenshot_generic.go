//go:build !windows && !linux

package screen

// Screenshot - Retrieve the screenshot of the active displays
func Screenshot() []byte {
	return []byte{}
}
