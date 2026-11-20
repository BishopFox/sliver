package builder

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// /proc/meminfo's MemAvailable estimates memory usable without swapping.
// A missing or untrustworthy reading makes the builder unavailable.
func availableMemoryBytes() uint64 {
	hostAvailable := hostAvailableMemoryBytes()
	if hostAvailable == 0 {
		return 0
	}
	cgroups, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return 0
	}
	mountinfo, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return 0
	}
	remaining, limited, reliable := memoryRemainingFromCgroup(string(cgroups), string(mountinfo))
	if !reliable {
		return 0
	}
	if limited && remaining < hostAvailable {
		return remaining
	}
	return hostAvailable
}

func hostAvailableMemoryBytes() uint64 {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemAvailable:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[2] != "kB" {
			return 0
		}
		kilobytes, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil || kilobytes > ^uint64(0)/1024 {
			return 0
		}
		return kilobytes * 1024
	}
	return 0
}

// memoryRemainingFromCgroup resolves the calling process's cgroup namespace
// path against its memory mounts. Every applicable ancestor may impose a
// tighter limit. A known limit without a readable usage is untrustworthy.
func memoryRemainingFromCgroup(cgroups, mountinfo string) (remaining uint64, limited, reliable bool) {
	v2Path, v1Path := parseMemoryCgroupPaths(cgroups)
	if v2Path == "" && v1Path == "" {
		return 0, false, true // No memory controller applies.
	}
	matched := false
	for _, mount := range parseMemoryCgroupMounts(mountinfo) {
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
		matched = true
		available, hasLimit, ok := readMemoryCgroupAncestors(mount, dir)
		if !ok {
			return 0, false, false
		}
		if hasLimit && (!limited || available < remaining) {
			remaining, limited = available, true
		}
	}
	if !matched {
		return 0, false, false
	}
	return remaining, limited, true
}

func parseMemoryCgroupPaths(content string) (v2, v1 string) {
	for _, line := range strings.Split(content, "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 || !path.IsAbs(parts[2]) {
			continue
		}
		if parts[0] == "0" && parts[1] == "" {
			v2 = path.Clean(parts[2])
		}
		for _, controller := range strings.Split(parts[1], ",") {
			if controller == "memory" {
				v1 = path.Clean(parts[2])
				break
			}
		}
	}
	return v2, v1
}

func parseMemoryCgroupMounts(content string) []cpuCgroupMount {
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
		if !v2 && (right[0] != "cgroup" || !strings.Contains(","+right[2]+",", ",memory,")) {
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

func readMemoryCgroupAncestors(mount cpuCgroupMount, dir string) (remaining uint64, limited, reliable bool) {
	for {
		limitName, usageName := "memory.limit_in_bytes", "memory.usage_in_bytes"
		if mount.v2 {
			limitName, usageName = "memory.max", "memory.current"
		}
		limit, hasLimit, limitOK := readMemoryLimit(filepath.Join(dir, limitName))
		if !limitOK {
			return 0, false, false
		}
		if hasLimit {
			usage, hasUsage := readCgroupBytes(filepath.Join(dir, usageName))
			if !hasUsage {
				return 0, false, false
			}
			available := uint64(0)
			if usage < limit {
				available = limit - usage
			}
			if !limited || available < remaining {
				remaining, limited = available, true
			}
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
	return remaining, limited, true
}

func readMemoryLimit(file string) (uint64, bool, bool) {
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return 0, false, true // This cgroup has no enabled memory controller.
	}
	if err != nil {
		return 0, false, false
	}
	value := strings.TrimSpace(string(data))
	if value == "max" {
		return 0, false, true
	}
	bytes, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false, false
	}
	return bytes, true, true
}

func readCgroupBytes(path string) (uint64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	value := strings.TrimSpace(string(data))
	if value == "max" {
		return 0, false
	}
	bytes, err := strconv.ParseUint(value, 10, 64)
	return bytes, err == nil
}
