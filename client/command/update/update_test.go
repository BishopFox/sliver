package update

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/util/minisign"
)

func TestDownloadAssetWithSignaturePreservesExistingArtifactOnFailure(t *testing.T) {
	const assetName = "sliver-client_linux-amd64.zip"
	oldContent := []byte("previous verified download")
	newContent := []byte("unverified replacement")

	for _, tc := range []struct {
		name            string
		signatureStatus int
		signatureBody   string
		wantError       string
	}{
		{name: "signature download fails", signatureStatus: http.StatusNotFound, wantError: "download signature"},
		{name: "signature is malformed", signatureStatus: http.StatusOK, signatureBody: "not a minisign signature", wantError: "signature verification failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/" + assetName:
					_, _ = w.Write(newContent)
				case "/" + assetName + ".minisig":
					w.WriteHeader(tc.signatureStatus)
					_, _ = w.Write([]byte(tc.signatureBody))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			dir := t.TempDir()
			finalPath := filepath.Join(dir, assetName)
			if err := os.WriteFile(finalPath, oldContent, 0o600); err != nil {
				t.Fatal(err)
			}
			asset := &version.Asset{
				Name:               assetName,
				Size:               len(newContent),
				BrowserDownloadURL: server.URL + "/" + assetName,
			}
			err := downloadAssetWithSignature(nil, server.Client(), asset, nil, dir, minisign.PublicKey{})
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("downloadAssetWithSignature error = %v, want %q", err, tc.wantError)
			}
			got, err := os.ReadFile(finalPath)
			if err != nil {
				t.Fatalf("previous artifact was removed: %v", err)
			}
			if !bytes.Equal(got, oldContent) {
				t.Fatalf("previous artifact changed: got %q, want %q", got, oldContent)
			}
			tempPaths, err := filepath.Glob(filepath.Join(dir, assetName+".tmp-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(tempPaths) != 0 {
				t.Fatalf("temporary downloads remain: %v", tempPaths)
			}
		})
	}
}

func TestReplaceFileMissingSourcePreservesDestination(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "download.zip")
	want := []byte("previous verified download")
	if err := os.WriteFile(dst, want, 0o600); err != nil {
		t.Fatal(err)
	}

	err := replaceFile(filepath.Join(dir, "missing.tmp"), dst)
	if err == nil {
		t.Fatal("replaceFile with a missing source succeeded")
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("previous artifact was removed: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("previous artifact changed: got %q, want %q", got, want)
	}
}

func TestReplaceFileReplacesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "download.zip.tmp")
	dst := filepath.Join(dir, "download.zip")
	want := []byte("new verified download")
	if err := os.WriteFile(src, want, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("previous verified download"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := replaceFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("replacement content = %q, want %q", got, want)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source still exists after replacement: %v", err)
	}
}
