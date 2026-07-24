package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/tetratelabs/wazero"
)

func TestOverlayArgumentPlanning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "build",
			args: []string{"build", "./..."},
			want: []string{"build", "-overlay=/tmp/overlay.json", "./..."},
		},
		{
			name: "global C",
			args: []string{"-C", "module", "build", "./..."},
			want: []string{"-C", "module", "build", "-overlay=/tmp/overlay.json", "./..."},
		},
		{
			name: "global C equals",
			args: []string{"--C=module", "install", "./cmd/tool"},
			want: []string{"--C=module", "install", "-overlay=/tmp/overlay.json", "./cmd/tool"},
		},
		{
			name: "command C",
			args: []string{"test", "-C", "module", "-c", "./..."},
			want: []string{"test", "-C", "module", "-overlay=/tmp/overlay.json", "-c", "./..."},
		},
		{
			name: "run requires Sliver runtime",
			args: []string{"run", "-C=module", "."},
			want: []string{"run", "-C=module", "."},
		},
		{
			name: "list",
			args: []string{"list", "std"},
			want: []string{"list", "-overlay=/tmp/overlay.json", "std"},
		},
		{
			name: "vet",
			args: []string{"vet", "./..."},
			want: []string{"vet", "-overlay=/tmp/overlay.json", "./..."},
		},
		{
			name: "pass through version",
			args: []string{"version"},
			want: []string{"version"},
		},
		{
			name: "pass through no arguments",
			args: nil,
			want: nil,
		},
		{
			name: "pass malformed C to Go",
			args: []string{"-C"},
			want: []string{"-C"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			insertion, inject := overlayInsertionIndex(test.args)
			got := append([]string(nil), test.args...)
			if inject {
				got = injectOverlayFlag(got, insertion, "/tmp/overlay.json")
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("planned arguments = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestOverlayFlagDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args []string
		want bool
	}{
		{args: []string{"build", "-overlay", "custom.json"}, want: true},
		{args: []string{"build", "--overlay=custom.json"}, want: true},
		{args: []string{"run", ".", "--", "-overlay=program-argument"}, want: false},
		{args: []string{"build", "./..."}, want: false},
	}
	for _, test := range tests {
		if got := hasOverlayFlag(test.args); got != test.want {
			t.Errorf("hasOverlayFlag(%q) = %v, want %v", test.args, got, test.want)
		}
	}

	if !envHasOverlayFlag([]string{`GOFLAGS="-trimpath -overlay=/tmp/custom overlay.json"`}) {
		t.Fatal("quoted GOFLAGS overlay was not detected")
	}
	if envHasOverlayFlag([]string{`GOFLAGS="-trimpath -tags=overlay"`}) {
		t.Fatal("unrelated GOFLAGS value was detected as an overlay")
	}
}

func TestValidateToolchain(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	content := []byte("original Go source\n")
	source := overlaySource{
		targetPath:   "src/net/net_fake.go",
		embeddedPath: "overlay/net_fake.go",
		sha256:       sha256String(content),
	}
	writeTestFile(t, filepath.Join(root, filepath.FromSlash(source.targetPath)), content, 0o600)

	if err := validateToolchain(root, []overlaySource{source}); err != nil {
		t.Fatalf("validateToolchain() error = %v", err)
	}

	writeTestFile(t, filepath.Join(root, filepath.FromSlash(source.targetPath)), []byte("changed\n"), 0o600)
	err := validateToolchain(root, []overlaySource{source})
	if err == nil {
		t.Fatal("validateToolchain() succeeded with modified source")
	}
	for _, fragment := range []string{requiredGoVersion, source.targetPath, "SHA-256", source.sha256} {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("validation error %q does not contain %q", err, fragment)
		}
	}

	err = validateToolchain(root, []overlaySource{{
		targetPath: "src/net/missing.go",
		sha256:     source.sha256,
	}})
	if err == nil || !strings.Contains(err.Error(), "missing.go") {
		t.Fatalf("missing source error = %v", err)
	}
}

func TestWriteOverlayUsesAbsolutePaths(t *testing.T) {
	t.Parallel()

	goRoot := t.TempDir()
	sourceFS := fstest.MapFS{
		"overlay/net_fake.go": &fstest.MapFile{Data: []byte("package net\n")},
	}
	sources := []overlaySource{{
		targetPath:   "src/net/net_fake.go",
		embeddedPath: "overlay/net_fake.go",
	}}
	configPath, cleanup, err := writeOverlay(goRoot, sourceFS, sources)
	if err != nil {
		t.Fatalf("writeOverlay() error = %v", err)
	}
	defer cleanup()

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config overlayJSON
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Replace) != 1 {
		t.Fatalf("replacement count = %d, want 1", len(config.Replace))
	}
	for target, backing := range config.Replace {
		if !filepath.IsAbs(target) || !filepath.IsAbs(backing) {
			t.Fatalf("overlay paths are not absolute: %q -> %q", target, backing)
		}
		data, err := os.ReadFile(backing)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "package net\n" {
			t.Fatalf("backing content = %q", data)
		}
	}
}

func TestToolchainEnvironment(t *testing.T) {
	t.Parallel()

	got := toolchainEnvironment(
		[]string{
			"GOROOT=/old/root",
			"goos=linux",
			"GOARCH=amd64",
			"CGO_ENABLED=1",
			"GOTOOLCHAIN=auto",
			"PATH=/usr/bin",
			"GOPROXY=https://proxy.example",
		},
		"/sliver/go",
		"/sliver/go/bin",
	)
	expected := map[string]string{
		"GOROOT":      "/sliver/go",
		"GOOS":        "wasip1",
		"GOARCH":      "wasm",
		"CGO_ENABLED": "0",
		"GOTOOLCHAIN": "local",
		"GOPROXY":     "https://proxy.example",
	}
	for name, want := range expected {
		value, ok := lookupEnvironment(got, name)
		if !ok || value != want {
			t.Errorf("%s = %q, %v; want %q", name, value, ok, want)
		}
	}
	path, _ := lookupEnvironment(got, "PATH")
	wantPath := "/sliver/go/bin" + string(os.PathListSeparator) + "/usr/bin"
	if path != wantPath {
		t.Errorf("PATH = %q, want %q", path, wantPath)
	}
}

func TestRunInjectsOverlayAndPreservesExitCode(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	executable := filepath.Join(binDir, "sliver-wasm-go")
	goName := "go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
	}
	writeTestFile(t, filepath.Join(binDir, goName), []byte("fake Go"), 0o700)
	writeTestFile(t, executable, []byte("fake wrapper"), 0o700)

	originalSources := requiredOverlaySources
	t.Cleanup(func() {
		requiredOverlaySources = originalSources
	})
	requiredOverlaySources = []overlaySource{
		testOverlaySource(t, root, "src/net/net_fake.go", "overlay/net_fake.go"),
		testOverlaySource(t, root, "src/net/lookup_unix.go", "overlay/lookup_unix.go"),
		testOverlaySource(t, root, "src/net/http/transport_default_wasm.go", "overlay/transport_default_wasm.go"),
	}

	var captured invocation
	var configPath string
	runner := func(inv invocation) (int, error) {
		captured = inv
		for _, arg := range inv.args {
			if strings.HasPrefix(arg, "-overlay=") {
				configPath = strings.TrimPrefix(arg, "-overlay=")
			}
		}
		if configPath == "" {
			t.Fatal("runner did not receive an overlay")
		}
		configData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read live overlay: %v", err)
		}
		var config overlayJSON
		if err := json.Unmarshal(configData, &config); err != nil {
			t.Fatal(err)
		}
		if len(config.Replace) != len(requiredOverlaySources) {
			t.Fatalf("replacement count = %d, want %d", len(config.Replace), len(requiredOverlaySources))
		}
		return 23, nil
	}

	code, err := run(
		[]string{"build", "-o", "module.wasm", "."},
		[]string{"PATH=/usr/bin"},
		executable,
		strings.NewReader("stdin"),
		io.Discard,
		io.Discard,
		runner,
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if code != 23 {
		t.Fatalf("exit code = %d, want 23", code)
	}
	if captured.path != filepath.Join(binDir, goName) {
		t.Errorf("child path = %q", captured.path)
	}
	if got, _ := lookupEnvironment(captured.env, "GOOS"); got != "wasip1" {
		t.Errorf("GOOS = %q", got)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Errorf("temporary overlay still exists after run: %v", err)
	}
}

func TestRunRejectsGoRun(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	executable := filepath.Join(binDir, "sliver-wasm-go")
	goName := "go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
	}
	writeTestFile(t, filepath.Join(binDir, goName), []byte("fake Go"), 0o700)
	writeTestFile(t, executable, []byte("fake wrapper"), 0o700)

	code, err := run(
		[]string{"run", "."},
		nil,
		executable,
		strings.NewReader(""),
		io.Discard,
		io.Discard,
		func(invocation) (int, error) {
			t.Fatal("go run reached the bundled compiler")
			return 0, nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "cannot execute Sliver-networked Wasm") {
		t.Fatalf("run error = %v", err)
	}
	if code != 0 {
		t.Fatalf("run exit code = %d, want 0 before main converts the error to exit 1", code)
	}
}

func TestEmbeddedOverlayABI(t *testing.T) {
	t.Parallel()

	data, err := fs.ReadFile(embeddedOverlays, "overlay/net_fake.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, name := range []string{
		"dial_start",
		"listen",
		"accept_start",
		"read_start",
		"write_start",
		"recv_from_start",
		"send_to_start",
		"op_poll",
		"op_cancel",
		"shutdown",
		"close",
		"get_addr",
		"set_deadline",
		"lookup_start",
	} {
		directive := "//go:wasmimport " + overlayModuleName + " " + name
		if !strings.Contains(source, directive) {
			t.Errorf("embedded overlay is missing %q", directive)
		}
	}
}

func TestEmbeddedOverlayCompilesWithCompatibleGo(t *testing.T) {
	goRoot := runtime.GOROOT()
	if err := validateToolchain(goRoot, requiredOverlaySources); err != nil {
		t.Skipf("host Go does not have the compatible %s standard library: %v", requiredGoVersion, err)
	}

	configPath, cleanup, err := writeOverlay(goRoot, embeddedOverlays, requiredOverlaySources)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	goName := "go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
	}
	outputPath := filepath.Join(t.TempDir(), "http-compile.wasm")
	cmd := exec.Command(
		filepath.Join(goRoot, "bin", goName),
		"build",
		"-trimpath",
		"-overlay="+configPath,
		"-o",
		outputPath,
		"./testdata/compile",
	)
	cmd.Env = append(
		toolchainEnvironment(os.Environ(), goRoot, filepath.Join(goRoot, "bin")),
		"GOWORK=off",
		"GOFLAGS=-mod=vendor",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compile test module: %v\n%s", err, output)
	}

	wasm, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(wasm) < 4 || string(wasm[:4]) != "\x00asm" {
		t.Fatalf("compiler output is not a Wasm module: %x", wasm[:min(len(wasm), 4)])
	}

	ctx := context.Background()
	wasmRuntime := wazero.NewRuntime(ctx)
	defer wasmRuntime.Close(ctx)
	compiled, err := wasmRuntime.CompileModule(ctx, wasm)
	if err != nil {
		t.Fatalf("wazero rejected compiler output: %v", err)
	}
	defer compiled.Close(ctx)

	imports := map[string]bool{}
	for _, definition := range compiled.ImportedFunctions() {
		module, name, imported := definition.Import()
		if imported && module == overlayModuleName {
			imports[name] = true
		}
	}
	for _, name := range []string{
		"dial_start",
		"read_start",
		"write_start",
		"op_poll",
		"op_cancel",
		"close",
		"get_addr",
		"set_deadline",
		"lookup_start",
	} {
		if !imports[name] {
			t.Errorf("compiled module does not import %s.%s", overlayModuleName, name)
		}
	}
}

func testOverlaySource(t *testing.T, root, targetPath, embeddedPath string) overlaySource {
	t.Helper()
	content := []byte("original " + targetPath + "\n")
	writeTestFile(t, filepath.Join(root, filepath.FromSlash(targetPath)), content, 0o600)
	return overlaySource{
		targetPath:   targetPath,
		embeddedPath: embeddedPath,
		sha256:       sha256String(content),
	}
}

func writeTestFile(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
}

func sha256String(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
