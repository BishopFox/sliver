//go:build !linux && !windows

package builder

// Other platforms have no Linux cgroup quota. runtime.NumCPU supplies their
// process-visible count, and GOMAXPROCS is applied by the shared combiner.
func osCPULimits() (affinity, quota int, quotaKnown bool) { return 0, 0, false }
