package e2e

import (
	"archive/tar"
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

type testTarEntry struct {
	name string
	data string
}

func makeTestTar(t *testing.T, entries ...testTarEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	for _, entry := range entries {
		data := []byte(entry.data)
		if err := writer.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o600, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatalf("write tar header %q: %v", entry.name, err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatalf("write tar data %q: %v", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	return buffer.Bytes()
}

func TestReadTarAcceptsSafeRegularFiles(t *testing.T) {
	entries, err := readTar(makeTestTar(t,
		testTarEntry{name: "root/alpha.txt", data: "alpha"},
		testTarEntry{name: "root/nested/beta.txt", data: "beta"},
	))
	if err != nil {
		t.Fatalf("read tar: %v", err)
	}
	if got := string(entries["root/alpha.txt"]); got != "alpha" {
		t.Fatalf("alpha content got %q", got)
	}
	if got := string(entries["root/nested/beta.txt"]); got != "beta" {
		t.Fatalf("beta content got %q", got)
	}
}

func TestReadTarRejectsTraversalAndAbsolutePaths(t *testing.T) {
	for _, name := range []string{"../escape", `..\escape`, "safe/../../escape", "/absolute"} {
		t.Run(strings.ReplaceAll(name, "/", "_"), func(t *testing.T) {
			_, err := readTar(makeTestTar(t, testTarEntry{name: name, data: "payload"}))
			if err == nil || !strings.Contains(err.Error(), "unsafe path") {
				t.Fatalf("error got %v, want unsafe path rejection", err)
			}
		})
	}
}

func TestReadTarRejectsDuplicateCanonicalPath(t *testing.T) {
	_, err := readTar(makeTestTar(t,
		testTarEntry{name: "dir/../same.txt", data: "first"},
		testTarEntry{name: "same.txt", data: "second"},
	))
	if err == nil || !strings.Contains(err.Error(), "duplicate path") {
		t.Fatalf("error got %v, want duplicate path rejection", err)
	}
}

func TestArchiveContentBySuffix(t *testing.T) {
	entries := map[string][]byte{
		"root/alpha.txt":       []byte("alpha"),
		"root/nested/beta.txt": []byte("beta"),
	}
	content, err := archiveContentBySuffix(entries, "nested/beta.txt")
	if err != nil {
		t.Fatalf("find archive content: %v", err)
	}
	if got := string(content); got != "beta" {
		t.Fatalf("content got %q, want beta", got)
	}

	if _, err := archiveContentBySuffix(entries, "nested/missing.txt"); err == nil || !strings.Contains(err.Error(), "archive missing") {
		t.Fatalf("missing error got %v", err)
	}

	entries["other/nested/beta.txt"] = []byte("duplicate")
	if _, err := archiveContentBySuffix(entries, "nested/beta.txt"); err == nil || !strings.Contains(err.Error(), "duplicate relative suffix") {
		t.Fatalf("duplicate error got %v", err)
	}
}

func TestEnsureWithinRoot(t *testing.T) {
	root := t.TempDir()
	if err := ensureWithinRoot(root, filepath.Join(root, "nested", "file.txt")); err != nil {
		t.Fatalf("child path rejected: %v", err)
	}

	for _, candidate := range []string{
		root,
		filepath.Join(root, "..", "outside"),
		root + "-lookalike",
	} {
		if err := ensureWithinRoot(root, candidate); err == nil {
			t.Fatalf("unsafe candidate %q was accepted", candidate)
		}
	}
}

func TestGrepHasExactMatch(t *testing.T) {
	expectedPath := filepath.Join(t.TempDir(), "seed.txt")
	matching := &sliverpb.Grep{Results: map[string]*sliverpb.GrepResultsForFile{
		expectedPath: {FileResults: []*sliverpb.GrepResult{{
			LineNumber: 2,
			Line:       "prefix beta-marker suffix",
			Positions:  []*sliverpb.GrepLinePosition{{Start: 7, End: 18}},
		}}},
	}}
	if !grepHasExactMatch(matching, expectedPath, 2, "beta-marker") {
		t.Fatal("expected exact grep match")
	}

	tests := []struct {
		name     string
		path     string
		line     int64
		text     string
		position bool
	}{
		{name: "error sentinel line", path: expectedPath, line: -1, text: "beta-marker", position: true},
		{name: "wrong line", path: expectedPath, line: 3, text: "beta-marker", position: true},
		{name: "missing marker", path: expectedPath, line: 2, text: "unrelated", position: true},
		{name: "missing position", path: expectedPath, line: 2, text: "beta-marker"},
		{name: "wrong path", path: expectedPath + ".other", line: 2, text: "beta-marker", position: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			positions := []*sliverpb.GrepLinePosition(nil)
			if test.position {
				positions = []*sliverpb.GrepLinePosition{{Start: 0, End: 1}}
			}
			response := &sliverpb.Grep{Results: map[string]*sliverpb.GrepResultsForFile{
				test.path: {FileResults: []*sliverpb.GrepResult{{LineNumber: test.line, Line: test.text, Positions: positions}}},
			}}
			if grepHasExactMatch(response, expectedPath, 2, "beta-marker") {
				t.Fatal("unexpected exact grep match")
			}
		})
	}
}

func TestGrepHasExactMatchWithContext(t *testing.T) {
	expectedPath := filepath.Join(t.TempDir(), "seed.txt")
	response := &sliverpb.Grep{Results: map[string]*sliverpb.GrepResultsForFile{
		expectedPath: {FileResults: []*sliverpb.GrepResult{{
			LineNumber:  2,
			Line:        "beta",
			Positions:   []*sliverpb.GrepLinePosition{{Start: 0, End: 4}},
			LinesBefore: []string{"alpha"},
			LinesAfter:  []string{"gamma"},
		}}},
	}}
	if !grepHasExactMatchWithContext(response, expectedPath, 2, "beta", []string{"alpha"}, []string{"gamma"}) {
		t.Fatal("exact grep context was rejected")
	}
	if grepHasExactMatchWithContext(response, expectedPath, 2, "beta", []string{"wrong"}, []string{"gamma"}) {
		t.Fatal("incorrect preceding context was accepted")
	}
	if grepHasExactMatchWithContext(response, expectedPath, 2, "beta", []string{"alpha"}, nil) {
		t.Fatal("missing following context was accepted")
	}
}

func TestNetstatContainsSocketRequiresProtocolAndState(t *testing.T) {
	entries := []*sliverpb.SockTabEntry{{
		LocalAddr: &sliverpb.SockTabEntry_SockAddr{Ip: "127.0.0.1", Port: 4141},
		SkState:   "LISTEN",
		Protocol:  "tcp",
	}}
	if !netstatContainsSocket(entries, 4141, "tcp", "LISTEN") {
		t.Fatal("exact socket was rejected")
	}
	for _, test := range []struct {
		name     string
		port     uint32
		protocol string
		state    string
	}{
		{name: "wrong port", port: 4142, protocol: "tcp", state: "LISTEN"},
		{name: "wrong protocol", port: 4141, protocol: "udp", state: "LISTEN"},
		{name: "wrong state", port: 4141, protocol: "tcp", state: "ESTABLISHED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if netstatContainsSocket(entries, test.port, test.protocol, test.state) {
				t.Fatal("mismatched socket was accepted")
			}
		})
	}
	if !netstatContainsSocket(entries, 4141, "tcp", "") {
		t.Fatal("state-agnostic socket match was rejected")
	}
}

func TestEnvHasKeyIsCaseInsensitiveAndDistinguishesEmptyValue(t *testing.T) {
	variables := []*commonpb.EnvVar{
		{Key: "SLIVER_E2E_EMPTY", Value: ""},
		{Key: "OTHER", Value: "value"},
	}
	if !envHasKey(variables, "sliver_e2e_empty") {
		t.Fatal("empty-valued environment key was reported absent")
	}
	if envHasKey(variables, "missing") {
		t.Fatal("missing environment key was reported present")
	}
	if envHasKey(nil, "missing") {
		t.Fatal("nil environment list reported a key present")
	}
}

func TestVerifyTrackedChildRequiresLiveExactPIDAndPath(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "helper")
	children := []*sliverpb.ExecuteChild{{Pid: 41, Path: filepath.Join(filepath.Dir(executable), ".", filepath.Base(executable))}}
	if err := verifyTrackedChild(children, 41, executable); err != nil {
		t.Fatalf("live tracked child rejected: %v", err)
	}

	tests := []struct {
		name      string
		children  []*sliverpb.ExecuteChild
		wantError string
	}{
		{name: "missing PID", children: children, wantError: "live child PID 99 missing"},
		{name: "already exited", children: []*sliverpb.ExecuteChild{{Pid: 99, Path: executable, Exited: true}}, wantError: "already exited"},
		{name: "wrong path", children: []*sliverpb.ExecuteChild{{Pid: 99, Path: executable + ".other"}}, wantError: "path got"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := verifyTrackedChild(test.children, 99, executable)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error got %v, want substring %q", err, test.wantError)
			}
		})
	}
}
