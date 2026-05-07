package docs

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestAllMatchesEmbeddedMarkdownSources(t *testing.T) {
	dirEntries, err := os.ReadDir("sliver-docs/pages/docs/md")
	if err != nil {
		t.Fatalf("read markdown source directory: %v", err)
	}

	expected := make(map[string]string, len(dirEntries))
	for _, entry := range dirEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		content, err := os.ReadFile(filepath.Join("sliver-docs/pages/docs/md", entry.Name()))
		if err != nil {
			t.Fatalf("read markdown source %q: %v", entry.Name(), err)
		}
		expected[entry.Name()] = string(content)
	}

	embeddedEntries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("read embedded directory: %v", err)
	}
	if len(embeddedEntries) != len(expected) {
		t.Fatalf("embedded entry count = %d, want %d", len(embeddedEntries), len(expected))
	}

	docs, err := All()
	if err != nil {
		t.Fatalf("All(): %v", err)
	}
	if len(docs.Docs) != len(expected) {
		t.Fatalf("All() doc count = %d, want %d", len(docs.Docs), len(expected))
	}

	gotNames := make([]string, 0, len(docs.Docs))
	for _, doc := range docs.Docs {
		filename := doc.Name + ".md"
		gotNames = append(gotNames, doc.Name)

		expectedContent, ok := expected[filename]
		if !ok {
			t.Fatalf("unexpected embedded doc %q", filename)
		}
		if doc.Content != expectedContent {
			t.Fatalf("embedded content mismatch for %q", filename)
		}
	}

	wantNames := make([]string, 0, len(expected))
	for filename := range expected {
		wantNames = append(wantNames, filename[:len(filename)-len(filepath.Ext(filename))])
	}
	sort.Strings(gotNames)
	sort.Strings(wantNames)
	for i := range gotNames {
		if gotNames[i] != wantNames[i] {
			t.Fatalf("embedded names mismatch at index %d: got %q want %q", i, gotNames[i], wantNames[i])
		}
	}
}

func TestReadAcceptsDocNameWithoutSuffix(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("sliver-docs/pages/docs/md", "Getting Started.md"))
	if err != nil {
		t.Fatalf("read source doc: %v", err)
	}

	got, err := Read("Getting Started")
	if err != nil {
		t.Fatalf("Read(): %v", err)
	}
	if got != string(want) {
		t.Fatal("Read() returned unexpected content")
	}
}

func TestAsciinemaFSMatchesEmbeddedSources(t *testing.T) {
	dirEntries, err := os.ReadDir("sliver-docs/public/asciinema")
	if err != nil {
		t.Fatalf("read asciinema source directory: %v", err)
	}

	expected := make(map[string][]byte, len(dirEntries))
	for _, entry := range dirEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".cast" {
			continue
		}

		content, err := os.ReadFile(filepath.Join("sliver-docs/public/asciinema", entry.Name()))
		if err != nil {
			t.Fatalf("read asciinema source %q: %v", entry.Name(), err)
		}
		expected[entry.Name()] = content
	}

	embeddedEntries, err := fs.ReadDir(AsciinemaFS, ".")
	if err != nil {
		t.Fatalf("read embedded asciinema directory: %v", err)
	}
	if len(embeddedEntries) != len(expected) {
		t.Fatalf("embedded asciinema entry count = %d, want %d", len(embeddedEntries), len(expected))
	}

	for filename, want := range expected {
		got, err := fs.ReadFile(AsciinemaFS, filename)
		if err != nil {
			t.Fatalf("read embedded asciinema %q: %v", filename, err)
		}
		if string(got) != string(want) {
			t.Fatalf("embedded asciinema mismatch for %q", filename)
		}
	}
}

func TestReadAsciinemaAcceptsNameWithoutSuffix(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("sliver-docs/public/asciinema", "intro.cast"))
	if err != nil {
		t.Fatalf("read source asciinema: %v", err)
	}

	got, err := ReadAsciinema("intro")
	if err != nil {
		t.Fatalf("ReadAsciinema(): %v", err)
	}
	if string(got) != string(want) {
		t.Fatal("ReadAsciinema() returned unexpected content")
	}
}
