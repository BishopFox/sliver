package ps

import (
	"os"
	"runtime"
	"testing"
)

func TestGetProcessArchitectureCurrentProcess(t *testing.T) {
	expected := map[string]string{
		"386":   "x86",
		"amd64": "x86_64",
		"arm64": "arm64",
	}[runtime.GOARCH]
	if expected == "" {
		t.Skipf("unsupported test architecture %s", runtime.GOARCH)
	}

	actual, err := getProcessArchitecture(uint32(os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("got %q, want %q", actual, expected)
	}
}
