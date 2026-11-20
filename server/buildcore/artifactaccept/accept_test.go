package artifactaccept

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/server/buildplan"
)

func TestAcceptCArchiveBundle(t *testing.T) {
	archive := testAR(t, "object.o", []byte("object bytes"))
	data := testZIP(t, []testZIPEntry{{name: "implant.a", data: archive}, {name: "implant.h", data: []byte("header")}})
	bin := t.TempDir()
	writeResult(t, bin, "implant.zip", "implant.zip", data)
	want := buildplan.Artifact{Kind: "c-archive-bundle", Container: "zip", Target: buildplan.Target{OS: "linux", Arch: "amd64"}}
	got, info, err := Accept(bin, want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) || !reflect.DeepEqual(info.Type, want) || info.Name != "implant.zip" || info.Size != int64(len(data)) || len(info.SHA256) != 64 {
		t.Fatalf("wrong accepted artifact: %+v", info)
	}
}

func TestInspectBytesMatchesFileAcceptanceAndRejectsUnsafeOutput(t *testing.T) {
	archive := testAR(t, "object.o", []byte("object bytes"))
	data := testZIP(t, []testZIPEntry{{name: "implant.a", data: archive}, {name: "implant.h", data: []byte("header")}})
	want := buildplan.Artifact{Kind: "c-archive-bundle", Container: "zip", Target: buildplan.Target{OS: "linux", Arch: "amd64"}}
	bin := t.TempDir()
	writeResult(t, bin, "implant.zip", "implant.zip", data)
	_, fileInfo, err := Accept(bin, want)
	if err != nil {
		t.Fatal(err)
	}
	byteInfo, err := InspectBytes(data, want, "implant.zip")
	if err != nil || !reflect.DeepEqual(byteInfo, fileInfo) {
		t.Fatalf("in-memory acceptance differs from file acceptance: %+v, %+v, %v", byteInfo, fileInfo, err)
	}
	for _, name := range []string{"../escape", "nested/artifact", `C:\artifact`, " artifact", ""} {
		if _, err := InspectBytes(data, want, name); err == nil {
			t.Fatalf("unsafe display name %q was accepted", name)
		}
	}
	if _, err := InspectBytes(nil, want, "implant.zip"); err == nil {
		t.Fatal("empty artifact was accepted")
	}
	if _, err := InspectBytes([]byte("not a ZIP"), want, "implant.zip"); err == nil {
		t.Fatal("malformed archive was accepted")
	}
}

func TestAcceptRejectsUnsafePrimary(t *testing.T) {
	for _, primary := range []string{"../other", "/tmp/other", `C:\outside`, "sub/file", "result.json"} {
		t.Run(primary, func(t *testing.T) {
			bin := t.TempDir()
			writeResult(t, bin, primary, "safe.zip", nil)
			if _, _, err := Accept(bin, buildplan.Artifact{Kind: "c-archive-bundle", Container: "zip"}); err == nil {
				t.Fatal("accepted unsafe primary path")
			}
		})
	}
}

func TestAcceptRejectsSymlinks(t *testing.T) {
	bin := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(bin, "artifact")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeResult(t, bin, "artifact", "artifact", nil)
	if _, _, err := Accept(bin, buildplan.Artifact{Kind: "shellcode", Container: "raw"}); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("accepted symlinked output: %v", err)
	}
}

func TestAcceptRejectsResultSymlinkAndUnknownFields(t *testing.T) {
	bin := t.TempDir()
	outside := filepath.Join(t.TempDir(), "result.json")
	if err := os.WriteFile(outside, []byte(`{"primary":"artifact","displayName":"artifact"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(bin, resultFileName)); err == nil {
		if _, _, err := Accept(bin, buildplan.Artifact{Kind: "shellcode", Container: "raw"}); err == nil {
			t.Fatal("accepted symlinked result manifest")
		}
		if err := os.Remove(filepath.Join(bin, resultFileName)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, resultFileName), []byte(`{"primary":"artifact","displayName":"artifact","type":"raw"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Accept(bin, buildplan.Artifact{Kind: "shellcode", Container: "raw"}); err == nil {
		t.Fatal("accepted result manifest with type override")
	}
}

func TestAcceptRejectsZIPHazards(t *testing.T) {
	goodArchive := testAR(t, "object.o", []byte("object bytes"))
	for _, test := range []struct {
		name    string
		entries []testZIPEntry
	}{
		{"traversal", []testZIPEntry{{name: "../escape", data: []byte("x")}}},
		{"absolute", []testZIPEntry{{name: "/escape", data: []byte("x")}}},
		{"windows separator", []testZIPEntry{{name: "..\\escape", data: []byte("x")}}},
		{"duplicate", []testZIPEntry{{name: "foo", data: []byte("a")}, {name: "FOO", data: []byte("b")}}},
		{"unicode alias", []testZIPEntry{{name: "caf\u00e9.h", data: []byte("a")}, {name: "cafe\u0301.h", data: []byte("b")}}},
		{"case fold alias", []testZIPEntry{{name: "Stra\u00dfe.h", data: []byte("a")}, {name: "STRASSE.h", data: []byte("b")}}},
		{"parent case alias", []testZIPEntry{{name: "Dir/one.h", data: []byte("a")}, {name: "dir/two.h", data: []byte("b")}}},
		{"file ancestor", []testZIPEntry{{name: "foo", data: []byte("a")}, {name: "foo/bar", data: []byte("b")}}},
		{"symlink", []testZIPEntry{{name: "link", data: []byte("target"), symlink: true}}},
		{"missing header", []testZIPEntry{{name: "implant.a", data: goodArchive}}},
		{"bad nested archive", []testZIPEntry{{name: "implant.a", data: []byte("not an ar")}, {name: "implant.h", data: []byte("header")}}},
		{"high expansion", []testZIPEntry{{name: "bomb.txt", data: bytes.Repeat([]byte("A"), 2<<20)}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := testZIP(t, test.entries)
			bin := t.TempDir()
			writeResult(t, bin, "artifact.zip", "artifact.zip", data)
			want := buildplan.Artifact{Kind: "c-archive-bundle", Container: "zip", Target: buildplan.Target{OS: "linux", Arch: "amd64"}}
			if _, _, err := Accept(bin, want); err == nil {
				t.Fatal("accepted unsafe ZIP")
			}
		})
	}
}

func TestValidateARRejectsTraversalAndTruncation(t *testing.T) {
	for _, archive := range [][]byte{
		testAR(t, "../evil", []byte("object")),
		testAR(t, "object.o", []byte("object"))[:70],
	} {
		if err := validateAR(bytes.NewReader(archive), uint64(len(archive))); err == nil {
			t.Fatal("accepted unsafe ar archive")
		}
	}
}

func TestValidateZIPRejectsCorruptionAndForgedCount(t *testing.T) {
	data := testZIP(t, []testZIPEntry{{name: "file", data: []byte("nonempty data")}})
	forged := append([]byte(nil), data...)
	end := len(forged) - 22
	binary.LittleEndian.PutUint16(forged[end+10:end+12], 0xffff)
	if err := validateZIP(forged, false); err == nil {
		t.Fatal("accepted ZIP with unsafe entry count")
	}
	corrupt := append([]byte(nil), data...)
	reader, err := zip.NewReader(bytes.NewReader(corrupt), int64(len(corrupt)))
	if err != nil {
		t.Fatal(err)
	}
	offset, err := reader.File[0].DataOffset()
	if err != nil {
		t.Fatal(err)
	}
	corrupt[offset] ^= 0xff
	if err := validateZIP(corrupt, false); err == nil {
		t.Fatal("accepted ZIP with corrupted member")
	}
}

func TestAcceptHostExecutableTarget(t *testing.T) {
	container := map[string]string{"windows": "pe", "linux": "elf", "darwin": "macho"}[runtime.GOOS]
	if container == "" || runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" && runtime.GOARCH != "386" {
		t.Skip("host binary format is not in the core target matrix")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(data)) > maxPrimaryBytes {
		t.Skip("test executable exceeds build output limit")
	}
	bin := t.TempDir()
	writeResult(t, bin, "host-bin", "host-bin", data)
	want := buildplan.Artifact{Kind: "executable", Container: container, Target: buildplan.Target{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	if _, _, err := Accept(bin, want); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectBytes(data, want, "host-bin"); err != nil {
		t.Fatalf("in-memory executable acceptance failed: %v", err)
	}
	wrong := want
	if runtime.GOARCH == "amd64" {
		wrong.Target.Arch = "arm64"
	} else {
		wrong.Target.Arch = "amd64"
	}
	if _, _, err := Accept(bin, wrong); err == nil {
		t.Fatal("accepted executable with wrong target architecture")
	}
	if _, err := InspectBytes(data, wrong, "host-bin"); err == nil {
		t.Fatal("in-memory acceptance accepted executable with wrong target architecture")
	}
}

type testZIPEntry struct {
	name    string
	data    []byte
	symlink bool
}

func testZIP(t *testing.T, entries []testZIPEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		if entry.symlink {
			header.SetMode(os.ModeSymlink | 0777)
		}
		file, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func testAR(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	buffer.Write(arMagic)
	header := fmt.Sprintf("%-16s%-12s%-6s%-6s%-8s%-10d`\n", name+"/", "0", "0", "0", "644", len(data))
	if len(header) != 60 {
		t.Fatalf("bad ar test header size: %d", len(header))
	}
	buffer.WriteString(header)
	buffer.Write(data)
	if len(data)%2 == 1 {
		buffer.WriteByte('\n')
	}
	return buffer.Bytes()
}

func writeResult(t *testing.T, binDir, primary, displayName string, data []byte) {
	t.Helper()
	result, err := json.Marshal(Result{Primary: primary, DisplayName: displayName})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, resultFileName), result, 0600); err != nil {
		t.Fatal(err)
	}
	if data != nil {
		if err := os.WriteFile(filepath.Join(binDir, primary), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
