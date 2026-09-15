package extensions

import (
	"archive/tar"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/klauspost/compress/gzip"
	"github.com/spf13/cobra"
)

type installArchiveMember struct {
	name string
	data []byte
}

const installTestManifest = `{
	"name": "Test Extension",
	"package_name": "test-extension",
	"version": "v1.0.0",
	"commands": [{
		"command_name": "test-command",
		"help": "test command",
		"files": [{"os": "windows", "arch": "amd64", "path": "foo/test.dll"}]
	}]
}`

func writeInstallArchive(t *testing.T, prefix string, members ...installArchiveMember) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "extension.tar.gz")
	archive, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(archive)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, member := range members {
		header := &tar.Header{
			Name:     prefix + member.name,
			Mode:     0o600,
			Size:     int64(len(member.data)),
			Typeflag: tar.TypeReg,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(member.data); err != nil {
			t.Fatal(err)
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

func testInstallConsole(t *testing.T) *console.SliverClient {
	t.Helper()
	con := console.NewConsole(false)
	emptyCommands := func() *cobra.Command { return &cobra.Command{Use: "test"} }
	if err := console.StartClient(
		con, nil, nil, nil, emptyCommands, emptyCommands, false, "",
	); err != nil {
		t.Fatal(err)
	}
	return con
}

func TestInstallFromDirAcceptsPrefixedAndUnprefixedArchives(t *testing.T) {
	for _, prefix := range []string{"./", ""} {
		name := "unprefixed"
		if prefix != "" {
			name = "prefixed"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())
			archivePath := writeInstallArchive(t, prefix,
				installArchiveMember{name: ManifestFileName, data: []byte(installTestManifest)},
				installArchiveMember{name: "foo/test.dll", data: []byte("extension data")},
			)

			if err := InstallFromDir(archivePath, false, testInstallConsole(t), true); err != nil {
				t.Fatal(err)
			}
			installRoot := filepath.Join(assets.GetExtensionsDir(), "test-extension")
			if _, err := os.Stat(filepath.Join(installRoot, ManifestFileName)); err != nil {
				t.Fatalf("installed manifest: %v", err)
			}
			artifact, err := os.ReadFile(filepath.Join(installRoot, "foo", "test.dll"))
			if err != nil {
				t.Fatalf("installed artifact: %v", err)
			}
			if string(artifact) != "extension data" {
				t.Fatalf("artifact data = %q", artifact)
			}
		})
	}
}

func TestInstallFromDirReturnsMissingManifest(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())
	archivePath := writeInstallArchive(t, "./",
		installArchiveMember{name: "foo/test.dll", data: []byte("extension data")},
	)
	err := InstallFromDir(archivePath, false, testInstallConsole(t), true)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("error = %v, want fs.ErrNotExist", err)
	}
}

func TestInstallFromDirReturnsMissingArtifactAndCleansUp(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())
	archivePath := writeInstallArchive(t, "./",
		installArchiveMember{name: ManifestFileName, data: []byte(installTestManifest)},
	)
	err := InstallFromDir(archivePath, false, testInstallConsole(t), true)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("error = %v, want fs.ErrNotExist", err)
	}
	installRoot := filepath.Join(assets.GetExtensionsDir(), "test-extension")
	if _, err := os.Stat(installRoot); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("install directory was not cleaned up: %v", err)
	}
}
