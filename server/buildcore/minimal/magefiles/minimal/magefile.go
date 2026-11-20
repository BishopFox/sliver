//go:build mage

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// Build accepts the same private JSON manifest from the programmatic runner
// and `mage -d magefiles/minimal Build /path/to/inputs/build.json`.
func Build(manifestPath string) error {
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest struct {
		SourceDir    string `json:"sourceDir"`
		BinDir       string `json:"binDir"`
		MaterialPath string `json:"materialPath"`
		GoBinary     string `json:"goBinary"`
		GoRoot       string `json:"goRoot"`
		GoCache      string `json:"goCache"`
		GoTempDir    string `json:"goTempDir"`
		GoModCache   string `json:"goModCache"`
		GOOS         string `json:"goos"`
		GOARCH       string `json:"goarch"`
		OutputName   string `json:"outputName"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return err
	}
	if manifest.SourceDir == "" || manifest.BinDir == "" || manifest.MaterialPath == "" ||
		manifest.GoBinary == "" || manifest.GoRoot == "" || manifest.GoCache == "" || manifest.GoTempDir == "" || manifest.GoModCache == "" ||
		manifest.GOOS == "" || manifest.GOARCH == "" || manifest.OutputName == "" ||
		filepath.Base(manifest.OutputName) != manifest.OutputName || manifest.OutputName == "." || manifest.OutputName == ".." {
		return errors.New("incomplete minimal core manifest")
	}
	material, err := os.ReadFile(manifest.MaterialPath)
	if err != nil {
		return err
	}
	if !json.Valid(material) {
		return errors.New("invalid core material")
	}
	// Keep the embedded configuration in a generated companion file. The
	// installed source tree itself is never passed through text/template.
	generated := []byte("package main\n\nfunc init() { configJSON = " + strconv.Quote(string(material)) + " }\n")
	if err := os.WriteFile(filepath.Join(manifest.SourceDir, "config_gen.go"), generated, 0600); err != nil {
		return err
	}
	output := filepath.Join(manifest.BinDir, manifest.OutputName)
	cmd := exec.Command(manifest.GoBinary, "build", "-trimpath", "-mod=vendor", "-o", output, ".")
	cmd.Dir = manifest.SourceDir
	cmd.Env = []string{
		"GOROOT=" + manifest.GoRoot, "GOCACHE=" + manifest.GoCache,
		"GOTMPDIR=" + manifest.GoTempDir, "TMPDIR=" + manifest.GoTempDir,
		"GOMODCACHE=" + manifest.GoModCache, "GOPROXY=off", "CGO_ENABLED=0",
		"GOOS=" + manifest.GOOS, "GOARCH=" + manifest.GOARCH,
		"PATH=" + filepath.Dir(manifest.GoBinary),
	}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compile minimal core: %w", err)
	}
	result, err := json.Marshal(struct {
		Primary     string `json:"primary"`
		DisplayName string `json:"displayName"`
	}{Primary: manifest.OutputName, DisplayName: manifest.OutputName})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(manifest.BinDir, "result.json"), result, 0600)
}
