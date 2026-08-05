// sliver-wasm-go invokes Sliver's bundled Go toolchain with the standard
// library overlay required by Sliver's WASI networking host module.
package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	requiredGoVersion = "go1.26.2"
	overlayModuleName = "sliver_wasi_net_v1"
)

type overlaySource struct {
	targetPath   string
	embeddedPath string
	sha256       string
}

var requiredOverlaySources = []overlaySource{
	{
		targetPath:   "src/net/net_fake.go",
		embeddedPath: "overlay/net_fake.go.txt",
		sha256:       "784b369c57be52fa87ace5f10e07e3c24b45d549639e1487734b793024188df3",
	},
	{
		targetPath:   "src/net/lookup_unix.go",
		embeddedPath: "overlay/lookup_unix.go.txt",
		sha256:       "7fc0ecb91aa268d3dbd6dad3b6b806d92223e11782d6fc1b4dcc4f5f9a4b788e",
	},
	{
		targetPath:   "src/net/http/transport_default_wasm.go",
		embeddedPath: "overlay/transport_default_wasm.go.txt",
		sha256:       "45f994092ea1a8c432f97fc3d4673a251eabe87fd0310d2d20bbb0c3bd637993",
	},
}

//go:embed overlay/*.go.txt
var embeddedOverlays embed.FS

type overlayJSON struct {
	Replace map[string]string
}

type invocation struct {
	path   string
	args   []string
	env    []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type childRunner func(invocation) (int, error)

func main() {
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sliver-wasm-go: locate executable: %v\n", err)
		os.Exit(1)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sliver-wasm-go: resolve executable: %v\n", err)
		os.Exit(1)
	}

	code, err := run(
		os.Args[1:],
		os.Environ(),
		executable,
		os.Stdin,
		os.Stdout,
		os.Stderr,
		runChild,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sliver-wasm-go: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(
	args []string,
	environ []string,
	executable string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	runner childRunner,
) (int, error) {
	goBinary, goRoot, err := bundledToolchainPaths(executable, runtime.GOOS)
	if err != nil {
		return 0, err
	}
	if index, ok := goCommandIndex(args); ok && args[index] == "run" {
		return 0, errors.New("go run cannot execute Sliver-networked Wasm; use sliver-wasm-go build and load the module in a Sliver WASI runtime")
	}
	if err := validateToolchain(goRoot, requiredOverlaySources); err != nil {
		return 0, err
	}
	if hasOverlayFlag(args) || envHasOverlayFlag(environ) {
		return 0, errors.New("a custom -overlay is not supported; sliver-wasm-go supplies its networking overlay")
	}

	insertionIndex, inject := overlayInsertionIndex(args)
	childArgs := append([]string(nil), args...)
	var cleanup func()
	if inject {
		overlayPath, remove, err := writeOverlay(goRoot, embeddedOverlays, requiredOverlaySources)
		if err != nil {
			return 0, err
		}
		cleanup = remove
		defer cleanup()
		childArgs = injectOverlayFlag(childArgs, insertionIndex, overlayPath)
	}

	env := toolchainEnvironment(environ, goRoot, filepath.Dir(goBinary))
	return runner(invocation{
		path:   goBinary,
		args:   childArgs,
		env:    env,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	})
}

func bundledToolchainPaths(executable, hostGOOS string) (goBinary, goRoot string, err error) {
	executable, err = filepath.Abs(executable)
	if err != nil {
		return "", "", fmt.Errorf("make executable path absolute: %w", err)
	}
	binDir := filepath.Dir(executable)
	goRoot = filepath.Dir(binDir)
	goName := "go"
	if hostGOOS == "windows" {
		goName += ".exe"
	}
	goBinary = filepath.Join(binDir, goName)
	if info, statErr := os.Stat(goBinary); statErr != nil {
		return "", "", fmt.Errorf("find bundled Go at %s: %w", goBinary, statErr)
	} else if info.IsDir() {
		return "", "", fmt.Errorf("bundled Go path is a directory: %s", goBinary)
	}
	return goBinary, goRoot, nil
}

func validateToolchain(goRoot string, sources []overlaySource) error {
	for _, source := range sources {
		path := filepath.Join(goRoot, filepath.FromSlash(source.targetPath))
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read overlay target %s: %w", path, err)
		}
		sum := sha256.Sum256(data)
		actual := hex.EncodeToString(sum[:])
		if actual != source.sha256 {
			return fmt.Errorf(
				"unsupported bundled %s source %s: SHA-256 %s (need %s)",
				requiredGoVersion,
				source.targetPath,
				actual,
				source.sha256,
			)
		}
	}
	return nil
}

func writeOverlay(goRoot string, sourceFS fs.FS, sources []overlaySource) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "sliver-wasm-go-overlay-")
	if err != nil {
		return "", nil, fmt.Errorf("create overlay directory: %w", err)
	}
	remove := func() {
		_ = os.RemoveAll(tempDir)
	}

	replacements := make(map[string]string, len(sources))
	for _, source := range sources {
		data, readErr := fs.ReadFile(sourceFS, source.embeddedPath)
		if readErr != nil {
			remove()
			return "", nil, fmt.Errorf("read embedded overlay %s: %w", source.embeddedPath, readErr)
		}
		backingPath := filepath.Join(tempDir, filepath.FromSlash(source.targetPath))
		if mkdirErr := os.MkdirAll(filepath.Dir(backingPath), 0o700); mkdirErr != nil {
			remove()
			return "", nil, fmt.Errorf("create overlay source directory: %w", mkdirErr)
		}
		if writeErr := os.WriteFile(backingPath, data, 0o600); writeErr != nil {
			remove()
			return "", nil, fmt.Errorf("write overlay source: %w", writeErr)
		}
		targetPath := filepath.Join(goRoot, filepath.FromSlash(source.targetPath))
		replacements[targetPath] = backingPath
	}

	configData, err := json.MarshalIndent(overlayJSON{Replace: replacements}, "", "  ")
	if err != nil {
		remove()
		return "", nil, fmt.Errorf("encode overlay configuration: %w", err)
	}
	configPath := filepath.Join(tempDir, "overlay.json")
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		remove()
		return "", nil, fmt.Errorf("write overlay configuration: %w", err)
	}
	return configPath, remove, nil
}

func overlayInsertionIndex(args []string) (int, bool) {
	index, ok := goCommandIndex(args)
	if !ok {
		return 0, false
	}

	switch args[index] {
	case "build", "install", "list", "test", "vet":
		insertion := index + 1
		if insertion < len(args) {
			switch {
			case args[insertion] == "-C" || args[insertion] == "--C":
				if insertion+1 >= len(args) {
					return 0, false
				}
				insertion += 2
			case strings.HasPrefix(args[insertion], "-C="), strings.HasPrefix(args[insertion], "--C="):
				insertion++
			}
		}
		return insertion, true
	default:
		return index, false
	}
}

func goCommandIndex(args []string) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	index := 0
	switch {
	case args[0] == "-C" || args[0] == "--C":
		if len(args) < 3 {
			return 0, false
		}
		index = 2
	case strings.HasPrefix(args[0], "-C=") || strings.HasPrefix(args[0], "--C="):
		if len(args) < 2 {
			return 0, false
		}
		index = 1
	}
	return index, true
}

func injectOverlayFlag(args []string, insertionIndex int, overlayPath string) []string {
	result := make([]string, 0, len(args)+1)
	result = append(result, args[:insertionIndex]...)
	result = append(result, "-overlay="+overlayPath)
	result = append(result, args[insertionIndex:]...)
	return result
}

func hasOverlayFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "-overlay" || arg == "--overlay" ||
			strings.HasPrefix(arg, "-overlay=") || strings.HasPrefix(arg, "--overlay=") {
			return true
		}
	}
	return false
}

func envHasOverlayFlag(environ []string) bool {
	goFlags, ok := lookupEnvironment(environ, "GOFLAGS")
	if !ok {
		return false
	}
	for _, field := range strings.Fields(goFlags) {
		field = strings.Trim(field, `"'`)
		if field == "-overlay" || field == "--overlay" ||
			strings.HasPrefix(field, "-overlay=") || strings.HasPrefix(field, "--overlay=") {
			return true
		}
	}
	return false
}

func toolchainEnvironment(environ []string, goRoot, binDir string) []string {
	path, _ := lookupEnvironment(environ, "PATH")
	if path == "" {
		path = binDir
	} else {
		path = binDir + string(os.PathListSeparator) + path
	}
	overrides := []string{
		"GOROOT=" + goRoot,
		"GOOS=wasip1",
		"GOARCH=wasm",
		"CGO_ENABLED=0",
		"GOTOOLCHAIN=local",
		"PATH=" + path,
	}

	result := make([]string, 0, len(environ)+len(overrides))
	for _, entry := range environ {
		key := entry
		if index := strings.IndexByte(entry, '='); index >= 0 {
			key = entry[:index]
		}
		replace := false
		for _, override := range overrides {
			overrideKey := override[:strings.IndexByte(override, '=')]
			if strings.EqualFold(key, overrideKey) {
				replace = true
				break
			}
		}
		if !replace {
			result = append(result, entry)
		}
	}
	return append(result, overrides...)
}

func lookupEnvironment(environ []string, name string) (string, bool) {
	for index := len(environ) - 1; index >= 0; index-- {
		entry := environ[index]
		separator := strings.IndexByte(entry, '=')
		if separator < 0 || !strings.EqualFold(entry[:separator], name) {
			continue
		}
		return entry[separator+1:], true
	}
	return "", false
}

func runChild(inv invocation) (int, error) {
	cmd := exec.Command(inv.path, inv.args...)
	cmd.Env = inv.env
	cmd.Stdin = inv.stdin
	cmd.Stdout = inv.stdout
	cmd.Stderr = inv.stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 0, fmt.Errorf("run bundled Go: %w", err)
	}
	return 0, nil
}
