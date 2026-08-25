//go:build (darwin && !ios && amd64) || (freebsd && amd64) || (linux && !android && (386 || amd64))

package bofloader

func (region *memoryRegion) flushInstructionCache(offset, length int) error {
	_, err := region.rangeBytes(offset, length)
	return err
}
