//go:build linux

package builder

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type cpuCgroupMount struct {
	root       string
	mountpoint string
	v2         bool
}

// osCPULimits checks the kernel's effective affinity mask (which includes
// cpuset restrictions) and the tightest applicable cgroup CPU quota.
func osCPULimits() (affinity, quota int, quotaKnown bool) {
	affinity = linuxAffinityCores()
	quota, quotaKnown = linuxCgroupQuotaCores()
	return affinity, quota, quotaKnown
}

func linuxAffinityCores() int {
	for maxCPU := 1024; maxCPU <= 65536; maxCPU *= 2 {
		set := unix.NewCPUSet(maxCPU)
		if err := unix.SchedGetaffinityDynamic(0, set); err == nil {
			return set.Count()
		} else if !errors.Is(err, unix.EINVAL) {
			return 0 // Unknown; runtime.NumCPU still bounds the report.
		}
	}
	return 0
}

func linuxCgroupQuotaCores() (quota int, quotaKnown bool) {
	cgroups, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return 0, false
	}
	mountinfo, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return 0, false
	}
	v2Path, v1Path := parseCPUCgroupPaths(string(cgroups))
	for _, mount := range parseCPUCgroupMounts(string(mountinfo)) {
		cgroupPath := v1Path
		if mount.v2 {
			cgroupPath = v2Path
		}
		if cgroupPath == "" {
			continue
		}
		dir, ok := resolveCPUCgroupDir(mount, cgroupPath)
		if !ok {
			continue
		}
		count, known := readCgroupQuotaAncestors(mount, dir)
		if known && (!quotaKnown || count < quota) {
			quota, quotaKnown = count, true
		}
	}
	return quota, quotaKnown
}

func readCgroupQuotaAncestors(mount cpuCgroupMount, dir string) (quota int, quotaKnown bool) {
	for {
		var count int
		var known bool
		if mount.v2 {
			count, known = readV2CPUQuota(filepath.Join(dir, "cpu.max"))
		} else {
			count, known = readV1CPUQuota(dir)
		}
		if known && (!quotaKnown || count < quota) {
			quota, quotaKnown = count, true
		}
		if dir == mount.mountpoint {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir || !insideDir(mount.mountpoint, parent) {
			break
		}
		dir = parent
	}
	return quota, quotaKnown
}

func parseCPUCgroupPaths(content string) (v2, v1 string) {
	for _, line := range strings.Split(content, "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 || !strings.HasPrefix(parts[2], "/") {
			continue
		}
		if parts[0] == "0" && parts[1] == "" {
			v2 = path.Clean(parts[2])
		}
		for _, controller := range strings.Split(parts[1], ",") {
			if controller == "cpu" {
				v1 = path.Clean(parts[2])
				break
			}
		}
	}
	return v2, v1
}

func parseCPUCgroupMounts(content string) []cpuCgroupMount {
	var mounts []cpuCgroupMount
	for _, line := range strings.Split(content, "\n") {
		sides := strings.SplitN(line, " - ", 2)
		if len(sides) != 2 {
			continue
		}
		left, right := strings.Fields(sides[0]), strings.Fields(sides[1])
		if len(left) < 5 || len(right) < 3 {
			continue
		}
		v2 := right[0] == "cgroup2"
		if !v2 && (right[0] != "cgroup" || !strings.Contains(","+right[2]+",", ",cpu,")) {
			continue
		}
		root, mountpoint := unescapeMountField(left[3]), unescapeMountField(left[4])
		if !path.IsAbs(root) || !filepath.IsAbs(mountpoint) {
			continue
		}
		mounts = append(mounts, cpuCgroupMount{root: path.Clean(root), mountpoint: filepath.Clean(mountpoint), v2: v2})
	}
	return mounts
}

func unescapeMountField(field string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return replacer.Replace(field)
}

func resolveCPUCgroupDir(mount cpuCgroupMount, cgroupPath string) (string, bool) {
	if !path.IsAbs(cgroupPath) {
		return "", false
	}
	clean := path.Clean(cgroupPath)
	root := path.Clean(mount.root)
	var relative string
	switch {
	case root == "/":
		relative = strings.TrimPrefix(clean, "/")
	case clean == root || clean == "/":
		// A cgroup namespace may present the mount root as '/'.
		relative = ""
	case strings.HasPrefix(clean, root+"/"):
		relative = strings.TrimPrefix(clean, root+"/")
	default:
		return "", false
	}
	dir := filepath.Join(mount.mountpoint, filepath.FromSlash(relative))
	return dir, insideDir(mount.mountpoint, dir)
}

func insideDir(root, child string) bool {
	relative, err := filepath.Rel(root, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func readV2CPUQuota(file string) (int, bool) {
	data, err := os.ReadFile(file)
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(data))
	if len(fields) != 2 || fields[0] == "max" {
		return 0, false
	}
	quota, err1 := strconv.ParseInt(fields[0], 10, 64)
	period, err2 := strconv.ParseInt(fields[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return quotaToCores(quota, period)
}

func readV1CPUQuota(dir string) (int, bool) {
	quotaData, err1 := os.ReadFile(filepath.Join(dir, "cpu.cfs_quota_us"))
	periodData, err2 := os.ReadFile(filepath.Join(dir, "cpu.cfs_period_us"))
	if err1 != nil || err2 != nil {
		return 0, false
	}
	quota, err1 := strconv.ParseInt(strings.TrimSpace(string(quotaData)), 10, 64)
	period, err2 := strconv.ParseInt(strings.TrimSpace(string(periodData)), 10, 64)
	if err1 != nil || err2 != nil || quota < 0 {
		return 0, false
	}
	return quotaToCores(quota, period)
}

func quotaToCores(quota, period int64) (int, bool) {
	if quota < 0 || period <= 0 {
		return 0, false
	}
	if quota == 0 {
		return 0, true
	}
	cores := quota / period // Round down; a positive fractional quota uses one.
	if cores == 0 {
		cores = 1
	}
	const maxInt = int(^uint(0) >> 1)
	if cores > int64(maxInt) {
		return maxInt, true
	}
	return int(cores), true
}
