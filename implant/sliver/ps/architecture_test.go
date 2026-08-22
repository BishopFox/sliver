package ps

import "testing"

func TestArchitectureFromELFMachine(t *testing.T) {
	tests := map[uint16]string{
		3:   "x86",
		62:  "x86_64",
		183: "aarch64",
		179: "",
	}
	for machine, expected := range tests {
		if actual := architectureFromELFMachine(machine); actual != expected {
			t.Errorf("machine %d: got %q, want %q", machine, actual, expected)
		}
	}
}
