package util

import (
	"archive/tar"
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/klauspost/compress/gzip"
)

type tarTestMember struct {
	name     string
	typeflag byte
	data     []byte
}

func writeTestTarGz(t *testing.T, members ...tarTestMember) string {
	t.Helper()
	archivePath := t.TempDir() + "/test.tar.gz"
	archive, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(archive)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, member := range members {
		memberSize := int64(0)
		if member.typeflag == tar.TypeReg {
			memberSize = int64(len(member.data))
		}
		header := &tar.Header{
			Name:     member.name,
			Mode:     0o600,
			Size:     memberSize,
			Typeflag: member.typeflag,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if member.typeflag == tar.TypeReg {
			if _, err := tarWriter.Write(member.data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func TestReadFileFromTarGzNormalizesLeadingDotSlash(t *testing.T) {
	tests := []struct {
		name        string
		archiveName string
		requestName string
	}{
		{name: "both prefixed", archiveName: "./file.txt", requestName: "./file.txt"},
		{name: "archive prefixed", archiveName: "./file.txt", requestName: "file.txt"},
		{name: "request prefixed", archiveName: "file.txt", requestName: "./file.txt"},
		{name: "repeated prefix", archiveName: "././file.txt", requestName: "./file.txt"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archivePath := writeTestTarGz(t, tarTestMember{
				name: test.archiveName, typeflag: tar.TypeReg, data: []byte("contents"),
			})
			data, err := ReadFileFromTarGz(archivePath, test.requestName)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "contents" {
				t.Fatalf("data = %q, want contents", data)
			}
		})
	}
}

func TestReadFileFromTarGzRejectsDifferentOrNonRegularPaths(t *testing.T) {
	tests := []struct {
		name        string
		archiveName string
		requestName string
		typeflag    byte
	}{
		{name: "missing", archiveName: "other.txt", requestName: "file.txt", typeflag: tar.TypeReg},
		{name: "parent path", archiveName: "file.txt", requestName: "../file.txt", typeflag: tar.TypeReg},
		{name: "absolute path", archiveName: "file.txt", requestName: "/file.txt", typeflag: tar.TypeReg},
		{name: "archive parent path", archiveName: "../file.txt", requestName: "file.txt", typeflag: tar.TypeReg},
		{name: "archive absolute path", archiveName: "/file.txt", requestName: "file.txt", typeflag: tar.TypeReg},
		{name: "directory", archiveName: "file.txt", requestName: "file.txt", typeflag: tar.TypeDir},
		{name: "symlink", archiveName: "file.txt", requestName: "file.txt", typeflag: tar.TypeSymlink},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archivePath := writeTestTarGz(t, tarTestMember{
				name: test.archiveName, typeflag: test.typeflag, data: []byte("contents"),
			})
			_, err := ReadFileFromTarGz(archivePath, test.requestName)
			if !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("error = %v, want fs.ErrNotExist", err)
			}
		})
	}
}

func TestReadFileFromTarGzReturnsEmptyRegularFile(t *testing.T) {
	archivePath := writeTestTarGz(t, tarTestMember{name: "file.txt", typeflag: tar.TypeReg})
	data, err := ReadFileFromTarGz(archivePath, "./file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("data length = %d, want 0", len(data))
	}
}
