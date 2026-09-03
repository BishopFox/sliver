package e2e

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/bishopfox/sliver/client/command/armory"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/util/minisign"
)

func TestValidateSAEnvOutputDoesNotDiscloseOutput(t *testing.T) {
	const rawOutput = "TOP-SECRET-SA-ENV-OUTPUT"
	err := validateSAEnvOutput([]byte(rawOutput), "SLIVER_E2E_MARKER=expected")
	if err == nil {
		t.Fatal("expected output validation to fail")
	}
	if strings.Contains(err.Error(), rawOutput) {
		t.Fatalf("validation error disclosed raw sa-env output: %q", err)
	}
}

func TestValidateSAWhoamiOutput(t *testing.T) {
	valid := []byte("UserName SID GROUP INFORMATION Privilege Name State")
	if err := validateSAWhoamiOutput(valid); err != nil {
		t.Fatalf("valid sa-whoami output rejected: %v", err)
	}

	const rawOutput = "TOP-SECRET-SA-WHOAMI-OUTPUT"
	err := validateSAWhoamiOutput([]byte(rawOutput))
	if err == nil {
		t.Fatal("expected output validation to fail")
	}
	if strings.Contains(err.Error(), rawOutput) {
		t.Fatalf("validation error disclosed raw sa-whoami output: %q", err)
	}
}

func TestDownloadPinnedRetriesTransientHTTPStatus(t *testing.T) {
	synctest.Test(t, testDownloadPinnedRetriesTransientHTTPStatus)
}

func testDownloadPinnedRetriesTransientHTTPStatus(t *testing.T) {
	content := []byte("verified Armory asset")
	digest := sha256.Sum256(content)
	attempts := atomic.Int32{}
	server := httptest.NewTestServer(t, http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		attempt := attempts.Add(1)
		if request.UserAgent() != "sliver-comprehensive-e2e" {
			t.Errorf("User-Agent = %q", request.UserAgent())
		}
		switch attempt {
		case 1:
			response.WriteHeader(http.StatusInternalServerError)
		case 2:
			response.WriteHeader(http.StatusTooManyRequests)
		default:
			_, _ = response.Write(content)
		}
	}))

	got, err := downloadPinned(t.Context(), server.Client(), pinnedAsset{
		url:    server.URL,
		sha256: hex.EncodeToString(digest[:]),
	})
	if err != nil {
		t.Fatalf("downloadPinned() error = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("downloadPinned() = %q, want %q", got, content)
	}
	if gotAttempts := attempts.Load(); gotAttempts != int32(armoryDownloadTries) {
		t.Fatalf("download attempts = %d, want %d", gotAttempts, armoryDownloadTries)
	}
}

func TestDownloadPinnedRetriesTransportError(t *testing.T) {
	synctest.Test(t, testDownloadPinnedRetriesTransportError)
}

func testDownloadPinnedRetriesTransportError(t *testing.T) {
	content := []byte("verified Armory asset")
	digest := sha256.Sum256(content)
	attempts := 0
	client := &http.Client{Transport: armoryRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("temporary network failure")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(content)),
			Request:    request,
		}, nil
	})}

	got, err := downloadPinned(t.Context(), client, pinnedAsset{
		url:    "https://armory.invalid/asset",
		sha256: hex.EncodeToString(digest[:]),
	})
	if err != nil {
		t.Fatalf("downloadPinned() error = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("downloadPinned() = %q, want %q", got, content)
	}
	if attempts != 2 {
		t.Fatalf("download attempts = %d, want 2", attempts)
	}
}

func TestDownloadPinnedDoesNotRetryPermanentFailures(t *testing.T) {
	t.Run("HTTP 404", func(t *testing.T) {
		attempts := atomic.Int32{}
		server := httptest.NewTestServer(t, http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			attempts.Add(1)
			response.WriteHeader(http.StatusNotFound)
		}))

		if _, err := downloadPinned(t.Context(), server.Client(), pinnedAsset{url: server.URL}); err == nil {
			t.Fatal("HTTP 404 unexpectedly succeeded")
		}
		if gotAttempts := attempts.Load(); gotAttempts != 1 {
			t.Fatalf("HTTP 404 attempts = %d, want 1", gotAttempts)
		}
	})

	t.Run("SHA-256 mismatch", func(t *testing.T) {
		attempts := atomic.Int32{}
		server := httptest.NewTestServer(t, http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			attempts.Add(1)
			_, _ = io.WriteString(response, "tampered")
		}))

		if _, err := downloadPinned(t.Context(), server.Client(), pinnedAsset{
			url:    server.URL,
			sha256: strings.Repeat("0", sha256.Size*2),
		}); err == nil {
			t.Fatal("SHA-256 mismatch unexpectedly succeeded")
		}
		if gotAttempts := attempts.Load(); gotAttempts != 1 {
			t.Fatalf("SHA-256 mismatch attempts = %d, want 1", gotAttempts)
		}
	})
}

func TestDownloadSignedPackageDoesNotRetryIntegrityFailures(t *testing.T) {
	t.Run("signature mismatch", func(t *testing.T) {
		archive := []byte("not an archive because signature verification runs first")
		publicKey, privateKey, err := minisign.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		publicKeyText, err := publicKey.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		signature := minisign.SignWithComments(privateKey, []byte("different bytes"), "test", "")
		server, archiveRequests, signatureRequests := armoryAssetServer(t, archive, signature)

		_, _, err = downloadSignedPackage(
			t.Context(), server.Client(),
			&armory.ArmoryPackage{CommandName: "sa-test", PublicKey: string(publicKeyText)},
			pinnedTestAsset(server.URL+"/archive", archive),
			pinnedTestAsset(server.URL+"/signature", signature),
			"v1", "sliverarmory/CS-Situational-Awareness-BOF",
		)
		if err == nil {
			t.Fatal("signature mismatch unexpectedly succeeded")
		}
		if archiveRequests.Load() != 1 || signatureRequests.Load() != 1 {
			t.Fatalf("signature mismatch requests archive=%d signature=%d, want one each", archiveRequests.Load(), signatureRequests.Load())
		}
	})

	t.Run("trusted-comment manifest mismatch", func(t *testing.T) {
		signedManifest := []byte(`{
			"name":"test-package",
			"version":"v1",
			"command_name":"sa-test",
			"extension_author":"test",
			"original_author":"test",
			"repo_url":"https://github.com/sliverarmory/CS-Situational-Awareness-BOF",
			"help":"signed help",
			"depends_on":"coff-loader",
			"entrypoint":"go",
			"files":[{"os":"windows","arch":"amd64","path":"payload/test.x64.o"}]
		}`)
		archiveManifest := bytes.Replace(signedManifest, []byte("signed help"), []byte("archive help"), 1)
		archive := armoryTarGzip(t,
			armoryTarEntry{name: "extension.json", data: archiveManifest},
			armoryTarEntry{name: "payload/test.x64.o", data: []byte("artifact")},
		)
		publicKey, privateKey, err := minisign.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		publicKeyText, err := publicKey.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		trustedComment := base64.StdEncoding.EncodeToString(signedManifest)
		signature := minisign.SignWithComments(privateKey, archive, trustedComment, "")
		server, archiveRequests, signatureRequests := armoryAssetServer(t, archive, signature)

		_, _, err = downloadSignedPackage(
			t.Context(), server.Client(),
			&armory.ArmoryPackage{CommandName: "sa-test", PublicKey: string(publicKeyText)},
			pinnedTestAsset(server.URL+"/archive", archive),
			pinnedTestAsset(server.URL+"/signature", signature),
			"v1", "sliverarmory/CS-Situational-Awareness-BOF",
		)
		if err == nil {
			t.Fatal("trusted-comment manifest mismatch unexpectedly succeeded")
		}
		if archiveRequests.Load() != 1 || signatureRequests.Load() != 1 {
			t.Fatalf("manifest mismatch requests archive=%d signature=%d, want one each", archiveRequests.Load(), signatureRequests.Load())
		}
	})
}

func pinnedTestAsset(url string, content []byte) pinnedAsset {
	digest := sha256.Sum256(content)
	return pinnedAsset{url: url, sha256: hex.EncodeToString(digest[:])}
}

func armoryAssetServer(t *testing.T, archive []byte, signature []byte) (*httptest.Server, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	archiveRequests := &atomic.Int32{}
	signatureRequests := &atomic.Int32{}
	server := httptest.NewTestServer(t, http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/archive":
			archiveRequests.Add(1)
			_, _ = response.Write(archive)
		case "/signature":
			signatureRequests.Add(1)
			_, _ = response.Write(signature)
		default:
			http.NotFound(response, request)
		}
	}))
	return server, archiveRequests, signatureRequests
}

type armoryRoundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip armoryRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestCanonicalGitHubRepository(t *testing.T) {
	accepted := map[string]string{
		"sliverarmory/COFFLoader":                         "sliverarmory/coffloader",
		"https://github.com/sliverarmory/COFFLoader":      "sliverarmory/coffloader",
		"https://GITHUB.COM/SliverArmory/COFFLoader.git/": "sliverarmory/coffloader",
	}
	for input, want := range accepted {
		t.Run(input, func(t *testing.T) {
			got, err := canonicalGitHubRepository(input)
			if err != nil {
				t.Fatalf("canonicalGitHubRepository() error = %v", err)
			}
			if got != want {
				t.Fatalf("canonicalGitHubRepository() = %q, want %q", got, want)
			}
		})
	}

	rejected := []string{
		"http://github.com/sliverarmory/COFFLoader",
		"https://github.com.evil.example/sliverarmory/COFFLoader",
		"https://github.com/attacker/sliverarmory/COFFLoader",
		"https://github.com/sliverarmory/COFFLoader/releases",
		"https://github.com/sliverarmory/COFFLoader?owner=attacker",
		"https://github.com/sliverarmory%2FCOFFLoader",
		"git@github.com:sliverarmory/COFFLoader.git",
		" sliverarmory/COFFLoader",
	}
	for _, input := range rejected {
		t.Run("reject_"+input, func(t *testing.T) {
			if _, err := canonicalGitHubRepository(input); err == nil {
				t.Fatalf("canonicalGitHubRepository(%q) unexpectedly succeeded", input)
			}
		})
	}
}

func TestRequireGitHubRepositoryUsesExactOwnerAndName(t *testing.T) {
	const expected = "sliverarmory/COFFLoader"
	if err := requireGitHubRepository("https://github.com/SliverArmory/COFFLoader.git", expected); err != nil {
		t.Fatalf("exact canonical repository rejected: %v", err)
	}
	for _, actual := range []string{
		"https://github.com/attacker/COFFLoader",
		"https://github.com/sliverarmory/COFFLoader-fork",
		"https://github.com/attacker/sliverarmory/COFFLoader",
	} {
		if err := requireGitHubRepository(actual, expected); err == nil {
			t.Fatalf("repository %q unexpectedly matched %q", actual, expected)
		}
	}
}

func TestFindIndexedPackageRejectsDuplicateMatches(t *testing.T) {
	first := &armory.ArmoryPackage{
		CommandName: "sa-env",
		PublicKey:   situationalPublicKey,
		RepoURL:     "https://github.com/sliverarmory/CS-Situational-Awareness-BOF",
	}
	second := *first
	index := &armory.ArmoryIndex{
		Extensions: []*armory.ArmoryPackage{first},
		Aliases:    []*armory.ArmoryPackage{&second},
	}
	if _, err := findIndexedPackage(index, "sa-env", situationalPublicKey, "sliverarmory/CS-Situational-Awareness-BOF"); err == nil {
		t.Fatal("duplicate matching index packages unexpectedly accepted")
	}
}

func TestExactCommandRejectsDuplicateMatches(t *testing.T) {
	manifest := parseTestManifest(t, `{
		"name":"test-package",
		"commands":[
			{"command_name":"sa-env","help":"test","depends_on":"coff-loader","entrypoint":"go","files":[{"os":"windows","arch":"amd64","path":"first/env.x64.o"}]},
			{"command_name":"sa-env","help":"test","depends_on":"coff-loader","entrypoint":"go","files":[{"os":"windows","arch":"386","path":"second/env.x86.o"}]}
		]
	}`)
	if _, err := exactCommand(manifest, "sa-env", "coff-loader", "go"); err == nil {
		t.Fatal("duplicate command entries unexpectedly accepted")
	}
}

func TestExactTargetFileUsesSignedDeclaredPath(t *testing.T) {
	manifest := parseTestManifest(t, `{
		"name":"env",
		"command_name":"sa-env",
		"version":"v0.0.28",
		"extension_author":"test",
		"original_author":"test",
		"repo_url":"https://github.com/sliverarmory/CS-Situational-Awareness-BOF",
		"help":"test",
		"depends_on":"coff-loader",
		"entrypoint":"go",
		"files":[{"os":"windows","arch":"amd64","path":"payload/env.x64.o"}]
	}`)
	command, err := exactCommand(manifest, "sa-env", "coff-loader", "go")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"decoy/env.x64.o":   []byte("decoy"),
		"payload/env.x64.o": []byte("signed path"),
	}
	got, err := exactTargetFile(command, "windows", "amd64", files)
	if err != nil {
		t.Fatalf("exactTargetFile() error = %v", err)
	}
	if string(got) != "signed path" {
		t.Fatalf("exactTargetFile() = %q, want signed path", got)
	}

	delete(files, "payload/env.x64.o")
	if _, err := exactTargetFile(command, "windows", "amd64", files); err == nil {
		t.Fatal("basename-only decoy unexpectedly satisfied the signed artifact path")
	}
}

func TestExactTargetFileRejectsDuplicateTargetEntries(t *testing.T) {
	manifest := parseTestManifest(t, `{
		"name":"test-package",
		"commands":[{
			"command_name":"sa-env","help":"test","depends_on":"coff-loader","entrypoint":"go",
			"files":[
				{"os":"windows","arch":"amd64","path":"first/env.x64.o"},
				{"os":"windows","arch":"amd64","path":"second/env.x64.o"}
			]
		}]
	}`)
	command, err := exactCommand(manifest, "sa-env", "coff-loader", "go")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exactTargetFile(command, "windows", "amd64", map[string][]byte{}); err == nil {
		t.Fatal("duplicate exact target entries unexpectedly accepted")
	}
}

func TestReadArmoryTarCanonicalPaths(t *testing.T) {
	archive := armoryTarGzip(t,
		armoryTarEntry{name: "./extension.json", data: []byte("manifest")},
		armoryTarEntry{name: "payload/env.x64.o", data: []byte("artifact")},
	)
	files, err := readTarGzip(archive)
	if err != nil {
		t.Fatalf("readTarGzip() error = %v", err)
	}
	if got := string(files["extension.json"]); got != "manifest" {
		t.Fatalf("leading ./ path normalized to %q, want manifest", got)
	}

	duplicates := armoryTarGzip(t,
		armoryTarEntry{name: "./payload/env.x64.o", data: []byte("first")},
		armoryTarEntry{name: "payload/env.x64.o", data: []byte("second")},
	)
	if _, err := readTarGzip(duplicates); err == nil {
		t.Fatal("duplicate canonical archive paths unexpectedly accepted")
	}

	ambiguous := armoryTarGzip(t, armoryTarEntry{name: "payload/../env.x64.o", data: []byte("artifact")})
	if _, err := readTarGzip(ambiguous); err == nil {
		t.Fatal("ambiguous non-canonical archive path unexpectedly accepted")
	}
}

func parseTestManifest(t *testing.T, data string) *extensions.ExtensionManifest {
	t.Helper()
	manifest, err := parseSignedExtensionManifest([]byte(data))
	if err != nil {
		t.Fatalf("parseSignedExtensionManifest() error = %v", err)
	}
	return manifest
}

type armoryTarEntry struct {
	name string
	data []byte
}

func armoryTarGzip(t *testing.T, entries ...armoryTarEntry) []byte {
	t.Helper()
	buffer := bytes.NewBuffer(nil)
	gzipWriter := gzip.NewWriter(buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{Name: entry.name, Mode: 0o600, Size: int64(len(entry.data)), Typeflag: tar.TypeReg}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader(%q): %v", entry.name, err)
		}
		if _, err := tarWriter.Write(entry.data); err != nil {
			t.Fatalf("Write(%q): %v", entry.name, err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buffer.Bytes()
}
