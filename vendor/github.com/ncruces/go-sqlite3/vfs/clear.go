//go:build !go1.21

package vfs

func clear(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
