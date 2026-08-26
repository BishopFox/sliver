package assets

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open tar.gz: %w", err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer gz.Close()

	root, err := openArchiveRoot(destDir, 0o755)
	if err != nil {
		return err
	}
	defer root.Close()

	tarReader := tar.NewReader(gz)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		entryPath, err := archiveEntryPath(hdr.Name)
		if err != nil {
			return err
		}

		mode := hdr.FileInfo().Mode()
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := root.MkdirAll(entryPath, mode.Perm()); err != nil {
				return fmt.Errorf("create dir: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := writeArchiveFile(root, entryPath, mode.Perm(), 0o755, tarReader); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := validateArchiveSymlink(hdr.Name, hdr.Linkname); err != nil {
				return err
			}
			if err := root.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
				return fmt.Errorf("create parent dir: %w", err)
			}
			if err := root.Symlink(filepath.FromSlash(hdr.Linkname), entryPath); err != nil {
				return fmt.Errorf("create symlink: %w", err)
			}
		default:
			// Skip non-file entries like extended headers.
			continue
		}
	}

	return nil
}

func extractZip(archivePath, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	root, err := openArchiveRoot(destDir, 0o755)
	if err != nil {
		return err
	}
	defer root.Close()

	for _, file := range reader.File {
		entryPath, err := archiveEntryPath(file.Name)
		if err != nil {
			return err
		}
		info := file.FileInfo()
		if info.IsDir() {
			if err := root.MkdirAll(entryPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("create dir: %w", err)
			}
			continue
		}

		in, err := file.Open()
		if err != nil {
			return fmt.Errorf("open zip entry: %w", err)
		}
		writeErr := writeArchiveFile(root, entryPath, info.Mode().Perm(), 0o755, in)
		closeErr := in.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return fmt.Errorf("close zip entry: %w", closeErr)
		}
	}

	return nil
}

func zipDir(baseDir, relRoot, destZip string) error {
	root := filepath.Join(baseDir, relRoot)

	if err := ensureDir(filepath.Dir(destZip)); err != nil {
		return err
	}
	zipFile, err := os.Create(destZip)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			if !strings.HasSuffix(header.Name, "/") {
				header.Name += "/"
			}
			header.Method = zip.Store
			_, err := zw.CreateHeader(header)
			return err
		}

		header.Method = zip.Deflate
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(writer, file); err != nil {
			file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}

		return nil
	})
}

func openArchiveRoot(destDir string, mode fs.FileMode) (*os.Root, error) {
	if err := os.MkdirAll(destDir, mode); err != nil {
		return nil, fmt.Errorf("create archive destination: %w", err)
	}
	root, err := os.OpenRoot(destDir)
	if err != nil {
		return nil, fmt.Errorf("open archive destination: %w", err)
	}
	return root, nil
}

func archiveEntryPath(name string) (string, error) {
	archiveName := strings.TrimSuffix(name, "/")
	if archiveName == "." || !fs.ValidPath(archiveName) {
		return "", fmt.Errorf("invalid path in archive: %q", name)
	}
	entryPath, err := filepath.Localize(archiveName)
	if err != nil {
		return "", fmt.Errorf("invalid path in archive %q: %w", name, err)
	}
	return entryPath, nil
}

func writeArchiveFile(root *os.Root, name string, mode, parentMode fs.FileMode, src io.Reader) error {
	if err := root.MkdirAll(filepath.Dir(name), parentMode); err != nil {
		return fmt.Errorf("create archive parent directory: %w", err)
	}
	out, err := root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	_, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("write archive file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close archive file: %w", closeErr)
	}
	return nil
}

func validateArchiveSymlink(name, target string) error {
	archiveName := strings.TrimSuffix(name, "/")
	target = strings.ReplaceAll(target, `\`, "/")
	nativeTarget := filepath.FromSlash(target)
	if path.IsAbs(target) || filepath.IsAbs(nativeTarget) || filepath.VolumeName(nativeTarget) != "" {
		return fmt.Errorf("absolute symlink target in archive: %q", target)
	}
	resolved := path.Clean(path.Join(path.Dir(archiveName), target))
	if resolved == "." || !fs.ValidPath(resolved) {
		return fmt.Errorf("symlink target escapes archive destination: %q", target)
	}
	return nil
}
