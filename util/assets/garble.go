package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type garblePlatform struct {
	os       string
	arch     string
	filename string
	url      string
	sha256   string
}

func (r *runner) buildGarbleAssets() error {
	r.logger.Section("Garble")

	for _, platform := range garblePlatforms() {
		r.garbleIndex++
		r.logger.Logf("Fetch garble %s/%s (%d/%d)", platform.os, platform.arch, r.garbleIndex, garbleTotal)
		outputDir := filepath.Join(r.outputDir, platform.os, platform.arch)
		if err := ensureDir(outputDir); err != nil {
			return err
		}
		destPath := filepath.Join(outputDir, platform.filename)
		downloadPath, err := r.downloadToTemp(platform.url, r.workDir)
		if err != nil {
			return err
		}
		if err := verifyGarbleSHA256(downloadPath, platform.sha256); err != nil {
			_ = os.Remove(downloadPath)
			return fmt.Errorf("verify garble %s/%s: %w", platform.os, platform.arch, err)
		}
		if err := moveFile(downloadPath, destPath); err != nil {
			_ = os.Remove(downloadPath)
			return fmt.Errorf("install garble %s/%s: %w", platform.os, platform.arch, err)
		}
	}

	return nil
}

func garblePlatforms() []garblePlatform {
	return []garblePlatform{
		{
			os:       "linux",
			arch:     "amd64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_linux-amd64", garbleVersion),
			sha256:   "ee10c850ec4b4078155e7fb6a28c6dc50ac1c991bbb8029829c40d41c8c57b64",
		},
		{
			os:       "linux",
			arch:     "arm64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_linux-arm64", garbleVersion),
			sha256:   "5503dc9fcb6db3b797f0c10e20078de50df16b1cd8bf00668701589a601ae821",
		},
		{
			os:       "windows",
			arch:     "amd64",
			filename: "garble.exe",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_windows-amd64.exe", garbleVersion),
			sha256:   "c7b979d9d61ecf5aa7de811a8e97a90364eaf03763778a9e1ee3c5a9473366be",
		},
		{
			os:       "windows",
			arch:     "arm64",
			filename: "garble.exe",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_windows-arm64.exe", garbleVersion),
			sha256:   "f1de3caed55ff9422b940108a85ea61a1e868ec3bca01ec51fc5a4163ce2b44d",
		},
		{
			os:       "darwin",
			arch:     "amd64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_darwin-amd64", garbleVersion),
			sha256:   "62c46c30c4983b176db5b46a22256d61fce3b9da647cecb7e7d479bfe47f17b8",
		},
		{
			os:       "darwin",
			arch:     "arm64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_darwin-arm64", garbleVersion),
			sha256:   "63626c1833f94732fb43a2d00dc2236450ecedfe0e51497cfc5a310818c32f05",
		},
	}
}

// ExpectedGarbleSHA256 returns the pinned digest for the Garble artifact used
// by a host platform. The server uses the same manifest at runtime so it does
// not advertise control-flow support for a stale or partially extracted tool.
func ExpectedGarbleSHA256(goos, goarch string) (string, bool) {
	for _, platform := range garblePlatforms() {
		if platform.os == goos && platform.arch == goarch {
			return platform.sha256, true
		}
	}
	return "", false
}

// VerifyGarbleBinary verifies an extracted Garble binary against the artifact
// manifest used when server assets are assembled.
func VerifyGarbleBinary(path, goos, goarch string) error {
	expected, ok := ExpectedGarbleSHA256(goos, goarch)
	if !ok {
		return fmt.Errorf("unsupported garble host platform %s/%s", goos, goarch)
	}
	if err := verifyGarbleSHA256(path, expected); err != nil {
		return fmt.Errorf("verify garble binary for %s/%s: %w", goos, goarch, err)
	}
	return nil
}

func verifyGarbleSHA256(path, expected string) (resultErr error) {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open artifact: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil && resultErr == nil {
			resultErr = fmt.Errorf("close artifact: %w", err)
		}
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("hash artifact: %w", err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, strings.TrimSpace(expected)) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", actual, expected)
	}
	return nil
}
