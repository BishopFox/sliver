//go:build darwin

package ps

import (
	"os"
	"testing"
)

func TestParseKinfoProcsIncludesEveryCompleteRecord(t *testing.T) {
	raw := make([]byte, 2*_KINFO_STRUCT_SIZE+_KINFO_STRUCT_SIZE/2)

	procs := parseKinfoProcs(raw)
	if got, want := len(procs), 2; got != want {
		t.Fatalf("parseKinfoProcs() returned %d records, want %d", got, want)
	}
}

func TestProcessesIncludesSelf(t *testing.T) {
	procs, err := processes(false)
	if err != nil {
		t.Fatalf("processes(false) returned an error: %v", err)
	}

	self := os.Getpid()
	for _, proc := range procs {
		if proc.Pid() == self {
			return
		}
	}
	t.Fatalf("processes(false) did not contain current PID %d", self)
}
