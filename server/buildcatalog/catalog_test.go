package buildcatalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	raw, err := builtinSpecs.ReadFile("builtin/transform-xor.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestRestrictedYAMLRejectsCaseEquivalentKeys(t *testing.T) {
	for _, document := range []string{
		"kind: CoreSpec\nKind: TransformSpec\n",
		"metadata:\n  name: one\n  Name: two\n",
	} {
		if _, err := RestrictedYAMLJSON([]byte(document), MaxSpecYAMLBytes); err == nil {
			t.Fatalf("case-equivalent YAML keys were accepted: %q", document)
		}
	}
}

func changedFixture(t *testing.T, before, after string) []byte {
	t.Helper()
	return redigestFixture(t, replaceFixture(t, fixture(t), before, after))
}

func replaceFixture(t *testing.T, raw []byte, before, after string) []byte {
	t.Helper()
	// The embedded files may have CRLF line endings on Windows checkouts.
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	if bytes.Count(raw, []byte(before)) != 1 {
		t.Fatalf("expected one %q in fixture", before)
	}
	return bytes.Replace(raw, []byte(before), []byte(after), 1)
}

func TestReplaceFixtureWithCRLF(t *testing.T) {
	raw := []byte("  - os: windows\r\n    arch: amd64\r\n")
	want := "  - os: linux\n    arch: amd64\n"
	if got := replaceFixture(t, raw, "os: windows\n    arch: amd64", "os: linux\n    arch: amd64"); string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func redigestFixture(t *testing.T, raw []byte) []byte {
	t.Helper()
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicalJSON(document.Content[0], true)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	old := yamlField(yamlField(document.Content[0], "metadata"), "digest").Value
	return replaceFixture(t, raw, old, "sha256:"+hex.EncodeToString(digest[:]))
}

func TestOpenBuiltinInventoryAndCopy(t *testing.T) {
	catalog, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(catalog.List(Query{Kind: "CoreSpec"})); got != 2 {
		t.Fatalf("got %d built-in cores, want 2", got)
	}
	if got := len(catalog.List(Query{Kind: "TransformSpec"})); got != 11 {
		t.Fatalf("got %d built-in transforms, want 11", got)
	}
	sliver, err := catalog.Get(Reference{Name: "builtin/sliver"})
	if err != nil {
		t.Fatal(err)
	}
	if !sliver.Available || len(sliver.Providers) != 1 || sliver.Providers[0].Name != "server" || sliver.Core == nil {
		t.Fatalf("unexpected built-in provider or core: %+v", sliver)
	}
	sliver.Core.Metadata.Name = "changed/name"
	sliver.RawYAML[0] = 'X'
	again, err := catalog.Get(Reference{Name: "builtin/sliver"})
	if err != nil || again.Core.Metadata.Name != "builtin/sliver" || again.RawYAML[0] == 'X' {
		t.Fatalf("catalog snapshot was mutated: %+v, %v", again, err)
	}
	if len(catalog.List(Query{Kind: "TransformSpec", TargetOS: "linux", TargetArch: "386"})) == 0 {
		t.Fatal("expected target filtered transform")
	}
}

func TestStageAndMetadataBuiltinsHaveTypedLocalEdges(t *testing.T) {
	catalog, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		executorID string
		inputKind  string
		outputKind string
	}{
		{"builtin/stage-frame", "stage-frame-v1", "executable", "stage"},
		{"builtin/stage-compress", "stage-compress-v1", "stage", "stage"},
		{"builtin/stage-aes", "stage-aes-v1", "stage", "stage"},
		{"builtin/stage-rc4", "stage-rc4-v1", "stage", "stage"},
		{"builtin/pe-metadata", "pe-metadata-v1", "executable", "executable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot, err := catalog.Get(Reference{Name: test.name})
			if err != nil {
				t.Fatal(err)
			}
			if !snapshot.Available || snapshot.Transform == nil || snapshot.Transform.Executor.ID != test.executorID ||
				len(snapshot.Transform.Capabilities) == 0 {
				t.Fatalf("stage transform is not installed: %+v", snapshot)
			}
			found := false
			for _, capability := range snapshot.Transform.Capabilities {
				if capability.Input.Kind == test.inputKind && capability.Output.Kind == test.outputKind &&
					capability.Toolchain.Driver == "builtin" {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing typed %s -> %s edge", test.inputKind, test.outputKind)
			}
		})
	}
}

func TestStageAndMetadataBuiltinsRequireTypedOptions(t *testing.T) {
	catalog, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, option, typ string }{
		{"builtin/stage-aes", "key", "secret-ref"},
		{"builtin/stage-aes", "iv", "secret-ref"},
		{"builtin/stage-rc4", "key", "secret-ref"},
		{"builtin/pe-metadata", "donorArtifact", "input-artifact-ref"},
	} {
		snapshot, err := catalog.Get(Reference{Name: test.name})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, option := range snapshot.Transform.Options {
			if option.Name == test.option && option.Type == test.typ && option.Required {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s lacks required %s option of type %s", test.name, test.option, test.typ)
		}
	}
}

func TestCoreMageDeclarationRequiresInstalledRelativeBundle(t *testing.T) {
	catalog, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := builtinSpecs.ReadFile("builtin/core-sliver.yaml")
	if err != nil {
		t.Fatal(err)
	}
	raw = replaceFixture(t, raw, "version: 1.0.0", "version: 2.0.0")
	raw = replaceFixture(t, raw, "options:\n", "source:\n  mode: copy\n  renderer: builtin-sliver-template-v1\nbuilder:\n  engine: mage\n  magefile: magefiles/sliver\n  target: Build\noptions:\n")
	raw = redigestFixture(t, raw)
	if _, err := catalog.ImportAdministrator(raw); err != nil {
		t.Fatalf("valid installed Mage core was rejected: %v", err)
	}
	for _, test := range []struct{ name, before, after string }{
		{"path-traversal", "magefile: magefiles/sliver", "magefile: ../outside"},
		{"absolute-path", "magefile: magefiles/sliver", "magefile: /tmp/outside"},
		{"no-template-on-none", "mode: copy", "mode: none"},
		{"unsupported-engine", "engine: mage", "engine: shell"},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := replaceFixture(t, raw, test.before, test.after)
			candidate = redigestFixture(t, candidate)
			if _, err := parseSpec(candidate); err == nil {
				t.Fatal("unsafe Mage core declaration was accepted")
			}
		})
	}
}

func TestImportRejectsMalformedConflictsAndUnknownExecutor(t *testing.T) {
	catalog, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		raw  []byte
	}{
		{"duplicate-key", append(fixture(t), []byte("\nkind: TransformSpec\n")...)},
		{"unknown-field", append(fixture(t), []byte("\nmadeUp: true\n")...)},
		{"alias", []byte("foo: &x value\nbar: *x\n")},
		{"invalid-tuple", changedFixture(t, "os: windows\n    arch: amd64", "os: solaris\n    arch: amd64")},
		{"unsafe-integer", changedFixture(t, "version: 1.0.0", "version: 9007199254740992")},
		{"unknown-executor", changedFixture(t, "id: xor-v1", "id: uninstalled-v1")},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := catalog.ImportAdministrator(test.raw); err == nil {
				t.Fatal("expected import rejection")
			}
		})
	}
	conflict := changedFixture(t, "id: xor-v1", "id: xor-dynamic-v1")
	if _, err := catalog.ImportAdministrator(conflict); !errors.Is(err, ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
}

func TestRemoteBatchOfflineAndRestart(t *testing.T) {
	root := t.TempDir()
	catalog, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	unique := changedFixture(t, "name: builtin/xor", "name: test/xor")
	invalid := append(append([]byte(nil), unique...), []byte("\nunknown: true\n")...)
	if err := catalog.AdvertiseRemoteBatch("worker-one", [][]byte{unique, invalid}, time.Now()); err == nil {
		t.Fatal("expected batch rejection")
	}
	if _, err := catalog.Get(Reference{Name: "test/xor"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("invalid batch partially imported: %v", err)
	}
	seen := time.Now().UTC().Truncate(time.Second)
	if err := catalog.AdvertiseRemoteBatch("worker-one", [][]byte{unique}, seen); err != nil {
		t.Fatal(err)
	}
	active, err := catalog.Get(Reference{Name: "test/xor"})
	if err != nil || !active.Available || len(active.Providers) != 1 || !active.Providers[0].Online {
		t.Fatalf("unexpected active remote spec: %+v, %v", active, err)
	}
	if err := catalog.SetRemoteOnline("worker-one", false, seen); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertOfflineRemoteSpec(t, reopened, seen, unique)
	if _, err := reopened.AdvertiseRemote("worker-one", unique, time.Now()); err != nil {
		t.Fatal(err)
	}
	other := changedFixture(t, "name: builtin/xor", "name: test/other")
	if err := reopened.AdvertiseRemoteBatch("worker-one", [][]byte{other}, time.Now()); err != nil {
		t.Fatal(err)
	}
	omitted, err := reopened.Get(Reference{Name: "test/xor"})
	if err != nil || omitted.Available {
		t.Fatalf("omitted remote claim stayed online: %+v, %v", omitted, err)
	}
}

func assertOfflineRemoteSpec(t *testing.T, catalog *Catalog, seen time.Time, raw []byte) {
	t.Helper()
	offline, err := catalog.Get(Reference{Name: "test/xor"})
	if err != nil {
		t.Fatal(err)
	}
	if offline.Available || len(offline.Providers) != 1 || offline.Providers[0].Online || !offline.Providers[0].LastSeen.Equal(seen) {
		t.Fatalf("remote status changed on restart: %+v", offline)
	}
	if len(offline.Sources) != 1 || offline.Sources[0].Source != "remote" || !bytes.Equal(offline.RawYAML, raw) {
		t.Fatalf("remote YAML or source was not retained: %+v", offline)
	}
	if got := catalog.List(Query{Name: "test/xor", AvailableOnly: true}); len(got) != 0 {
		t.Fatalf("offline spec listed as available: %+v", got)
	}
}

func TestAdministratorDirectoryAndPersistence(t *testing.T) {
	root := t.TempDir()
	adminDir := filepath.Join(root, "build-catalog", "admin")
	if err := os.MkdirAll(adminDir, 0700); err != nil {
		t.Fatal(err)
	}
	unique := changedFixture(t, "name: builtin/xor", "name: test/xor")
	if err := os.WriteFile(filepath.Join(adminDir, "xor.yaml"), unique, 0600); err != nil {
		t.Fatal(err)
	}
	catalog, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := catalog.Get(Reference{Name: "test/xor"})
	if err != nil || !admin.Available || len(admin.Sources) != 1 || admin.Sources[0].Source != "administrator" {
		t.Fatalf("administrator spec was not imported: %+v, %v", admin, err)
	}
	if err := os.Remove(filepath.Join(adminDir, "xor.yaml")); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := reopened.Get(Reference{Name: "test/xor"})
	if err != nil || !persisted.Available || !bytes.Equal(persisted.RawYAML, unique) {
		t.Fatalf("administrator snapshot was not retained: %+v, %v", persisted, err)
	}
	if !strings.HasPrefix(string(admin.RawYAML), "---") {
		t.Fatal("raw YAML header was not preserved")
	}
}

func TestAmbiguousReference(t *testing.T) {
	catalog, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	second := changedFixture(t, "version: 1.0.0", "version: 2.0.0")
	if _, err := catalog.ImportAdministrator(second); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Get(Reference{Name: "builtin/xor"}); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("want ambiguous bare name, got %v", err)
	}
	if _, err := catalog.Get(Reference{Name: "builtin/xor", Version: "2.0.0"}); err != nil {
		t.Fatal(err)
	}
}

func TestRemovedAdministratorExecutorLeavesOfflineSnapshot(t *testing.T) {
	root := t.TempDir()
	adminDir := filepath.Join(root, "build-catalog", "admin")
	if err := os.MkdirAll(adminDir, 0700); err != nil {
		t.Fatal(err)
	}
	custom := changedFixture(t, "name: builtin/xor", "name: test/custom")
	custom = replaceFixture(t, custom, "id: xor-v1", "id: custom-v1")
	custom = redigestFixture(t, custom)
	if err := os.WriteFile(filepath.Join(adminDir, "custom.yaml"), custom, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root, nil); err == nil {
		t.Fatal("unknown new administrator executor was accepted")
	}
	installed, err := Open(root, []string{"custom-v1"})
	if err != nil {
		t.Fatal(err)
	}
	online, err := installed.Get(Reference{Name: "test/custom"})
	if err != nil || !online.Available {
		t.Fatalf("custom integration was not available: %+v, %v", online, err)
	}
	removed, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	offline, err := removed.Get(Reference{Name: "test/custom"})
	if err != nil || offline.Available || len(offline.Sources) != 1 || offline.Sources[0].Source != "administrator" {
		t.Fatalf("removed integration did not leave offline snapshot: %+v, %v", offline, err)
	}
}
