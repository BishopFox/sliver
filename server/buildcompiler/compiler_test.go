package buildcompiler

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/server/gogo"
)

type testDriver struct {
	id    string
	calls int
	got   Request
}

func (driver *testDriver) ID() string { return driver.id }

func (driver *testDriver) Build(_ context.Context, request Request) ([]byte, error) {
	driver.calls++
	driver.got = request
	if len(request.Tags) != 0 {
		request.Tags[0] = "changed"
	}
	if len(request.LDFlags) != 0 {
		request.LDFlags[0] = "changed"
	}
	return []byte("compiler output"), nil
}

func TestRegistrySelectsInstalledDriverAndCopiesFlags(t *testing.T) {
	registry := NewRegistry()
	driver := &testDriver{id: "fixture"}
	if err := registry.Register(driver); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(driver); err == nil {
		t.Fatal("duplicate driver registration should fail")
	}
	request := Request{
		GoConfig: gogo.GoConfig{GOOS: "linux", GOARCH: "amd64"},
		Tags:     []string{"netgo"}, LDFlags: []string{"-s"},
	}
	output, err := registry.Build(context.Background(), "fixture", request)
	if err != nil || !bytes.Equal(output, []byte("compiler output")) || driver.calls != 1 {
		t.Fatalf("driver did not execute once: %q, %v, %d", output, err, driver.calls)
	}
	if driver.got.GoConfig.GOOS != "linux" || request.Tags[0] != "netgo" || request.LDFlags[0] != "-s" {
		t.Fatalf("compiler request was changed: %+v", request)
	}
	if _, err := registry.Build(context.Background(), "missing", request); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("unknown driver should fail closed: %v", err)
	}
}

func TestRegistryDoesNotExecuteCanceledBuild(t *testing.T) {
	registry := NewRegistry()
	driver := &testDriver{id: "fixture"}
	if err := registry.Register(driver); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := registry.Build(ctx, "fixture", Request{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled build returned %v", err)
	}
	if driver.calls != 0 {
		t.Fatal("canceled build reached compiler driver")
	}
}

func TestDefaultGoDriverBoundary(t *testing.T) {
	if _, err := Default().Build(context.Background(), GoDriverID, Request{}); err == nil || !strings.Contains(err.Error(), "source and destination") {
		t.Fatalf("Go driver did not validate typed request: %v", err)
	}
}

func TestGoDriverPreservesGoAndGarbleInvocation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake compiler fixture is a POSIX shell script")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	fixture := []byte("#!/bin/sh\nif [ \"$1\" = tool ]; then\n  echo linux/amd64\n  exit 0\nfi\nprintf '%s\\n' \"$0\" \"$@\" > \"$FAKE_COMPILER_LOG\"\nprintf 'CC=%s\\nCGO_ENABLED=%s\\nGOGARBLE=%s\\n' \"$CC\" \"$CGO_ENABLED\" \"$GOGARBLE\" >> \"$FAKE_COMPILER_LOG\"\nprevious=\nfor argument in \"$@\"; do\n  if [ \"$previous\" = -o ]; then printf fixture > \"$argument\"; fi\n  previous=\"$argument\"\ndone\n")
	for _, name := range []string{"go", "garble"} {
		if err := os.WriteFile(filepath.Join(bin, name), fixture, 0700); err != nil {
			t.Fatal(err)
		}
	}
	logPath := filepath.Join(root, "compiler.log")
	t.Setenv("FAKE_COMPILER_LOG", logPath)
	request := Request{
		GoConfig: gogo.GoConfig{
			ProjectDir: root, GOOS: "linux", GOARCH: "amd64", GOROOT: root,
			CGO: "1", CC: "zig cc", GOGARBLE: "github.com/bishopfox/sliver/implant/sliver",
		},
		SourceDir: root, Destination: filepath.Join(root, "bin", "payload.so"), BuildMode: "c-shared",
		Tags: []string{"netgo"}, LDFlags: []string{"-s -w"},
		ExtraEnv: []string{"FAKE_COMPILER_LOG=" + logPath},
	}
	assertInvocation := func(want ...string) {
		t.Helper()
		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range want {
			if !strings.Contains(string(data), fragment) {
				t.Fatalf("compiler invocation missing %q: %s", fragment, data)
			}
		}
	}
	if _, err := Default().Build(context.Background(), GoDriverID, request); err != nil {
		t.Fatal(err)
	}
	assertInvocation("/bin/go", "build", "-mod=vendor", "-buildmode=c-shared", "netgo", "-s -w", request.Destination, "CC=zig cc", "CGO_ENABLED=1")

	request.GoConfig.Obfuscation = true
	if _, err := Default().Build(context.Background(), GoDriverID, request); err != nil {
		t.Fatal(err)
	}
	assertInvocation("/bin/garble", "-seed=random", "-literals", "-tiny", "build", "GOGARBLE=github.com/bishopfox/sliver/implant/sliver")
}
