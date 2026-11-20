package buildprofile

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func exampleProfile(t *testing.T) Profile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "build-pipeline", "examples", "profile-windows-exe.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func TestBuildProfileCanonicalYAMLRoundTrip(t *testing.T) {
	profile := exampleProfile(t)
	yaml, err := CanonicalYAML(profile)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(profile, parsed) {
		t.Fatalf("canonical YAML changed profile:\n%+v\n%+v", profile, parsed)
	}
	second, err := CanonicalYAML(parsed)
	if err != nil || !bytes.Equal(yaml, second) {
		t.Fatalf("canonical YAML is not stable: %v", err)
	}
}

func TestBuildProfileRejectsUnsafeYAMLAndFields(t *testing.T) {
	yaml, err := os.ReadFile(filepath.Join("..", "..", "docs", "build-pipeline", "examples", "profile-windows-exe.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	base := string(yaml)
	for name, altered := range map[string]string{
		"unknown field":          string(yaml) + "unknownField: true\n",
		"duplicate key":          string(yaml) + "kind: BuildProfile\n",
		"multiple documents":     string(yaml) + "---\nkind: BuildProfile\n",
		"alias":                  strings.Replace(string(yaml), "kind: BuildProfile", "kind: &kind BuildProfile", 1) + "alias: *kind\n",
		"traversal name":         strings.Replace(string(yaml), "name: windows-exe", "name: ../unsafe", 1),
		"invalid builder":        strings.Replace(string(yaml), "policy: automatic", "policy: pinned\n  builder: ../unsafe", 1),
		"invalid input artifact": string(yaml) + "perBuildInputs:\n- name: donor\n  kind: input-artifact\n  artifact:\n    kind: junk\n    container: bogus\n",
		"invalid protocol":       strings.Replace(string(yaml), "protocol: mtls", "protocol: bogus", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if altered == base {
				t.Fatal("negative fixture did not alter YAML")
			}
			if _, err := Parse([]byte(altered)); err == nil {
				t.Fatal("unsafe profile parsed successfully")
			}
		})
	}
}

func TestBuildProfileInputReferenceBinding(t *testing.T) {
	profile := exampleProfile(t)
	profile.CoreOptions[0].ValueJSON = []byte(`{"kind":"secret","name":"missing"}`)
	if err := profile.Validate(); err == nil {
		t.Fatal("unbound secret reference was accepted")
	}
	profile.PerBuildInputs = []Input{{Name: "missing", Kind: "secret"}}
	if err := profile.Validate(); err != nil {
		t.Fatalf("bound secret reference was rejected: %v", err)
	}
	profile.CoreOptions[0].ValueJSON = []byte(`{"custom":"object"}`)
	if err := profile.Validate(); err != nil {
		t.Fatalf("unrelated typed object was rejected: %v", err)
	}
	profile.CoreOptions[0].ValueJSON = []byte(`{"kind":"secret"}`)
	if err := profile.Validate(); err == nil {
		t.Fatal("malformed input reference was accepted")
	}
}

func TestBuildProfileNameLengthBoundary(t *testing.T) {
	profile := exampleProfile(t)
	profile.Metadata.Name = strings.Repeat("a", 255)
	yaml, err := CanonicalYAML(profile)
	if err != nil {
		t.Fatalf("valid maximum-length name was rejected: %v", err)
	}
	if parsed, err := Parse(yaml); err != nil || parsed.Metadata.Name != profile.Metadata.Name {
		t.Fatalf("maximum-length name failed round-trip: %+v, %v", parsed.Metadata, err)
	}
	profile.Metadata.Name += "a"
	if _, err := CanonicalYAML(profile); err == nil {
		t.Fatal("overlong name was accepted")
	}
}

func TestBuildProfileStoreBackfillRevisionAndTombstone(t *testing.T) {
	root := t.TempDir()
	store, err := OpenStore(root)
	if err != nil {
		t.Fatal(err)
	}
	legacy := exampleProfile(t)
	config := []byte("sanitized legacy config")
	first, err := store.UpsertLegacy(legacy, "config-one", config, false)
	if err != nil {
		t.Fatal(err)
	}
	first.Profile.C2.Endpoints[0].Address = "mtls://mutated.invalid:8888"
	first.YAML[0] = 'X'
	first.LegacyConfig[0] = 'X'
	intact, err := store.Get(legacy.Metadata.Name)
	if err != nil || intact.Profile.C2.Endpoints[0].Address == "mtls://mutated.invalid:8888" || intact.YAML[0] == 'X' || intact.LegacyConfig[0] == 'X' {
		t.Fatalf("caller mutation changed stored document: %+v, %v", intact, err)
	}
	first = intact
	legacy.C2.Endpoints[0].Address = "mtls://changed.example.invalid:8888"
	second, err := store.UpsertLegacy(legacy, "config-two", []byte("changed"), false)
	if err != nil || !bytes.Equal(first.YAML, second.YAML) || first.Profile.Metadata.Revision != second.Profile.Metadata.Revision {
		t.Fatalf("idempotent backfill changed stored profile: %+v, %v", second, err)
	}
	second, err = store.UpsertLegacy(legacy, "config-two", []byte("changed"), true)
	if err != nil || second.Profile.Metadata.Revision != 2 {
		t.Fatalf("legacy edit did not advance revision: %+v, %v", second, err)
	}
	secondAgain, err := store.UpsertLegacy(legacy, "config-two", []byte("changed"), true)
	if err != nil || secondAgain.Profile.Metadata.Revision != 2 {
		t.Fatalf("unchanged legacy edit was not idempotent: %+v, %v", secondAgain, err)
	}
	previous := first // An exported old revision cannot overwrite a newer profile.
	if _, err := store.Import(previous.Profile); !errors.Is(err, ErrRevision) {
		t.Fatalf("stale import returned %v", err)
	}
	authored := second.Profile
	authored.Metadata.Revision = 3
	authored.C2.Endpoints[0].Address = "mtls://authored.example.invalid:8888"
	imported, err := store.Import(authored)
	if err != nil || imported.Migrated {
		t.Fatalf("authored import did not supersede migrated definition: %+v, %v", imported, err)
	}
	if _, err := store.UpsertLegacy(legacy, "config-three", []byte("third"), true); err != nil {
		t.Fatal(err)
	}
	current, err := store.Get(authored.Metadata.Name)
	if err != nil || !bytes.Equal(current.YAML, imported.YAML) {
		t.Fatalf("backfill overwrote authored profile: %+v, %v", current, err)
	}
	if err := store.Delete(authored.Metadata.Name); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertLegacy(legacy, "config-four", nil, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted legacy profile was revived: %v", err)
	}
	reopened, err := OpenStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.IsTombstoned(authored.Metadata.Name) {
		t.Fatal("tombstone was not durable")
	}
	if err := reopened.Restore(authored.Metadata.Name, &current); err != nil {
		t.Fatal(err)
	}
	if got, err := reopened.Get(authored.Metadata.Name); err != nil || !bytes.Equal(got.YAML, imported.YAML) {
		t.Fatalf("compensation did not restore profile: %+v, %v", got, err)
	}
}
