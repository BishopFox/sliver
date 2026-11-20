// Package magebuild runs installed Magefile bundles for core builds.
package magebuild

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/mage"
)

const (
	// Version is the pinned Mage library version used to compile core runners.
	Version     = "v1.17.2"
	maxLogBytes = 1 << 20
)

var compileMu sync.Mutex

// Runner compiles a trusted Magefile bundle once per host and runs its target
// in an isolated per-build directory. RootDir is a private cache directory
// below the Sliver app directory. GoCmd may select an installed Go compiler.
type Runner struct {
	RootDir string
	GoCmd   string
}

// Invocation selects an installed bundle and supplies one build workspace.
// BundleDigest is the DigestBundle result pinned by the installed core spec.
// Env is the complete environment for the target; nil means an empty one.
type Invocation struct {
	BundleDir    string
	BundleDigest string
	WorkDir      string
	ManifestPath string
	Target       string
	Env          []string
	Stdout       io.Writer
	Stderr       io.Writer
}

// Run verifies the installed Magefile bundle, compiles it if needed, and runs
// its target. The build material is read by the target from ManifestPath; no
// material contents are passed through arguments, environment, or errors.
//
//nolint:gocyclo // Bundle validation, manifest confinement, and process execution are reviewed together.
func (runner Runner) Run(ctx context.Context, inv Invocation) error {
	if ctx == nil {
		return errors.New("missing build context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if runner.RootDir == "" || inv.BundleDir == "" || inv.BundleDigest == "" || inv.WorkDir == "" || inv.ManifestPath == "" || inv.Target == "" {
		return errors.New("incomplete Mage build invocation")
	}
	if strings.ContainsAny(inv.Target, "/\\\x00\r\n") {
		return errors.New("invalid Mage target name")
	}
	if err := validateDigest(inv.BundleDigest); err != nil {
		return err
	}
	workDir, err := filepath.Abs(inv.WorkDir)
	if err != nil {
		return fmt.Errorf("resolve Mage work directory: %w", err)
	}
	manifestPath, err := filepath.Abs(inv.ManifestPath)
	if err != nil {
		return fmt.Errorf("resolve Mage manifest path: %w", err)
	}
	if !within(workDir, manifestPath) {
		return errors.New("the Mage manifest is outside build workspace")
	}
	if info, err := os.Lstat(manifestPath); err != nil || !info.Mode().IsRegular() {
		return errors.New("the Mage manifest must be a regular file")
	}
	if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
		return errors.New("the Mage workspace does not exist")
	}
	resolvedWorkDir, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		return fmt.Errorf("resolve Mage workspace: %w", err)
	}
	resolvedManifest, err := filepath.EvalSymlinks(manifestPath)
	if err != nil || !within(resolvedWorkDir, resolvedManifest) {
		return errors.New("the Mage manifest leaves build workspace")
	}

	actualDigest, err := DigestBundle(inv.BundleDir)
	if err != nil {
		return fmt.Errorf("inspect Magefile bundle: %w", err)
	}
	if actualDigest != inv.BundleDigest {
		return errors.New("the Magefile bundle digest does not match installed core")
	}
	compiled, err := runner.compiledTarget(ctx, inv.BundleDir, inv.BundleDigest, inv.Stdout, inv.Stderr)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, compiled, inv.Target, manifestPath)
	cmd.Dir = workDir
	cmd.Env = append([]string{}, inv.Env...)
	cmd.Stdout = newBoundedWriter(inv.Stdout)
	cmd.Stderr = newBoundedWriter(inv.Stderr)
	configureProcess(cmd)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("run Mage target %q: %w", inv.Target, err)
	}
	return nil
}

//nolint:gocyclo // Lookup, staging, verification, and publication use a single cache lock.
func (runner Runner) compiledTarget(ctx context.Context, bundleDir, digest string, stdout, stderr io.Writer) (string, error) {
	compileMu.Lock()
	defer compileMu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	root, err := filepath.Abs(runner.RootDir)
	if err != nil {
		return "", fmt.Errorf("resolve Mage cache: %w", err)
	}
	goVersion := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(runtime.Version())
	cacheDir := filepath.Join(root, "mage-runners", Version, goVersion, runtime.GOOS+"-"+runtime.GOARCH, strings.TrimPrefix(digest, "sha256:"))
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", fmt.Errorf("create Mage cache: %w", err)
	}
	path := filepath.Join(cacheDir, executableName("core-mage"))
	if info, err := os.Lstat(path); err == nil {
		if info.Mode().IsRegular() && info.Size() > 0 {
			return path, nil
		}
		return "", errors.New("the Mage cache contains invalid compiled target")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect Mage cache: %w", err)
	}

	// Mage generates a mainfile in its source directory. Keep that write away
	// from the installed bundle and from other concurrent build workspaces.
	stageDir, err := os.MkdirTemp(cacheDir, "compile-*")
	if err != nil {
		return "", fmt.Errorf("stage Magefile bundle: %w", err)
	}
	defer func() { _ = os.RemoveAll(stageDir) }()
	stageBundle := filepath.Join(stageDir, "magefiles")
	if err := copyBundle(bundleDir, stageBundle); err != nil {
		return "", fmt.Errorf("copy Magefile bundle: %w", err)
	}
	stageDigest, err := DigestBundle(stageBundle)
	if err != nil || stageDigest != digest {
		return "", errors.New("the Magefile bundle changed during staging")
	}
	stagedBinary := filepath.Join(stageDir, executableName("core-mage"))
	goCmd := runner.GoCmd
	if goCmd == "" {
		goCmd = "go"
	}
	code := mage.Invoke(mage.Invocation{
		Dir: stageBundle, WorkDir: stageBundle, CompileOut: stagedBinary,
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		GoCmd: goCmd, CacheDir: filepath.Join(stageDir, "mage-cache"),
		Stdout: newBoundedWriter(stdout), Stderr: newBoundedWriter(stderr),
	})
	if code != 0 {
		return "", fmt.Errorf("compile Magefile bundle: Mage exited with status %d", code)
	}
	info, err := os.Lstat(stagedBinary)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return "", errors.New("the Mage invocation did not produce a compiled target")
	}
	if err := os.Chmod(stagedBinary, 0700); err != nil {
		return "", fmt.Errorf("set Mage target permissions: %w", err)
	}
	if err := os.Rename(stagedBinary, path); err != nil {
		if info, statErr := os.Lstat(path); statErr == nil && info.Mode().IsRegular() && info.Size() > 0 {
			return path, nil
		}
		return "", fmt.Errorf("publish compiled Mage target: %w", err)
	}
	return path, nil
}

// DigestBundle hashes every regular file under an installed Magefile bundle,
// including relative path, owner execute bit, and content. Symlinks and
// special files are rejected.
func DigestBundle(dir string) (string, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() {
		return "", errors.New("the Magefile bundle must be a directory")
	}
	h := sha256.New()
	files := 0
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("the Magefile bundle contains non-regular file %q", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		if err := hashFile(h, filepath.ToSlash(rel), file); err != nil {
			return err
		}
		files++
		return nil
	})
	if err != nil {
		return "", err
	}
	if files == 0 {
		return "", errors.New("the Magefile bundle is empty")
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func hashFile(h hash.Hash, rel string, file *os.File) error {
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("the Magefile bundle file is not regular")
	}
	if _, err := io.WriteString(h, rel); err != nil {
		return err
	}
	if _, err := h.Write([]byte{0}); err != nil {
		return err
	}
	if err := binary.Write(h, binary.LittleEndian, info.Size()); err != nil {
		return err
	}
	if _, err := h.Write([]byte{byte((info.Mode().Perm() & 0100) >> 6)}); err != nil {
		return err
	}
	_, err = io.Copy(h, file)
	return err
}

func copyBundle(source, dest string) error {
	if err := os.MkdirAll(dest, 0700); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.Mkdir(target, 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("the Magefile bundle contains non-regular file %q", rel)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = in.Close() }()
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0100 != 0 {
			return os.Chmod(target, 0700)
		}
		return nil
	})
}

func validateDigest(value string) error {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return errors.New("invalid Magefile bundle digest")
	}
	encoded := strings.TrimPrefix(value, "sha256:")
	decoded, err := hex.DecodeString(encoded)
	if err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != encoded {
		return errors.New("invalid Magefile bundle digest")
	}
	return nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

type boundedWriter struct {
	writer    io.Writer
	remaining int
	truncated bool
}

func newBoundedWriter(writer io.Writer) io.Writer {
	if writer == nil {
		writer = io.Discard
	}
	return &boundedWriter{writer: writer, remaining: maxLogBytes}
}

func (writer *boundedWriter) Write(data []byte) (int, error) {
	length := len(data)
	overflow := length > writer.remaining
	if writer.remaining > 0 {
		chunk := data
		if len(chunk) > writer.remaining {
			chunk = chunk[:writer.remaining]
		}
		written, err := writer.writer.Write(chunk)
		if err != nil {
			return 0, err
		}
		if written != len(chunk) {
			return 0, io.ErrShortWrite
		}
		writer.remaining -= written
	}
	if overflow && !writer.truncated {
		writer.truncated = true
		_, _ = io.WriteString(writer.writer, "\n[Mage output truncated]\n")
	}
	return length, nil
}

// WaitDelay limits the time spent waiting for descendants after cancellation.
func setWaitDelay(cmd *exec.Cmd) { cmd.WaitDelay = 3 * time.Second }
