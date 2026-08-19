package reflektor

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sliverarmory/reflektor/memmod"
)

var ErrLibraryClosed = errors.New("reflektor: library is closed")

// ErrGoExportArgumentsUnsupported reports that CallExportWithArgs was used on
// a Go c-shared image. Use CallExport for its zero-argument exports instead.
var ErrGoExportArgumentsUnsupported = memmod.ErrGoExportArgumentsUnsupported

// MaxExportArguments is the maximum number of machine-word arguments accepted
// by CallExportWithArgs on every supported platform.
const MaxExportArguments = memmod.MaxExportArguments

var memoryOriginSequence atomic.Uint64

const maxRecursiveDependencyFileSize = int64(512 << 20)

type Library struct {
	mu     sync.RWMutex
	module *memmod.Module
	closed bool
}

// LoadLibrary loads a shared library image from memory.
func LoadLibrary(data []byte) (*Library, error) {
	if len(data) == 0 {
		return nil, errors.New("reflektor: empty library image")
	}

	module, err := memmod.LoadLibrary(data)
	if err != nil {
		return nil, fmt.Errorf("reflektor: load library: %w", err)
	}
	return &Library{module: module}, nil
}

// LoadLibraryRecursive loads a shared library image and recursively reads
// non-system dependencies from disk into Reflektor's in-memory loader. Relative
// dependency names are resolved from the current working directory. Use
// LoadLibraryFileRecursive when the root image has a filesystem location.
func LoadLibraryRecursive(data []byte) (*Library, error) {
	if len(data) == 0 {
		return nil, errors.New("reflektor: empty library image")
	}

	origin, err := memoryLibraryOrigin(data)
	if err != nil {
		return nil, err
	}
	module, err := memmod.LoadLibraryRecursive(data, origin, dependencyFileReader(origin))
	if err != nil {
		return nil, fmt.Errorf("reflektor: recursively load library: %w", err)
	}
	return &Library{module: module}, nil
}

// LoadLibraryFile loads a shared library image from disk into memory.
func LoadLibraryFile(path string) (*Library, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reflektor: read library file: %w", err)
	}
	return LoadLibrary(data)
}

// LoadLibraryFileRecursive reads a root shared library and all of its
// non-system, file-backed dependencies into Reflektor's in-memory loader.
func LoadLibraryFileRecursive(path string) (*Library, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("reflektor: resolve library file path: %w", err)
	}
	root, err := readRegularDependency(absPath)
	if err != nil {
		return nil, fmt.Errorf("reflektor: read library file: %w", err)
	}
	module, err := memmod.LoadLibraryRecursive(root.Data, root.Path, readDependencyFile)
	if err != nil {
		return nil, fmt.Errorf("reflektor: recursively load library file %q: %w", absPath, err)
	}
	return &Library{module: module}, nil
}

func dependencyFileReader(memoryOrigin string) memmod.DependencyReader {
	return func(request memmod.DependencyRequest) (memmod.Dependency, error) {
		// A byte-only root has no real loader directory. Treat the working
		// directory captured in its synthetic origin as the root search location,
		// without adding a process-CWD fallback for file-backed roots or children.
		if request.ImporterPath == memoryOrigin {
			request.SearchPaths = append(request.SearchPaths, filepath.Dir(memoryOrigin))
		}
		return readDependencyFile(request)
	}
}

func memoryLibraryOrigin(data []byte) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("reflektor: get working directory for recursive load: %w", err)
	}
	ext := map[string]string{
		"darwin":  ".dylib",
		"linux":   ".so",
		"windows": ".dll",
	}[runtime.GOOS]
	digest := sha256.Sum256(data)
	sequence := memoryOriginSequence.Add(1)
	name := fmt.Sprintf("reflektor-memory-root-%x-%d%s", digest[:6], sequence, ext)
	return filepath.Join(cwd, name), nil
}

func readDependencyFile(request memmod.DependencyRequest) (memmod.Dependency, error) {
	candidates := dependencyCandidates(request)
	for _, candidate := range candidates {
		dependency, err := readRegularDependency(candidate)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return memmod.Dependency{}, fmt.Errorf("read dependency %q imported by %q from %q: %w", request.Name, request.ImporterPath, candidate, err)
			}
			continue
		}
		return dependency, nil
	}

	return memmod.Dependency{}, fmt.Errorf("%w: %q imported by %q", memmod.ErrDependencyNotFound, request.Name, request.ImporterPath)
}

func readRegularDependency(candidate string) (memmod.Dependency, error) {
	resolved, err := filepath.Abs(candidate)
	if err != nil {
		return memmod.Dependency{}, fmt.Errorf("resolve dependency path %q: %w", candidate, err)
	}
	resolved = filepath.Clean(resolved)
	if isUNCDependencyPath(resolved) {
		return memmod.Dependency{}, fmt.Errorf("refuse UNC dependency path %q", resolved)
	}
	if evaluated, err := filepath.EvalSymlinks(resolved); err == nil {
		resolved = evaluated
	}
	if isUNCDependencyPath(resolved) {
		return memmod.Dependency{}, fmt.Errorf("refuse UNC dependency path %q", resolved)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return memmod.Dependency{}, err
	}
	if err := validateDependencyFileInfo(resolved, info); err != nil {
		return memmod.Dependency{}, err
	}

	file, err := os.Open(resolved)
	if err != nil {
		return memmod.Dependency{}, err
	}
	defer file.Close()
	info, err = file.Stat()
	if err != nil {
		return memmod.Dependency{}, fmt.Errorf("stat dependency %q: %w", resolved, err)
	}
	if err := validateDependencyFileInfo(resolved, info); err != nil {
		return memmod.Dependency{}, err
	}

	data, err := io.ReadAll(io.LimitReader(file, maxRecursiveDependencyFileSize+1))
	if err != nil {
		return memmod.Dependency{}, fmt.Errorf("read dependency %q: %w", resolved, err)
	}
	if int64(len(data)) > maxRecursiveDependencyFileSize {
		return memmod.Dependency{}, fmt.Errorf("dependency %q grew beyond %d bytes while reading", resolved, maxRecursiveDependencyFileSize)
	}
	return memmod.Dependency{Data: data, Path: resolved}, nil
}

func isUNCDependencyPath(path string) bool {
	return runtime.GOOS == "windows" && strings.HasPrefix(path, `\\`)
}

func validateDependencyFileInfo(path string, info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("dependency %q is not a regular file", path)
	}
	if info.Size() < 0 || info.Size() > maxRecursiveDependencyFileSize {
		return fmt.Errorf("dependency %q size %d exceeds %d bytes", path, info.Size(), maxRecursiveDependencyFileSize)
	}
	return nil
}

func dependencyCandidates(request memmod.DependencyRequest) []string {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return nil
	}

	importerDir := filepath.Dir(request.ImporterPath)
	executableDir := ""
	if executable, err := os.Executable(); err == nil {
		executableDir = filepath.Dir(executable)
	}
	expand := func(value string) string {
		value = strings.ReplaceAll(value, "${ORIGIN}", importerDir)
		value = strings.ReplaceAll(value, "$ORIGIN", importerDir)
		value = strings.ReplaceAll(value, "@loader_path", importerDir)
		if executableDir != "" {
			value = strings.ReplaceAll(value, "@executable_path", executableDir)
		}
		return value
	}

	paths := make([]string, 0, 24)
	seen := make(map[string]struct{}, 24)
	add := func(candidate string) {
		candidate = filepath.Clean(strings.TrimSpace(expand(candidate)))
		if candidate == "" || candidate == "." {
			return
		}
		key := candidate
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		paths = append(paths, candidate)
	}

	expandedName := expand(name)
	if filepath.IsAbs(expandedName) {
		add(expandedName)
		return paths
	} else if strings.HasPrefix(name, "@rpath/") {
		rel := strings.TrimPrefix(name, "@rpath/")
		for _, searchPath := range request.SearchPaths {
			add(filepath.Join(expand(searchPath), rel))
		}
		return paths
	} else {
		for _, searchPath := range request.SearchPaths {
			add(filepath.Join(expand(searchPath), expandedName))
		}
	}

	base := filepath.Base(expandedName)
	if base != expandedName {
		return paths
	}
	for _, dir := range dependencyEnvironmentDirs() {
		add(filepath.Join(dir, base))
	}
	for _, dir := range dependencySystemDirs() {
		add(filepath.Join(dir, base))
	}
	return paths
}

func dependencyEnvironmentDirs() []string {
	var keys []string
	switch runtime.GOOS {
	case "darwin":
		keys = []string{"DYLD_LIBRARY_PATH", "DYLD_FALLBACK_LIBRARY_PATH"}
	case "linux":
		keys = []string{"LD_LIBRARY_PATH"}
	case "windows":
		// Recursive Windows resolution receives explicit importer/root search
		// paths. System32 fallback is owned by the PE backend, not process PATH.
	}

	var dirs []string
	for _, key := range keys {
		for _, dir := range filepath.SplitList(os.Getenv(key)) {
			if dir != "" {
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs
}

func dependencySystemDirs() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/usr/local/lib", "/opt/homebrew/lib", "/usr/lib", "/System/Library/Frameworks"}
	case "linux":
		dirs := []string{"/usr/local/lib", "/lib", "/lib64", "/usr/lib", "/usr/lib64"}
		switch runtime.GOARCH {
		case "amd64":
			dirs = append(dirs, "/lib/x86_64-linux-gnu", "/usr/lib/x86_64-linux-gnu")
		case "386":
			dirs = append(dirs, "/lib/i386-linux-gnu", "/usr/lib/i386-linux-gnu")
		case "arm64":
			dirs = append(dirs, "/lib/aarch64-linux-gnu", "/usr/lib/aarch64-linux-gnu")
		}
		return dirs
	}
	return nil
}

// CallExport resolves and calls a zero-argument exported function.
func (library *Library) CallExport(name string) error {
	library.mu.RLock()
	defer library.mu.RUnlock()

	if library.closed || library.module == nil {
		return ErrLibraryClosed
	}
	if err := library.module.CallExport(name); err != nil {
		return fmt.Errorf("reflektor: call export %q: %w", name, err)
	}
	return nil
}

// CallExportWithArgs resolves an export, calls it with up to three machine-word
// arguments, and returns the value from the platform's primary return register.
// Go c-shared images must continue to use CallExport because their runtime
// exports require isolated zero-argument invocation.
//
//go:uintptrescapes
func (library *Library) CallExportWithArgs(name string, args ...uintptr) (uintptr, error) {
	library.mu.RLock()
	defer library.mu.RUnlock()

	if library.closed || library.module == nil {
		return 0, ErrLibraryClosed
	}
	result, err := library.module.CallExportWithArgs(name, args...)
	if err != nil {
		return 0, fmt.Errorf("reflektor: call export %q with arguments: %w", name, err)
	}
	return result, nil
}

// Close releases library resources. Go c-shared images remain mapped until
// process exit because a started Go runtime cannot be unloaded safely.
func (library *Library) Close() error {
	library.mu.Lock()
	defer library.mu.Unlock()

	if library.closed {
		return nil
	}
	library.closed = true

	if library.module != nil {
		library.module.Free()
		library.module = nil
	}
	return nil
}
