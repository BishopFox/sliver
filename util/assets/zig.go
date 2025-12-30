package assets

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	insecureRand "math/rand"

	minisign "github.com/bishopfox/sliver/util/minisign"
)

type zigPlatform struct {
	os         string
	arch       string
	remoteName string
	localName  string
}

func (r *runner) buildZigAssets() error {
	r.logger.Section("Zig")

	if err := r.loadZigMirrors(); err != nil {
		return err
	}

	platforms := []zigPlatform{
		{os: "darwin", arch: "amd64", remoteName: fmt.Sprintf("zig-x86_64-macos-%s.tar.xz", zigVersion), localName: "zig.tar.xz"},
		{os: "darwin", arch: "arm64", remoteName: fmt.Sprintf("zig-aarch64-macos-%s.tar.xz", zigVersion), localName: "zig.tar.xz"},
		{os: "linux", arch: "amd64", remoteName: fmt.Sprintf("zig-x86_64-linux-%s.tar.xz", zigVersion), localName: "zig.tar.xz"},
		{os: "linux", arch: "arm64", remoteName: fmt.Sprintf("zig-aarch64-linux-%s.tar.xz", zigVersion), localName: "zig.tar.xz"},
		{os: "windows", arch: "amd64", remoteName: fmt.Sprintf("zig-x86_64-windows-%s.zip", zigVersion), localName: "zig.zip"},
	}

	for _, platform := range platforms {
		if err := r.downloadZig(platform); err != nil {
			return err
		}
	}

	return nil
}

func (r *runner) loadZigMirrors() error {
	if len(r.zigMirrors) > 0 {
		return nil
	}

	client := *r.httpClient
	client.Timeout = 30 * time.Second
	resp, err := client.Get("https://ziglang.org/download/community-mirrors.txt")
	if err == nil && resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			mirrors, err := parseZigMirrors(resp.Body)
			if err == nil && len(mirrors) > 0 {
				r.zigMirrors = mirrors
				return nil
			}
		}
	}

	r.zigMirrors = append([]string{}, defaultZigMirrors...)
	return nil
}

func parseZigMirrors(reader io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(reader)
	mirrors := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.IndexRune(line, '#'); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		mirrors = append(mirrors, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return mirrors, nil
}

func (r *runner) downloadZig(platform zigPlatform) error {
	outputDir := filepath.Join(r.outputDir, platform.os, platform.arch)
	if err := ensureDir(outputDir); err != nil {
		return err
	}
	destPath := filepath.Join(outputDir, platform.localName)
	if err := os.RemoveAll(destPath); err != nil {
		return fmt.Errorf("remove existing zig file: %w", err)
	}

	r.zigIndex++
	mirrors := r.randomizedMirrors()
	for idx, mirror := range mirrors {
		mirrorBase := strings.TrimRight(mirror, "/") + "/" + zigVersion
		artifactURL := appendQueryParam(mirrorBase+"/"+platform.remoteName, zigSourceParam)
		signatureURL := appendQueryParam(mirrorBase+"/"+platform.remoteName+".minisig", zigSourceParam)

		r.logger.Logf("Fetch zig %s/%s (%d/%d) via %s", platform.os, platform.arch, r.zigIndex, zigTotal, mirrorBase)
		r.logger.VLogf("  artifact:  %s", artifactURL)
		r.logger.VLogf("  signature: %s", signatureURL)

		artifactPath, err := r.downloadToTemp(artifactURL, r.workDir)
		if err != nil {
			r.logger.Errorf("Failed to download Zig artifact from %s", artifactURL)
			if idx < len(mirrors)-1 {
				r.logger.Warnf("Trying alternate mirror")
			}
			continue
		}
		signaturePath, err := r.downloadToTemp(signatureURL, r.workDir)
		if err != nil {
			_ = os.Remove(artifactPath)
			r.logger.Errorf("Failed to download Zig signature from %s", signatureURL)
			if idx < len(mirrors)-1 {
				r.logger.Warnf("Trying alternate mirror")
			}
			continue
		}

		if err := verifyZigSignature(artifactPath, signaturePath, platform.remoteName); err != nil {
			r.logger.Errorf("Signature verification failed for %s from %s", platform.remoteName, mirrorBase)
			r.logger.Errorf("Deleting corrupted download %s", artifactPath)
			_ = os.Remove(artifactPath)
			_ = os.Remove(signaturePath)
			if idx < len(mirrors)-1 {
				r.logger.Warnf("Trying alternate mirror")
			}
			continue
		}

		if err := moveFile(artifactPath, destPath); err != nil {
			_ = os.Remove(artifactPath)
			_ = os.Remove(signaturePath)
			return err
		}
		_ = os.Remove(signaturePath)
		r.logger.Successf("Verified zig package -> %s", destPath)
		return nil
	}

	return fmt.Errorf("unable to download and verify Zig package %s", platform.remoteName)
}

func (r *runner) randomizedMirrors() []string {
	mirrors := append([]string{}, r.zigMirrors...)
	if len(mirrors) == 0 {
		return mirrors
	}
	insecureRand.Seed(time.Now().UnixNano())
	insecureRand.Shuffle(len(mirrors), func(i, j int) {
		mirrors[i], mirrors[j] = mirrors[j], mirrors[i]
	})
	return mirrors
}

func appendQueryParam(url, param string) string {
	if strings.Contains(url, "?") {
		return url + "&" + param
	}
	return url + "?" + param
}

func verifyZigSignature(artifactPath, signaturePath, expectedFile string) error {
	sigData, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("read signature: %w", err)
	}
	pubKey, err := loadZigPublicKey()
	if err != nil {
		return err
	}

	artifact, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("open artifact: %w", err)
	}
	defer artifact.Close()

	reader := minisign.NewReader(artifact)
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("stream artifact: %w", err)
	}

	if !reader.Verify(pubKey, sigData) {
		return errors.New("signature verification failed")
	}

	if err := verifyTrustedComment(sigData, expectedFile); err != nil {
		return err
	}

	return nil
}

func loadZigPublicKey() (minisign.PublicKey, error) {
	key := os.Getenv("ZIG_PUBLIC_KEY")
	if key == "" {
		key = zigMinisignPublicKey
	}

	var publicKey minisign.PublicKey
	if err := publicKey.UnmarshalText([]byte(key)); err != nil {
		return minisign.PublicKey{}, fmt.Errorf("invalid minisign public key: %w", err)
	}

	return publicKey, nil
}

func verifyTrustedComment(signature []byte, expectedFile string) error {
	var sig minisign.Signature
	if err := sig.UnmarshalText(signature); err != nil {
		return fmt.Errorf("parse signature: %w", err)
	}

	fileField := trustedCommentFile(sig.TrustedComment)
	if fileField == "" {
		return fmt.Errorf("trusted comment missing file field: %q", sig.TrustedComment)
	}
	if fileField != expectedFile {
		return fmt.Errorf("trusted comment file mismatch: expected %q, got %q", expectedFile, fileField)
	}

	return nil
}

func trustedCommentFile(trustedComment string) string {
	for _, field := range strings.Fields(trustedComment) {
		if after, ok := strings.CutPrefix(field, "file:"); ok {
			return after
		}
	}
	return ""
}
