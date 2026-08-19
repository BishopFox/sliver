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
			sha256:   "2d835f2720ca97f0a8c105adad93e791ab52064d6ecd441f4b4637fd9c64c71b",
		},
		{
			os:       "linux",
			arch:     "arm64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_linux-arm64", garbleVersion),
			sha256:   "f8f70f1d55e1b1fa55963face8b0731b7a534eade3f70acde130bc343ba72738",
		},
		{
			os:       "windows",
			arch:     "amd64",
			filename: "garble.exe",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_windows-amd64.exe", garbleVersion),
			sha256:   "11f5b9233abd4e17c878141bf7f9b353116ad46f722b070cb3ae5ca5ef275781",
		},
		{
			os:       "windows",
			arch:     "arm64",
			filename: "garble.exe",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_windows-arm64.exe", garbleVersion),
			sha256:   "0d6c26fa39ae0e178f298ae56f5b27ae16cbe9c449f7b8a6270d00984ac39b64",
		},
		{
			os:       "darwin",
			arch:     "amd64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_darwin-amd64", garbleVersion),
			sha256:   "adb6377da4e935ceb49aef94dd9e9d56dd565affcfe2b28c08fcd402905f2a5a",
		},
		{
			os:       "darwin",
			arch:     "arm64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_darwin-arm64", garbleVersion),
			sha256:   "a046ca07e333f69069e14464e3bb0ae47d41f01de143c0e821ae25ddf69e7d45",
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
