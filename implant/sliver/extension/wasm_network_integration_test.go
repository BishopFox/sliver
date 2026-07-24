//go:build (windows && (arm64 || amd64 || 386)) || (darwin && (arm64 || amd64)) || (linux && (arm64 || amd64 || 386))

package extension

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWasmHTTPFetchExample(t *testing.T) {
	wasmPath := buildWasmTestModule(
		t,
		"SLIVER_WASM_HTTP_FETCH_WASM",
		"./implant/sliver/extension/testdata/http-fetch",
	)
	wasm, err := os.ReadFile(wasmPath)
	mustNoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(response, "local-network-ok")
	}))
	t.Cleanup(server.Close)
	stdout, stderr, exitCode, err := runWasmTestModule(t, wasm, []string{server.URL})
	mustNoError(t, err, string(stderr))
	mustZero(t, exitCode, string(stderr))
	assert.Contains(t, string(stdout), "200 OK")
	assert.Contains(t, string(stdout), "local-network-ok")

	if os.Getenv("SLIVER_WASM_LIVE_HTTPS") != "1" {
		return
	}
	stdout, stderr, exitCode, err = runWasmTestModule(t, wasm, nil)
	mustNoError(t, err, string(stderr))
	mustZero(t, exitCode, string(stderr))
	assert.Contains(t, string(stdout), "GET https://example.com/: 200 OK")
	assert.Contains(t, strings.ToLower(string(stdout)), "example domain")
}

func TestWasmNetworkGuestRoundTrip(t *testing.T) {
	wasmPath := buildWasmTestModule(
		t,
		"SLIVER_WASM_NETWORK_SMOKE_WASM",
		"./implant/sliver/extension/testdata/network-smoke",
	)
	wasm, err := os.ReadFile(wasmPath)
	mustNoError(t, err)

	stdout, stderr, exitCode, err := runWasmTestModule(t, wasm, nil)
	mustNoError(t, err, string(stderr))
	mustZero(t, exitCode, string(stderr))
	assert.Contains(t, string(stdout), "network-smoke-ok")
}

func buildWasmTestModule(t *testing.T, environmentName, packagePath string) string {
	t.Helper()
	if path := os.Getenv(environmentName); path != "" {
		return path
	}
	rootAppDir := os.Getenv("SLIVER_ROOT_DIR")
	if rootAppDir == "" {
		t.Skipf("set %s or run through ./go-tests.sh with unpacked compiler assets", environmentName)
	}
	goName := "go"
	wrapperName := "sliver-wasm-go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
		wrapperName += ".exe"
	}
	bundledGo := filepath.Join(rootAppDir, "go", "bin", goName)
	if _, err := os.Stat(bundledGo); err != nil {
		t.Skipf("bundled Go toolchain is not available at %s: %v", bundledGo, err)
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve repository root")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	wrapperPath := filepath.Join(rootAppDir, "go", "bin", wrapperName)
	if _, err := os.Stat(wrapperPath); os.IsNotExist(err) {
		wrapperPattern := ".sliver-wasm-go-test-*"
		if runtime.GOOS == "windows" {
			wrapperPattern += ".exe"
		}
		wrapperFile, createErr := os.CreateTemp(filepath.Dir(wrapperPath), wrapperPattern)
		if createErr != nil {
			t.Fatalf("create temporary wrapper path: %v", createErr)
		}
		wrapperPath = wrapperFile.Name()
		if closeErr := wrapperFile.Close(); closeErr != nil {
			t.Fatalf("close temporary wrapper path: %v", closeErr)
		}
		t.Cleanup(func() {
			_ = os.Remove(wrapperPath)
		})
		command := exec.Command(
			bundledGo,
			"build",
			"-trimpath",
			"-buildvcs=false",
			"-mod=vendor",
			"-o",
			wrapperPath,
			"./util/cmd/sliver-wasm-go",
		)
		command.Dir = repositoryRoot
		command.Env = wasmTestEnvironment(
			"GOOS="+runtime.GOOS,
			"GOARCH="+runtime.GOARCH,
			"CGO_ENABLED=0",
			"GOTOOLCHAIN=local",
			"GOWORK=off",
			"GOFLAGS=",
		)
		output, buildErr := command.CombinedOutput()
		if buildErr != nil {
			t.Fatalf("build sliver-wasm-go: %v\n%s", buildErr, output)
		}
	} else if err != nil {
		t.Fatalf("inspect sliver-wasm-go: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), filepath.Base(packagePath)+".wasm")
	command := exec.Command(wrapperPath, "build", "-trimpath", "-o", outputPath, packagePath)
	command.Dir = repositoryRoot
	command.Env = wasmTestEnvironment("GOWORK=off", "GOFLAGS=-mod=vendor")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build %s: %v\n%s", packagePath, err, output)
	}
	return outputPath
}

func wasmTestEnvironment(overrides ...string) []string {
	overrideKeys := make([]string, 0, len(overrides))
	for _, override := range overrides {
		key, _, _ := strings.Cut(override, "=")
		overrideKeys = append(overrideKeys, key)
	}
	environment := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		replaced := false
		for _, overrideKey := range overrideKeys {
			if strings.EqualFold(key, overrideKey) {
				replaced = true
				break
			}
		}
		if !replaced {
			environment = append(environment, entry)
		}
	}
	return append(environment, overrides...)
}

func runWasmTestModule(t *testing.T, wasm []byte, args []string) ([]byte, []byte, uint32, error) {
	t.Helper()
	extension, err := NewWasmExtension(t.Name(), wasm, nil)
	mustNoError(t, err)
	defer extension.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		_, _ = io.Copy(&stdout, extension.Stdout.Reader)
	}()
	go func() {
		defer readers.Done()
		_, _ = io.Copy(&stderr, extension.Stderr.Reader)
	}()

	exitCode, executeErr := extension.Execute(args)
	_ = extension.Stdout.Writer.Close()
	_ = extension.Stderr.Writer.Close()
	readers.Wait()
	return stdout.Bytes(), stderr.Bytes(), exitCode, executeErr
}

func mustNoError(t *testing.T, err error, context ...string) {
	t.Helper()
	if len(context) > 0 {
		if !assert.NoError(t, err, context[0]) {
			t.FailNow()
		}
		return
	}
	if !assert.NoError(t, err) {
		t.FailNow()
	}
}

func mustZero(t *testing.T, value uint32, context ...string) {
	t.Helper()
	if len(context) > 0 {
		if !assert.Zero(t, value, context[0]) {
			t.FailNow()
		}
		return
	}
	if !assert.Zero(t, value) {
		t.FailNow()
	}
}
