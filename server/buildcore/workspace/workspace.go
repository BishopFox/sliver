// Package workspace stages installed core source and build inputs in a private,
// per-build directory under the Sliver application directory.
package workspace

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var componentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var legacyNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
var buildIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// Workspace is a private copy of one core's source and build inputs. ID is
// unique even when concurrent builds use the same implant name and target.
type Workspace struct {
	ID           string
	Root         string
	SourceDir    string
	MagefilesDir string
	InputsDir    string
	BinDir       string

	appDir  string
	rootRel string
}

// New creates a per-build directory with the legacy src/bin layout. The
// caller owns the workspace and must call Remove after ingesting its output.
func New(appDir, targetOS, targetArch, name string) (*Workspace, error) {
	if appDir == "" {
		return nil, errors.New("missing app directory")
	}
	for label, value := range map[string]string{"target OS": targetOS, "target architecture": targetArch} {
		if !safeComponent(value) {
			return nil, fmt.Errorf("invalid %s %q", label, value)
		}
	}
	nameDir, err := safeNameDirectory(name)
	if err != nil {
		return nil, err
	}
	appDir, err = filepath.Abs(appDir)
	if err != nil {
		return nil, fmt.Errorf("resolve app directory: %w", err)
	}
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return nil, fmt.Errorf("create app directory: %w", err)
	}
	appRoot, err := os.OpenRoot(appDir)
	if err != nil {
		return nil, fmt.Errorf("open app directory: %w", err)
	}
	defer appRoot.Close()
	parent := filepath.Join("slivers", targetOS, targetArch, nameDir)
	if err := appRoot.MkdirAll(parent, 0o700); err != nil {
		return nil, fmt.Errorf("create build parent: %w", err)
	}
	var id, rel string
	for attempt := 0; attempt < 8; attempt++ {
		nonce := make([]byte, 16)
		if _, err := rand.Read(nonce); err != nil {
			return nil, fmt.Errorf("allocate build ID: %w", err)
		}
		id = hex.EncodeToString(nonce)
		rel = filepath.Join(parent, id)
		err = appRoot.Mkdir(rel, 0o700)
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("create build workspace: %w", err)
		}
	}
	if err != nil {
		return nil, errors.New("could not allocate a unique build workspace")
	}
	for _, directory := range []string{"src", "magefiles", "inputs", "bin"} {
		if err := appRoot.Mkdir(filepath.Join(rel, directory), 0o700); err != nil {
			_ = appRoot.RemoveAll(rel)
			return nil, fmt.Errorf("create %s directory: %w", directory, err)
		}
	}
	root := filepath.Join(appDir, rel)
	return &Workspace{
		ID: id, Root: root, SourceDir: filepath.Join(root, "src"),
		MagefilesDir: filepath.Join(root, "magefiles"), InputsDir: filepath.Join(root, "inputs"),
		BinDir: filepath.Join(root, "bin"), appDir: appDir, rootRel: rel,
	}, nil
}

// CopySource copies installed source bytes without interpreting templates.
func (w *Workspace) CopySource(source fs.FS) error {
	return copyFS(source, w.SourceDir)
}

// CopyMagefiles stages the installed Magefile source for CLI use.
func (w *Workspace) CopyMagefiles(source fs.FS) error {
	return copyFS(source, w.MagefilesDir)
}

// WriteInput writes one private input file; secret values must never be passed
// in an argument or environment variable to the Mage runner.
func (w *Workspace) WriteInput(name string, contents []byte) (string, error) {
	if w == nil || !safeComponent(name) {
		return "", fmt.Errorf("invalid input file name %q", name)
	}
	root, err := os.OpenRoot(w.InputsDir)
	if err != nil {
		return "", fmt.Errorf("open input directory: %w", err)
	}
	defer root.Close()
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create input file: %w", err)
	}
	_, writeErr := file.Write(contents)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = root.Remove(name)
		return "", errors.Join(writeErr, closeErr)
	}
	return filepath.Join(w.InputsDir, name), nil
}

// Remove deletes the workspace through a root-relative handle, so a replaced
// parent symlink cannot redirect cleanup outside the application directory.
func (w *Workspace) Remove() error {
	if w == nil || w.appDir == "" || w.rootRel == "" {
		return errors.New("invalid build workspace")
	}
	root, err := os.OpenRoot(w.appDir)
	if err != nil {
		return fmt.Errorf("open app directory: %w", err)
	}
	defer root.Close()
	return root.RemoveAll(w.rootRel)
}

// RemoveOutputWorkspace removes the private workspace that produced an output
// after its bytes have been ingested. The output path must be a direct child of
// bin/ in a workspace created by New for this name and target. The output file
// itself need not still exist, so cleanup remains possible after a failed
// post-build operation. This is a cleanup guard, not output validation.
func RemoveOutputWorkspace(appDir, targetOS, targetArch, name, outputPath string) error {
	if !safeComponent(targetOS) || !safeComponent(targetArch) {
		return errors.New("invalid build target")
	}
	nameDir, err := safeNameDirectory(name)
	if err != nil {
		return err
	}
	if appDir == "" || !filepath.IsAbs(outputPath) || filepath.Clean(outputPath) != outputPath {
		return errors.New("output path is not a clean absolute path")
	}
	appDir, err = filepath.Abs(appDir)
	if err != nil {
		return fmt.Errorf("resolve app directory: %w", err)
	}
	parentRel := filepath.Join("slivers", targetOS, targetArch, nameDir)
	parent := filepath.Join(appDir, parentRel)
	underParent, err := filepath.Rel(parent, outputPath)
	if err != nil {
		return fmt.Errorf("resolve output workspace: %w", err)
	}
	parts := strings.Split(underParent, string(filepath.Separator))
	if len(parts) != 3 || !buildIDPattern.MatchString(parts[0]) || parts[1] != "bin" ||
		!safeComponent(parts[2]) {
		return errors.New("output path is outside a unique build workspace bin directory")
	}
	root, err := os.OpenRoot(appDir)
	if err != nil {
		return fmt.Errorf("open app directory: %w", err)
	}
	defer root.Close()
	workspaceRel := filepath.Join(parentRel, parts[0])
	for _, rel := range []string{
		"slivers", filepath.Join("slivers", targetOS),
		filepath.Join("slivers", targetOS, targetArch), parentRel,
		workspaceRel, filepath.Join(workspaceRel, "bin"),
	} {
		info, err := root.Lstat(rel)
		if err != nil {
			return fmt.Errorf("inspect output workspace: %w", err)
		}
		if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("output workspace path %q is not a directory", rel)
		}
	}
	if err := root.RemoveAll(workspaceRel); err != nil {
		return fmt.Errorf("remove output workspace: %w", err)
	}
	return nil
}

func safeComponent(value string) bool {
	return value != "." && value != ".." && componentPattern.MatchString(value) &&
		!strings.HasSuffix(value, ".")
}

func safeNameDirectory(name string) (string, error) {
	if name == "" || name == "." || strings.HasPrefix(name, "..") || !legacyNamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid name %q", name)
	}
	if len(name) <= 80 && safeComponent(name) && !windowsReservedName(name) {
		return name, nil
	}
	// Legacy names such as .hidden and name. remain valid logical implant names,
	// but use a portable directory component to avoid Windows path aliases.
	sum := sha256.Sum256([]byte(name))
	return "name-" + hex.EncodeToString(sum[:16]), nil
}

func windowsReservedName(name string) bool {
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	}
	return false
}

func copyFS(source fs.FS, dest string) error {
	if source == nil {
		return errors.New("missing source bundle")
	}
	if err := requireEmptyDir(dest); err != nil {
		return err
	}
	root, err := os.OpenRoot(dest)
	if err != nil {
		return fmt.Errorf("open copy destination: %w", err)
	}
	defer root.Close()
	seen := map[string]struct{}{}
	var entries int
	var totalBytes int64
	return fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		local, err := safeEntryPath(name)
		if err != nil {
			return err
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate source path %q", name)
		}
		seen[key] = struct{}{}
		entries++
		if entries > defaultMaxEntries {
			return errors.New("source bundle has too many entries")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return root.MkdirAll(local, 0o700)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular source path %q", name)
		}
		if info.Size() < 0 || info.Size() > defaultMaxFileBytes || info.Size() > defaultMaxTotalBytes-totalBytes {
			return fmt.Errorf("source file %q exceeds size limit", name)
		}
		if err := root.MkdirAll(filepath.Dir(local), 0o700); err != nil {
			return err
		}
		in, err := source.Open(name)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0o600)
		if info.Mode()&0o100 != 0 {
			mode = 0o700
		}
		out, err := root.OpenFile(local, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if err != nil {
			_ = in.Close()
			return err
		}
		limit := int64(defaultMaxFileBytes)
		if remaining := defaultMaxTotalBytes - totalBytes; remaining < limit {
			limit = remaining
		}
		count, copyErr := io.Copy(out, io.LimitReader(in, limit+1))
		outCloseErr := out.Close()
		inCloseErr := in.Close()
		if count > limit || count != info.Size() || copyErr != nil || outCloseErr != nil || inCloseErr != nil {
			_ = root.Remove(local)
			if count > limit || count != info.Size() {
				return fmt.Errorf("source file %q changed or exceeded its size limit", name)
			}
			return errors.Join(copyErr, outCloseErr, inCloseErr)
		}
		totalBytes += count
		return nil
	})
}

func requireEmptyDir(directory string) error {
	info, err := os.Lstat(directory)
	if err != nil {
		return fmt.Errorf("stat bundle destination: %w", err)
	}
	if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("bundle destination %q is not a directory", directory)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read bundle destination: %w", err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("bundle destination %q is not empty", directory)
	}
	return nil
}

func safeEntryPath(name string) (string, error) {
	if !utf8.ValidString(name) || !fs.ValidPath(name) || strings.ContainsAny(name, "\\:\x00") {
		return "", fmt.Errorf("unsafe bundle path %q", name)
	}
	local, err := filepath.Localize(name)
	if err != nil {
		return "", fmt.Errorf("unsafe bundle path %q: %w", name, err)
	}
	return local, nil
}
