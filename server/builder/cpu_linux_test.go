//go:build linux

package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCPUQuotaParsing(t *testing.T) {
	for _, tt := range []struct {
		quota, period int64
		want          int
		known         bool
	}{{150000, 100000, 1, true}, {50000, 100000, 1, true}, {250000, 100000, 2, true}, {0, 100000, 0, true}, {-1, 100000, 0, false}, {100000, 0, 0, false}} {
		got, known := quotaToCores(tt.quota, tt.period)
		if got != tt.want || known != tt.known {
			t.Fatalf("quota %d/%d: got %d,%t, want %d,%t", tt.quota, tt.period, got, known, tt.want, tt.known)
		}
	}
}

func TestCPUCgroupMountAndAncestorQuota(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	if err := os.MkdirAll(child, 0700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		filepath.Join(root, "cpu.max"):   "max 100000\n",
		filepath.Join(parent, "cpu.max"): "150000 100000\n",
		filepath.Join(child, "cpu.max"):  "350000 100000\n",
	} {
		if err := os.WriteFile(name, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	v2, v1 := parseCPUCgroupPaths("0::/parent/child\n2:cpu,cpuacct:/legacy\n")
	if v2 != "/parent/child" || v1 != "/legacy" {
		t.Fatalf("cgroup paths: %q %q", v2, v1)
	}
	mounts := parseCPUCgroupMounts("31 30 0:22 / " + root + " rw - cgroup2 cgroup rw\n")
	if len(mounts) != 1 {
		t.Fatalf("cgroup mounts: %+v", mounts)
	}
	dir, ok := resolveCPUCgroupDir(mounts[0], v2)
	if !ok || dir != child {
		t.Fatalf("resolved cgroup directory: %q %t", dir, ok)
	}
	if got, known := readCgroupQuotaAncestors(mounts[0], dir); !known || got != 1 {
		t.Fatalf("ancestor quota: got %d,%t, want 1,true", got, known)
	}
	v1Mounts := parseCPUCgroupMounts("32 30 0:23 /legacy " + root + " rw - cgroup cgroup rw,cpu,cpuacct\n")
	if len(v1Mounts) != 1 || v1Mounts[0].v2 {
		t.Fatalf("v1 CPU mount was not recognized: %+v", v1Mounts)
	}
	if resolved, ok := resolveCPUCgroupDir(v1Mounts[0], "/legacy/parent"); !ok || resolved != parent {
		t.Fatalf("v1 CPU cgroup path: %q %t", resolved, ok)
	}
}
