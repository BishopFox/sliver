// Package buildcompiler selects an installed compiler driver for a resolved
// core step. Compiler drivers receive build inputs, never Sliver source or
// implant configuration objects from the generic planner.
package buildcompiler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcore/magebuild"
	"github.com/bishopfox/sliver/server/gogo"
)

// GoDriverID identifies the installed Go compiler driver.
const GoDriverID = "go"

// Request describes a compiler invocation. GoConfig is the typed configuration
// for the initial Go driver; other compiler drivers can be added to the
// registry without changing the planner or Sliver source rendering.
type Request struct {
	GoConfig     gogo.GoConfig
	SourceDir    string
	Destination  string
	BuildMode    string
	Tags         []string
	LDFlags      []string
	GCFlags      string
	ASMFlags     string
	MaterialPath string
	MagefilesDir string
	MageDigest   string
	MageTarget   string
	ExtraEnv     []string
}

// Driver executes one compiler invocation. Implementations are registered by
// a stable ID declared in the selected core's toolchain capability.
type Driver interface {
	ID() string
	Build(ctx context.Context, request Request) ([]byte, error)
}

// Registry holds installed compiler integrations. Registration and lookup are
// safe to use concurrently during server startup and build execution.
type Registry struct {
	mu      sync.RWMutex
	drivers map[string]Driver
}

// NewRegistry creates an empty compiler driver registry.
func NewRegistry() *Registry {
	return &Registry{drivers: make(map[string]Driver)}
}

// Register installs a compiler driver under its unique ID.
func (registry *Registry) Register(driver Driver) error {
	if registry == nil {
		return errors.New("nil compiler registry")
	}
	if driver == nil || driver.ID() == "" {
		return errors.New("compiler driver needs a nonempty ID")
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.drivers[driver.ID()]; exists {
		return fmt.Errorf("compiler driver %q is already installed", driver.ID())
	}
	registry.drivers[driver.ID()] = driver
	return nil
}

// Build delegates a request to the installed driver with the given ID.
func (registry *Registry) Build(ctx context.Context, id string, request Request) ([]byte, error) {
	if registry == nil {
		return nil, errors.New("nil compiler registry")
	}
	if ctx == nil {
		return nil, errors.New("nil compiler context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	registry.mu.RLock()
	driver := registry.drivers[id]
	registry.mu.RUnlock()
	if driver == nil {
		return nil, fmt.Errorf("compiler driver %q is not installed", id)
	}
	request.Tags = slices.Clone(request.Tags)
	request.LDFlags = slices.Clone(request.LDFlags)
	request.ExtraEnv = slices.Clone(request.ExtraEnv)
	return driver.Build(ctx, request)
}

var defaultRegistry = func() *Registry {
	registry := NewRegistry()
	_ = registry.Register(GoDriver{})
	return registry
}()

// Default returns the process's installed compiler registry. Additional
// drivers may be registered at startup before their specs are advertised.
func Default() *Registry { return defaultRegistry }

// GoDriver routes Go, Zig, and Garble through the selected core Magefile.
// A core can supply its own installed Magefile bundle; Sliver uses the built-in
// bundle when no override is selected.
type GoDriver struct{}

// ID returns the stable identifier for the Go driver.
func (GoDriver) ID() string { return GoDriverID }

// Build runs the installed Go compiler target and verifies its result manifest.
func (GoDriver) Build(ctx context.Context, request Request) ([]byte, error) {
	if request.SourceDir == "" || request.Destination == "" {
		return nil, errors.New("Go build needs source and destination paths") //nolint:staticcheck // Go is the toolchain's proper name (ST1005).
	}
	project := request.GoConfig.ProjectDir
	if project == "" || !filepath.IsAbs(project) || !filepath.IsAbs(request.SourceDir) || !filepath.IsAbs(request.Destination) {
		return nil, errors.New("Mage Go build requires absolute workspace paths") //nolint:staticcheck // Mage is the toolchain's proper name (ST1005).
	}
	binDir := filepath.Join(project, "bin")
	if filepath.Dir(request.Destination) != binDir {
		return nil, errors.New("Mage Go output must be in the workspace bin directory") //nolint:staticcheck // Mage is the toolchain's proper name (ST1005).
	}
	inputsDir := filepath.Join(project, "inputs")
	if err := os.MkdirAll(inputsDir, 0700); err != nil {
		return nil, fmt.Errorf("create Mage build inputs: %w", err)
	}
	if err := os.MkdirAll(binDir, 0700); err != nil {
		return nil, fmt.Errorf("create Mage build output: %w", err)
	}
	bundleDir, digest, target, err := selectMageBundle(request)
	if err != nil {
		return nil, err
	}
	resultPath := filepath.Join(binDir, "result.json")
	manifest := magebuild.GoManifest{
		Version: 1,
		GoConfig: magebuild.GoConfig{
			ProjectDir: project, GOOS: request.GoConfig.GOOS, GOARCH: request.GoConfig.GOARCH,
			GOROOT: request.GoConfig.GOROOT, GOCACHE: request.GoConfig.GOCACHE,
			GOMODCACHE: request.GoConfig.GOMODCACHE, GOPROXY: request.GoConfig.GOPROXY,
			CGO: request.GoConfig.CGO, CC: request.GoConfig.CC, CXX: request.GoConfig.CXX,
			HTTPPROXY: request.GoConfig.HTTPPROXY, HTTPSPROXY: request.GoConfig.HTTPSPROXY,
			Obfuscation: request.GoConfig.Obfuscation, GOGARBLE: request.GoConfig.GOGARBLE,
		},
		MaterialPath: request.MaterialPath, SourceDir: request.SourceDir,
		Destination: request.Destination, BuildMode: request.BuildMode,
		Tags: request.Tags, LDFlags: request.LDFlags, GCFlags: request.GCFlags,
		ASMFlags: request.ASMFlags, ResultPath: resultPath,
	}
	manifestPath, err := writeGoManifest(inputsDir, manifest)
	if err != nil {
		return nil, err
	}
	hostGo, err := exec.LookPath("go")
	if err != nil {
		hostGo = filepath.Join(request.GoConfig.GOROOT, "bin", "go")
	}
	var stdout, stderr bytes.Buffer
	runner := magebuild.Runner{RootDir: assets.GetRootAppDir(), GoCmd: hostGo}
	err = runner.Run(ctx, magebuild.Invocation{
		BundleDir: bundleDir, BundleDigest: digest, WorkDir: project,
		ManifestPath: manifestPath, Target: target, Env: mageRuntimeEnv(request.ExtraEnv),
		Stdout: &stdout, Stderr: &stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("run core Mage target: %w", err)
	}
	if err := validateGoBuildResult(resultPath, filepath.Base(request.Destination)); err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func writeGoManifest(dir string, manifest magebuild.GoManifest) (string, error) {
	file, err := os.CreateTemp(dir, "go-build-*.json")
	if err != nil {
		return "", fmt.Errorf("create Mage build manifest: %w", err)
	}
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := json.NewEncoder(file).Encode(manifest); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("write Mage build manifest: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func validateGoBuildResult(resultPath, expectedName string) error {
	resultInfo, err := os.Lstat(resultPath)
	if err != nil || !resultInfo.Mode().IsRegular() || resultInfo.Size() > 4096 {
		return errors.New("core Mage result is missing, nonregular, or too large")
	}
	resultFile, err := os.Open(resultPath)
	if err != nil {
		return fmt.Errorf("read core Mage result: %w", err)
	}
	defer func() { _ = resultFile.Close() }()
	var result magebuild.Result
	decoder := json.NewDecoder(io.LimitReader(resultFile, 4097))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil || result.Primary != expectedName ||
		result.DisplayName != result.Primary {
		return errors.New("core Mage result does not match planned output")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("core Mage result has trailing data or exceeds size limit")
	}
	return nil
}

func mageRuntimeEnv(extra []string) []string {
	allowed := map[string]bool{
		"PATH": true, "HOME": true, "TMPDIR": true, "TMP": true, "TEMP": true,
		"USERPROFILE": true, "SYSTEMROOT": true, "WINDIR": true, "COMSPEC": true,
		"PATHEXT": true, "LOCALAPPDATA": true, "APPDATA": true,
		"HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true,
		"http_proxy": true, "https_proxy": true, "no_proxy": true,
		"SSL_CERT_FILE": true, "SSL_CERT_DIR": true,
		"SDKROOT": true, "DEVELOPER_DIR": true, "MACOSX_DEPLOYMENT_TARGET": true,
	}
	values := make([]string, 0, len(allowed)+len(extra))
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if ok && (allowed[name] || strings.HasPrefix(name, "CGO_") || strings.HasPrefix(name, "ZIG_")) {
			values = append(values, entry)
		}
	}
	return append(values, extra...)
}

func selectMageBundle(request Request) (bundleDir, digest, target string, err error) {
	if request.MagefilesDir != "" {
		if request.MageDigest == "" || request.MageTarget == "" {
			return "", "", "", errors.New("custom Magefile requires a pinned digest and target")
		}
		return request.MagefilesDir, request.MageDigest, request.MageTarget, nil
	}
	bundleDir = filepath.Join(request.GoConfig.ProjectDir, "magefiles")
	want, err := magebuild.BuiltinSliverDigest()
	if err != nil {
		return "", "", "", err
	}
	entries, err := os.ReadDir(bundleDir)
	if errors.Is(err, os.ErrNotExist) {
		if _, err := magebuild.StageBuiltinSliverBundle(bundleDir); err != nil {
			return "", "", "", fmt.Errorf("stage Sliver Magefile: %w", err)
		}
	} else if err != nil {
		return "", "", "", err
	} else if len(entries) == 0 {
		if _, err := magebuild.StageBuiltinSliverBundle(bundleDir); err != nil {
			return "", "", "", fmt.Errorf("stage Sliver Magefile: %w", err)
		}
	}
	actual, err := magebuild.DigestBundle(bundleDir)
	if err != nil || actual != want {
		return "", "", "", errors.New("Sliver Magefile bundle does not match installed version") //nolint:staticcheck // Sliver is the bundle's proper name (ST1005).
	}
	return bundleDir, want, "Build", nil
}
