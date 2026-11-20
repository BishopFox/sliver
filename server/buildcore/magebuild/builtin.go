package magebuild

import (
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed builtin/sliver/magefile.go
var builtinSliverFiles embed.FS

// BuiltinSliverBundle exposes the same Magefile source used by the compiled
// server runner, so a developer can invoke it with the Mage CLI.
func BuiltinSliverBundle() (fs.FS, error) {
	return fs.Sub(builtinSliverFiles, "builtin/sliver")
}

// BuiltinSliverDigest is the immutable content digest of the embedded bundle.
func BuiltinSliverDigest() (string, error) {
	data, err := builtinSliverFiles.ReadFile("builtin/sliver/magefile.go")
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.WriteString(h, "magefile.go"); err != nil {
		return "", err
	}
	if _, err := h.Write([]byte{0}); err != nil {
		return "", err
	}
	if err := binary.Write(h, binary.LittleEndian, int64(len(data))); err != nil {
		return "", err
	}
	if _, err := h.Write([]byte{0}); err != nil {
		return "", err
	}
	if _, err := h.Write(data); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// StageBuiltinSliverBundle writes the embedded Magefile to a fresh or empty
// private directory under one build workspace. It refuses symlinks and files.
func StageBuiltinSliverBundle(dir string) (string, error) {
	data, err := builtinSliverFiles.ReadFile("builtin/sliver/magefile.go")
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(dir, 0700); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	} else if !info.IsDir() || runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return "", errors.New("Magefile destination must be a private directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	if len(entries) != 0 {
		return "", errors.New("Magefile destination is not empty")
	}
	path := filepath.Join(dir, "magefile.go")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return "", writeErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	digest, err := DigestBundle(dir)
	if err != nil {
		return "", err
	}
	want, err := BuiltinSliverDigest()
	if err != nil || digest != want {
		return "", errors.New("staged Magefile differs from embedded bundle")
	}
	return digest, nil
}
