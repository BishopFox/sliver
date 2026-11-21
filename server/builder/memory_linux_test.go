//go:build linux

package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMemoryFixture(t *testing.T, dir, limitName, usageName, limit, usage string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{limitName: limit, usageName: usage} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMemoryCgroupV2UsesProcessPathAndTightestAncestor(t *testing.T) {
	root := t.TempDir()
	writeMemoryFixture(t, root, "memory.max", "memory.current", "1000\n", "600\n")
	writeMemoryFixture(t, filepath.Join(root, "parent"), "memory.max", "memory.current", "900\n", "700\n")
	writeMemoryFixture(t, filepath.Join(root, "parent", "child"), "memory.max", "memory.current", "800\n", "500\n")
	mountinfo := "31 30 0:22 / " + root + " rw - cgroup2 cgroup rw\n"
	remaining, limited, reliable := memoryRemainingFromCgroup("0::/parent/child\n", mountinfo)
	if !reliable || !limited || remaining != 200 {
		t.Fatalf("nested v2 remaining = %d, limited=%t, reliable=%t; want 200,true,true", remaining, limited, reliable)
	}
	if _, _, reliable := memoryRemainingFromCgroup("0::/outside\n", "31 30 0:22 /parent "+root+" rw - cgroup2 cgroup rw\n"); reliable {
		t.Fatal("unresolved process cgroup was treated as reliable")
	}
}

func TestMemoryCgroupV1UsesProcessPathAndFailsClosedOnMissingUsage(t *testing.T) {
	root := t.TempDir()
	writeMemoryFixture(t, root, "memory.limit_in_bytes", "memory.usage_in_bytes", "1000\n", "500\n")
	writeMemoryFixture(t, filepath.Join(root, "team"), "memory.limit_in_bytes", "memory.usage_in_bytes", "700\n", "400\n")
	writeMemoryFixture(t, filepath.Join(root, "team", "job"), "memory.limit_in_bytes", "memory.usage_in_bytes", "600\n", "550\n")
	mountinfo := "32 30 0:23 /tenant " + root + " rw - cgroup cgroup rw,memory\n"
	cgroups := "2:cpu,cpuacct:/cpu/team\n3:memory:/tenant/team/job\n"
	remaining, limited, reliable := memoryRemainingFromCgroup(cgroups, mountinfo)
	if !reliable || !limited || remaining != 50 {
		t.Fatalf("nested v1 remaining = %d, limited=%t, reliable=%t; want 50,true,true", remaining, limited, reliable)
	}
	if err := os.Remove(filepath.Join(root, "team", "memory.usage_in_bytes")); err != nil {
		t.Fatal(err)
	}
	if _, _, reliable := memoryRemainingFromCgroup(cgroups, mountinfo); reliable {
		t.Fatal("known ancestor limit without usage was treated as reliable")
	}
}

func TestMemoryCgroupUnlimitedAndMissingController(t *testing.T) {
	root := t.TempDir()
	writeMemoryFixture(t, root, "memory.max", "memory.current", "max\n", "100\n")
	mountinfo := "31 30 0:22 / " + root + " rw - cgroup2 cgroup rw\n"
	if remaining, limited, reliable := memoryRemainingFromCgroup("0::/\n", mountinfo); !reliable || limited || remaining != 0 {
		t.Fatalf("unlimited memory cgroup: %d,%t,%t", remaining, limited, reliable)
	}
	if remaining, limited, reliable := memoryRemainingFromCgroup("2:cpu:/foo\n", mountinfo); !reliable || limited || remaining != 0 {
		t.Fatalf("no memory controller: %d,%t,%t", remaining, limited, reliable)
	}
}

func TestMemoryCgroupMalformedLimitFailsClosed(t *testing.T) {
	root := t.TempDir()
	writeMemoryFixture(t, root, "memory.max", "memory.current", "invalid\n", "100\n")
	mountinfo := "31 30 0:22 / " + root + " rw - cgroup2 cgroup rw\n"
	if _, _, reliable := memoryRemainingFromCgroup("0::/\n", mountinfo); reliable {
		t.Fatal("malformed cgroup memory limit was treated as reliable")
	}
}
