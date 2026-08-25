//go:build (linux && !android && (386 || amd64)) || (freebsd && amd64)

package linuxmem

func flushMappedInstructionCache(mapping []byte) error {
	_ = mapping
	return nil
}
