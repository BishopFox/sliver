// Package gogo tests Sliver's Go and Garble compiler integration.
package gogo

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/server/assets"
	utilAssets "github.com/bishopfox/sliver/util/assets"
)

func TestPinnedGarbleControlFlow(t *testing.T) {
	garblePath := pinnedGarblePath(t)
	moduleDir := controlFlowProbeModule(t)

	transformedRuns := make([]map[string]string, 0, 2)
	for run := 1; run <= 2; run++ {
		runID := fmt.Sprintf("run-%d", run)
		transformedRuns = append(transformedRuns, runControlFlowProbe(t, moduleDir, garblePath, runID))
	}

	if !reflect.DeepEqual(transformedRuns[0], transformedRuns[1]) {
		t.Fatal("fixed Garble seed produced different xor-hardened control-flow output across independent clean caches")
	}
}

func pinnedGarblePath(t *testing.T) string {
	t.Helper()
	if _, ok := utilAssets.ExpectedGarbleSHA256(runtime.GOOS, runtime.GOARCH); !ok {
		t.Skipf("no pinned Garble asset for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	garblePath := goToolPath(GetGoRootDir(assets.GetRootAppDir()), "garble")
	if _, err := os.Stat(garblePath); err != nil {
		t.Fatalf("pinned Garble asset is unavailable at %s: %v", garblePath, err)
	}
	if err := utilAssets.VerifyGarbleBinary(garblePath, runtime.GOOS, runtime.GOARCH); err != nil {
		t.Fatalf("verify pinned Garble asset at %s: %v", garblePath, err)
	}
	return garblePath
}

func controlFlowProbeModule(t *testing.T) string {
	t.Helper()
	moduleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module example.com/sliver-control-flow-probe\n\ngo 1.26.0\n"), 0o600); err != nil {
		t.Fatalf("write probe go.mod: %v", err)
	}
	probeSource := `package main

import "fmt"

//garble:controlflow block_splits=2 junk_jumps=2 flatten_passes=1 flatten_hardening=xor trash_blocks=0
func classify(value int) string {
	if value%2 == 0 {
		return "even"
	}
	return "odd"
}

func main() {
	fmt.Print(classify(42))
}
`
	if err := os.WriteFile(filepath.Join(moduleDir, "main.go"), []byte(probeSource), 0o600); err != nil {
		t.Fatalf("write probe source: %v", err)
	}
	return moduleDir
}

func runControlFlowProbe(t *testing.T, moduleDir string, garblePath string, runID string) map[string]string {
	t.Helper()
	debugDir := filepath.Join(moduleDir, "debug-"+runID)
	outputPath := filepath.Join(moduleDir, "probe-"+runID)
	if runtime.GOOS == "windows" {
		outputPath += ".exe"
	}
	config := GoConfig{
		ProjectDir:             moduleDir,
		GOOS:                   runtime.GOOS,
		GOARCH:                 runtime.GOARCH,
		GOROOT:                 GetGoRootDir(assets.GetRootAppDir()),
		GOCACHE:                filepath.Join(moduleDir, "go-cache-"+runID),
		GOMODCACHE:             filepath.Join(moduleDir, "go-mod-cache-"+runID),
		GOPROXY:                "off",
		CGO:                    "0",
		Obfuscation:            true,
		ObfuscationControlFlow: true,
		GOGARBLE:               "*",
	}
	cmd := exec.Command(
		garblePath,
		"-seed=AAAAAAAAAAA",
		"-literals",
		"-tiny",
		"-debugdir="+debugDir,
		"build",
		"-o", outputPath,
		".",
	)
	cmd.Dir = moduleDir
	cmd.Env = garbleCommandEnv(config)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build control-flow probe %s with pinned Garble: %v\n%s", runID, err, output)
	}

	output, err := exec.Command(outputPath).CombinedOutput()
	if err != nil {
		t.Fatalf("run control-flow probe %s: %v\n%s", runID, err, output)
	}
	if string(output) != "even" {
		t.Fatalf("control-flow probe %s output = %q, want %q", runID, output, "even")
	}
	return transformedControlFlowSources(t, debugDir, runID)
}

func transformedControlFlowSources(t *testing.T, debugDir string, runID string) map[string]string {
	t.Helper()
	transformed := map[string]string{}
	err := filepath.WalkDir(debugDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "GARBLE_controlflow.go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(data), "goto ") {
			return nil
		}
		relativePath, err := filepath.Rel(debugDir, path)
		if err != nil {
			return err
		}
		transformed[relativePath] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("inspect Garble debug output for %s: %v", runID, err)
	}
	if len(transformed) == 0 {
		t.Fatalf("Garble debug output for %s did not contain a transformed GARBLE_controlflow.go", runID)
	}
	return transformed
}
