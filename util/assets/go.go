package assets

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type goPlatform struct {
	os          string
	arch        string
	archiveExt  string
	includeSrc  bool
	toolRemoves []string
}

var wasmOverlaySourceHashes = map[string]string{
	"src/net/net_fake.go":                    "784b369c57be52fa87ace5f10e07e3c24b45d549639e1487734b793024188df3",
	"src/net/lookup_unix.go":                 "7fc0ecb91aa268d3dbd6dad3b6b806d92223e11782d6fc1b4dcc4f5f9a4b788e",
	"src/net/http/transport_default_wasm.go": "45f994092ea1a8c432f97fc3d4673a251eabe87fd0310d2d20bbb0c3bd637993",
}

func (r *runner) buildGoAssets() error {
	r.logger.Section("Go")

	platforms := []goPlatform{
		{
			os:         "darwin",
			arch:       "amd64",
			archiveExt: "tar.gz",
			includeSrc: true,
			toolRemoves: []string{
				"pkg/tool/darwin_amd64/doc",
				"pkg/tool/darwin_amd64/tour",
				"pkg/tool/darwin_amd64/test2json",
			},
		},
		{
			os:         "darwin",
			arch:       "arm64",
			archiveExt: "tar.gz",
			includeSrc: true,
			toolRemoves: []string{
				"pkg/tool/darwin_arm64/doc",
				"pkg/tool/darwin_arm64/tour",
				"pkg/tool/darwin_arm64/test2json",
			},
		},
		{
			os:         "linux",
			arch:       "amd64",
			archiveExt: "tar.gz",
			toolRemoves: []string{
				"pkg/tool/linux_amd64/doc",
				"pkg/tool/linux_amd64/tour",
				"pkg/tool/linux_amd64/test2json",
			},
		},
		{
			os:         "linux",
			arch:       "arm64",
			archiveExt: "tar.gz",
			toolRemoves: []string{
				"pkg/tool/linux_arm64/doc",
				"pkg/tool/linux_arm64/tour",
				"pkg/tool/linux_arm64/test2json",
			},
		},
		{
			os:         "windows",
			arch:       "amd64",
			archiveExt: "zip",
			toolRemoves: []string{
				"pkg/tool/windows_amd64/doc.exe",
				"pkg/tool/windows_amd64/tour.exe",
				"pkg/tool/windows_amd64/test2json.exe",
			},
		},
		{
			os:         "windows",
			arch:       "arm64",
			archiveExt: "zip",
			toolRemoves: []string{
				"pkg/tool/windows_arm64/doc.exe",
				"pkg/tool/windows_arm64/tour.exe",
				"pkg/tool/windows_arm64/test2json.exe",
			},
		},
	}

	for _, platform := range platforms {
		label := fmt.Sprintf("%s/%s", platform.os, platform.arch)
		r.goIndex++
		r.logger.Logf("Fetch go %s (%d/%d)", label, r.goIndex, goTotal)

		archiveName := fmt.Sprintf("go%s.%s-%s.%s", goVersion, platform.os, platform.arch, platform.archiveExt)
		archiveURL := fmt.Sprintf("https://dl.google.com/go/%s", archiveName)
		archivePath := filepath.Join(r.workDir, archiveName)

		if err := r.downloadFile(archiveURL, archivePath); err != nil {
			return err
		}

		r.logger.Logf("Extract go %s", label)
		goDir := filepath.Join(r.workDir, "go")
		if err := os.RemoveAll(goDir); err != nil {
			return fmt.Errorf("remove previous go dir: %w", err)
		}

		if platform.archiveExt == "zip" {
			if err := extractZip(archivePath, r.workDir); err != nil {
				return err
			}
		} else {
			if err := extractTarGz(archivePath, r.workDir); err != nil {
				return err
			}
		}

		if err := removePaths(goDir, goBloatPaths); err != nil {
			return err
		}
		if err := validateWasmOverlaySources(goDir); err != nil {
			return err
		}

		if platform.includeSrc {
			r.logger.Logf("Pack src.zip (%s)", label)
			srcZipPath := filepath.Join(r.outputDir, "src.zip")
			if err := zipDir(goDir, "src", srcZipPath); err != nil {
				return err
			}
		}

		if err := os.RemoveAll(filepath.Join(goDir, "src")); err != nil {
			return fmt.Errorf("remove src dir: %w", err)
		}

		if err := removePaths(goDir, platform.toolRemoves); err != nil {
			return err
		}
		r.logger.Logf("Build sliver-wasm-go (%s)", label)
		if err := buildWasmGoWrapper(goDir, platform); err != nil {
			return err
		}

		r.logger.Logf("Pack go.zip (%s)", label)
		outputDir := filepath.Join(r.outputDir, platform.os, platform.arch)
		if err := ensureDir(outputDir); err != nil {
			return err
		}
		destZip := filepath.Join(outputDir, "go.zip")
		wrapperPath := "go/bin/" + wasmGoWrapperName(platform.os)
		modeOverrides := map[string]os.FileMode{
			wrapperPath: wasmGoWrapperMode(platform.os),
		}
		if err := zipDirWithModeOverrides(r.workDir, "go", destZip, modeOverrides); err != nil {
			return err
		}
		if err := verifyWasmGoWrapperArchive(destZip, platform); err != nil {
			return err
		}

		if err := os.RemoveAll(goDir); err != nil {
			return fmt.Errorf("remove go dir: %w", err)
		}
		if err := os.Remove(archivePath); err != nil {
			return fmt.Errorf("remove go archive: %w", err)
		}
	}

	return nil
}

func validateWasmOverlaySources(goRoot string) error {
	return validateWasmOverlaySourceHashes(goRoot, wasmOverlaySourceHashes)
}

func validateWasmOverlaySourceHashes(goRoot string, expectedHashes map[string]string) error {
	for relativePath, expectedHash := range expectedHashes {
		path := filepath.Join(goRoot, filepath.FromSlash(relativePath))
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read Go %s overlay target %s: %w", goVersion, relativePath, err)
		}
		sum := sha256.Sum256(data)
		actualHash := hex.EncodeToString(sum[:])
		if actualHash != expectedHash {
			return fmt.Errorf(
				"unsupported Go %s overlay target %s: SHA-256 %s (need %s)",
				goVersion,
				relativePath,
				actualHash,
				expectedHash,
			)
		}
	}
	return nil
}

func buildWasmGoWrapper(goRoot string, platform goPlatform) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}

	hostGo, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("find host Go executable: %w", err)
	}
	outputPath := filepath.Join(goRoot, "bin", wasmGoWrapperName(platform.os))
	cmd := exec.Command(
		hostGo,
		"build",
		"-trimpath",
		"-buildvcs=false",
		"-mod=vendor",
		"-ldflags=-s -w",
		"-o",
		outputPath,
		"./util/cmd/sliver-wasm-go",
	)
	cmd.Dir = repoRoot
	cmd.Env = overrideEnvironment(
		os.Environ(),
		"GOOS="+platform.os,
		"GOARCH="+platform.arch,
		"CGO_ENABLED=0",
		"GOTOOLCHAIN=local",
		"GOWORK=off",
		"GOFLAGS=",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"build sliver-wasm-go for %s/%s: %w: %s",
			platform.os,
			platform.arch,
			err,
			strings.TrimSpace(string(output)),
		)
	}
	if err := os.Chmod(outputPath, wasmGoWrapperMode(platform.os)); err != nil {
		return fmt.Errorf("set sliver-wasm-go permissions: %w", err)
	}
	return nil
}

func overrideEnvironment(environ []string, overrides ...string) []string {
	overrideKeys := make([]string, 0, len(overrides))
	for _, override := range overrides {
		key, _, _ := strings.Cut(override, "=")
		overrideKeys = append(overrideKeys, key)
	}
	result := make([]string, 0, len(environ)+len(overrides))
	for _, entry := range environ {
		key, _, _ := strings.Cut(entry, "=")
		replaced := false
		for _, overrideKey := range overrideKeys {
			if strings.EqualFold(key, overrideKey) {
				replaced = true
				break
			}
		}
		if !replaced {
			result = append(result, entry)
		}
	}
	return append(result, overrides...)
}

func wasmGoWrapperName(goos string) string {
	if goos == "windows" {
		return "sliver-wasm-go.exe"
	}
	return "sliver-wasm-go"
}

func wasmGoWrapperMode(goos string) os.FileMode {
	if goos == "windows" {
		return 0o644
	}
	return 0o755
}

func verifyWasmGoWrapperArchive(archivePath string, platform goPlatform) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open generated Go archive: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	expectedPath := "go/bin/" + wasmGoWrapperName(platform.os)
	for _, file := range reader.File {
		if file.Name != expectedPath {
			continue
		}
		if file.UncompressedSize64 == 0 {
			return fmt.Errorf("%s is empty in %s", expectedPath, archivePath)
		}
		if platform.os != "windows" && file.Mode().Perm()&0o111 == 0 {
			return fmt.Errorf("%s is not executable in %s", expectedPath, archivePath)
		}
		stream, err := file.Open()
		if err != nil {
			return fmt.Errorf("open %s in %s: %w", expectedPath, archivePath, err)
		}
		magic := make([]byte, 4)
		_, readErr := io.ReadFull(stream, magic)
		closeErr := stream.Close()
		if readErr != nil {
			return fmt.Errorf("read %s in %s: %w", expectedPath, archivePath, readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close %s in %s: %w", expectedPath, archivePath, closeErr)
		}
		if !validExecutableMagic(platform.os, magic) {
			return fmt.Errorf("%s has invalid %s executable magic in %s", expectedPath, platform.os, archivePath)
		}
		return nil
	}
	return fmt.Errorf("%s is missing from %s", expectedPath, archivePath)
}

func validExecutableMagic(goos string, magic []byte) bool {
	switch goos {
	case "darwin":
		return len(magic) >= 4 &&
			((magic[0] == 0xcf && magic[1] == 0xfa && magic[2] == 0xed && magic[3] == 0xfe) ||
				(magic[0] == 0xfe && magic[1] == 0xed && magic[2] == 0xfa && magic[3] == 0xcf))
	case "linux":
		return len(magic) >= 4 && string(magic[:4]) == "\x7fELF"
	case "windows":
		return len(magic) >= 2 && string(magic[:2]) == "MZ"
	default:
		return false
	}
}

func removePaths(root string, paths []string) error {
	for _, path := range paths {
		path = trimLeadingDot(path)
		if path == "" {
			continue
		}
		full := filepath.Join(root, path)
		if err := os.RemoveAll(full); err != nil {
			return fmt.Errorf("remove %s: %w", full, err)
		}
	}
	return nil
}
