//go:build !linux && !darwin && !windows

package builder

func availableMemoryBytes() uint64 { return 0 }
