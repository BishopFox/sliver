//go:build sliver_generate

package generate

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
)

// TestSliverArchiveAdvertisedTargets compiles both session and beacon archives
// for every target reported by the server's compiler inventory.
func TestSliverArchiveAdvertisedTargets(t *testing.T) {
	certs.SetupCAs()
	targets := make([]*clientpb.CompilerTarget, 0)
	seen := make(map[string]bool)
	for _, target := range GetCompilerTargets() {
		if target.Format != clientpb.OutputFormat_GO_ARCHIVE {
			continue
		}
		key := target.GOOS + "/" + target.GOARCH
		if seen[key] {
			continue
		}
		seen[key] = true
		targets = append(targets, target)
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].GOOS == targets[j].GOOS {
			return targets[i].GOARCH < targets[j].GOARCH
		}
		return targets[i].GOOS < targets[j].GOOS
	})
	if len(targets) == 0 {
		t.Skip("this server advertises no C archive compiler targets")
	}

	for _, target := range targets {
		for _, beacon := range []bool{false, true} {
			mode := "session"
			if beacon {
				mode = "beacon"
			}
			t.Run(target.GOOS+"/"+target.GOARCH+"/"+mode, func(t *testing.T) {
				buildArchiveForTarget(t, target, beacon)
			})
		}
	}
}

func buildArchiveForTarget(t *testing.T, target *clientpb.CompilerTarget, beacon bool) {
	t.Helper()
	mode := "session"
	if beacon {
		mode = "beacon"
	}
	name := fmt.Sprintf("archive_test_%s_%s_%s", target.GOOS, target.GOARCH, mode)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(GetSliversDir(), target.GOOS, target.GOARCH, name)) })

	config := &clientpb.ImplantConfig{
		GOOS: target.GOOS, GOARCH: target.GOARCH,
		C2:    []*clientpb.ImplantC2{{URL: "mtls://1.example.com"}},
		Debug: true, Format: clientpb.OutputFormat_GO_ARCHIVE,
		IsBeacon: beacon, IsSharedLib: true, IncludeMTLS: true,
		Exports: []string{"StartW"},
	}
	build, err := GenerateConfig(name, config)
	if err != nil {
		t.Fatalf("prepare %s archive: %v", mode, err)
	}
	httpConfig := configs.GenerateDefaultHTTPC2Config()
	gotPath, err := SliverArchive(name, build, config, httpConfig.ImplantConfig)
	if err != nil {
		t.Fatalf("compile %s archive for %s/%s: %v", mode, target.GOOS, target.GOARCH, err)
	}
	assertGeneratedOutputPath(t, gotPath, target.GOOS, target.GOARCH, name, name+".zip")
	assertArchiveBundle(t, gotPath, name)
}

func assertArchiveBundle(t *testing.T, archivePath, name string) {
	t.Helper()
	bundle, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open C archive bundle: %v", err)
	}
	defer bundle.Close()
	want := map[string]bool{name + ".a": false, name + ".h": false, "main.h": false}
	if len(bundle.File) != len(want) {
		t.Fatalf("bundle has %d entries, want %d", len(bundle.File), len(want))
	}
	for _, file := range bundle.File {
		if _, ok := want[file.Name]; !ok || want[file.Name] {
			t.Fatalf("unexpected or duplicate bundle entry %q", file.Name)
		}
		want[file.Name] = true
		if file.UncompressedSize64 == 0 {
			t.Fatalf("bundle entry %q is empty", file.Name)
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open bundle entry %q: %v", file.Name, err)
		}
		if file.Name == name+".a" {
			magic := make([]byte, len("!<arch>\n"))
			if _, err := io.ReadFull(reader, magic); err != nil || !bytes.Equal(magic, []byte("!<arch>\n")) {
				reader.Close()
				t.Fatalf("bundle entry %q is not an ar archive: %v", file.Name, err)
			}
		}
		if file.Name == name+".h" {
			header, err := io.ReadAll(reader)
			if err != nil {
				reader.Close()
				t.Fatalf("read bundle entry %q: %v", file.Name, err)
			}
			if !bytes.Contains(header, []byte("StartW")) {
				reader.Close()
				t.Fatalf("generated header %q is missing the StartW export", file.Name)
			}
		} else if _, err := io.Copy(io.Discard, reader); err != nil {
			reader.Close()
			t.Fatalf("read bundle entry %q: %v", file.Name, err)
		}
		if err := reader.Close(); err != nil {
			t.Fatalf("close bundle entry %q: %v", file.Name, err)
		}
	}
}
