package builder

import "runtime"

// effectiveCPUCores reports the CPUs this process can actually use. An
// absent OS limit is ignored. An explicit zero CPU quota is reported as zero
// so registration/status validation rejects an unusable builder.
func effectiveCPUCores() uint32 {
	affinity, quota, quotaKnown := osCPULimits()
	count := combineCPULimits(runtime.NumCPU(), runtime.GOMAXPROCS(0), affinity, quota, quotaKnown)
	if count < 0 {
		return 0
	}
	return uint32(count)
}

func combineCPULimits(host, concurrency, affinity, quota int, quotaKnown bool) int {
	if host <= 0 || concurrency <= 0 {
		return 0
	}
	count := min(host, concurrency)
	if affinity > 0 {
		count = min(count, affinity)
	}
	if quotaKnown {
		if quota <= 0 {
			return 0
		}
		count = min(count, quota)
	}
	return count
}
