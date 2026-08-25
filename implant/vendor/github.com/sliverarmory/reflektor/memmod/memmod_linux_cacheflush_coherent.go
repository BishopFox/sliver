//go:build (linux && !android && (386 || amd64)) || (freebsd && amd64)

package memmod

func flushLinuxInstructionCache(start, end uintptr) error {
	_, _ = start, end
	return nil
}
