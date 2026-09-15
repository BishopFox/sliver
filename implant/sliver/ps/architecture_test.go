// Package ps provides process enumeration and architecture detection.
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

func TestArchitectureFromWindowsMachines(t *testing.T) {
	tests := []struct {
		name           string
		processMachine uint16
		nativeMachine  uint16
		expected       string
	}{
		{name: "x86 on amd64", processMachine: 0x014c, nativeMachine: 0x8664, expected: "x86"},
		{name: "x86 on arm64", processMachine: 0x014c, nativeMachine: 0xaa64, expected: "x86"},
		{name: "native amd64", nativeMachine: 0x8664, expected: "x86_64"},
		{name: "native arm64", nativeMachine: 0xaa64, expected: "arm64"},
		{name: "exact amd64", processMachine: 0x8664, nativeMachine: 0xaa64, expected: "x86_64"},
		{name: "unknown", processMachine: 0xffff, nativeMachine: 0xaa64, expected: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := architectureFromWindowsMachines(test.processMachine, test.nativeMachine)
			if actual != test.expected {
				t.Fatalf("got %q, want %q", actual, test.expected)
			}
		})
	}
}
