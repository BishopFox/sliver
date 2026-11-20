//go:build mage

// This is the installed Sliver core's Mage target. It deliberately imports
// only the standard library so its compiled runner can be built offline.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type goConfig struct {
	ProjectDir  string
	GOOS        string
	GOARCH      string
	GOROOT      string
	GOCACHE     string
	GOMODCACHE  string
	GOPROXY     string
	CGO         string
	CC          string
	CXX         string
	HTTPPROXY   string
	HTTPSPROXY  string
	Obfuscation bool
	GOGARBLE    string
}

type goManifest struct {
	Version      int      `json:"version"`
	GoConfig     goConfig `json:"goConfig"`
	MaterialPath string   `json:"materialPath"`
	SourceDir    string   `json:"sourceDir"`
	Destination  string   `json:"destination"`
	BuildMode    string   `json:"buildMode"`
	Tags         []string `json:"tags"`
	LDFlags      []string `json:"ldFlags"`
	GCFlags      string   `json:"gcFlags"`
	ASMFlags     string   `json:"asmFlags"`
	ResultPath   string   `json:"resultPath"`
}

type buildResult struct {
	Primary     string `json:"primary"`
	DisplayName string `json:"displayName"`
}

// Build compiles the staged core and declares its primary output. The path is
// a private, versioned manifest written by the Sliver build executor.
func Build(ctx context.Context, manifestPath string) error {
	manifest, err := readManifest(manifestPath)
	if err != nil {
		return err
	}
	if err := validateManifest(manifest); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	args := []string{"build", "-trimpath", "-mod=vendor"}
	if len(manifest.Tags) > 0 {
		args = append(args, "-tags")
		args = append(args, manifest.Tags...)
	}
	ldflags := darwinNoCGOLinknameCompatibility(manifest.GoConfig, manifest.LDFlags)
	if len(ldflags) > 0 {
		args = append(args, "-ldflags")
		args = append(args, ldflags...)
	}
	if manifest.GCFlags != "" {
		args = append(args, "-gcflags="+manifest.GCFlags)
	}
	if manifest.ASMFlags != "" {
		args = append(args, "-asmflags="+manifest.ASMFlags)
	}
	if manifest.BuildMode != "" {
		args = append(args, "-buildmode="+manifest.BuildMode)
	}
	args = append(args, "-o", manifest.Destination, ".")
	compiler := filepath.Join(manifest.GoConfig.GOROOT, "bin", executableName("go"))
	if manifest.GoConfig.Obfuscation {
		compiler = filepath.Join(manifest.GoConfig.GOROOT, "bin", executableName("garble"))
		args = append([]string{"-seed=random", "-literals", "-literals-shared", "-literals-shared-short", "-tiny"}, args...)
	}
	cmd := exec.CommandContext(ctx, compiler, args...)
	cmd.Dir = manifest.SourceDir
	cmd.Env = compilerEnv(manifest.GoConfig)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("core compiler failed: %w", err)
	}
	info, err := os.Lstat(manifest.Destination)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return errors.New("core compiler did not produce a regular output file")
	}
	return writeResult(manifest.ResultPath, buildResult{
		Primary: filepath.Base(manifest.Destination), DisplayName: filepath.Base(manifest.Destination),
	})
}

func readManifest(path string) (goManifest, error) {
	var manifest goManifest
	file, err := os.Open(path)
	if err != nil {
		return manifest, fmt.Errorf("open core build manifest: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 1<<20+1))
	if err != nil || len(data) > 1<<20 {
		return manifest, errors.New("core build manifest exceeds size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("decode core build manifest: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return manifest, errors.New("core build manifest has trailing data")
	}
	return manifest, nil
}

func validateManifest(manifest goManifest) error {
	if manifest.Version != 1 || manifest.GoConfig.GOROOT == "" || manifest.GoConfig.ProjectDir == "" ||
		manifest.GoConfig.GOOS == "" || manifest.GoConfig.GOARCH == "" || manifest.SourceDir == "" ||
		manifest.Destination == "" || manifest.ResultPath == "" {
		return errors.New("incomplete core build manifest")
	}
	paths := []string{manifest.GoConfig.ProjectDir, manifest.SourceDir, manifest.Destination, manifest.ResultPath}
	if manifest.MaterialPath != "" {
		paths = append(paths, manifest.MaterialPath)
	}
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			return errors.New("core build paths must be absolute")
		}
	}
	project, err := filepath.EvalSymlinks(manifest.GoConfig.ProjectDir)
	if err != nil {
		return fmt.Errorf("resolve core build workspace: %w", err)
	}
	source, err := filepath.EvalSymlinks(manifest.SourceDir)
	if err != nil {
		return fmt.Errorf("resolve core source: %w", err)
	}
	if !inside(project, source) {
		return errors.New("core source is outside build workspace")
	}
	if manifest.MaterialPath != "" {
		if !inside(filepath.Clean(manifest.GoConfig.ProjectDir), manifest.MaterialPath) {
			return errors.New("core material is outside build workspace")
		}
		info, err := os.Lstat(manifest.MaterialPath)
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("core material must be a regular file")
		}
		resolvedMaterial, err := filepath.EvalSymlinks(manifest.MaterialPath)
		if err != nil || !inside(project, resolvedMaterial) {
			return errors.New("core material leaves build workspace")
		}
	}
	bin := filepath.Dir(manifest.Destination)
	if filepath.Base(manifest.Destination) == "." || manifest.ResultPath != filepath.Join(bin, "result.json") ||
		manifest.Destination == manifest.ResultPath || !inside(filepath.Clean(manifest.GoConfig.ProjectDir), bin) {
		return errors.New("core output is outside build workspace")
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		return fmt.Errorf("create core output directory: %w", err)
	}
	resolvedBin, err := filepath.EvalSymlinks(bin)
	if err != nil || !inside(project, resolvedBin) {
		return errors.New("core output directory leaves build workspace")
	}
	for _, output := range []string{manifest.Destination, manifest.ResultPath} {
		if info, err := os.Lstat(output); err == nil {
			if !info.Mode().IsRegular() {
				return errors.New("core output path is not a regular file")
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect core output path: %w", err)
		}
	}
	return nil
}

func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func compilerEnv(config goConfig) []string {
	env := append([]string{}, os.Environ()...)
	path := filepath.Join(config.GOROOT, "bin")
	if inherited := os.Getenv("PATH"); inherited != "" {
		path += string(os.PathListSeparator) + inherited
	}
	home := os.Getenv("HOME")
	if home == "" {
		home = os.TempDir()
	}
	return append(env,
		"CC="+config.CC, "CXX="+config.CXX, "CGO_ENABLED="+config.CGO,
		"GOTOOLCHAIN=local", "GOOS="+config.GOOS, "GOARCH="+config.GOARCH,
		"GOROOT="+config.GOROOT, "GOPATH="+config.ProjectDir,
		"GOCACHE="+config.GOCACHE, "GOMODCACHE="+config.GOMODCACHE,
		"GOPROXY="+config.GOPROXY, "HTTP_PROXY="+config.HTTPPROXY,
		"HTTPS_PROXY="+config.HTTPSPROXY, "PATH="+path, "HOME="+home,
		"GOGARBLE="+config.GOGARBLE,
	)
}

func darwinNoCGOLinknameCompatibility(config goConfig, ldflags []string) []string {
	if config.GOOS != "darwin" || config.CGO != "0" ||
		(config.GOARCH != "amd64" && config.GOARCH != "arm64") {
		return ldflags
	}
	const flag = "-checklinkname=0"
	joined := strings.TrimSpace(strings.Join(ldflags, " "))
	for _, value := range strings.Fields(joined) {
		if value == flag {
			return []string{joined}
		}
	}
	if joined == "" {
		return []string{flag}
	}
	return []string{joined + " " + flag}
}

func writeResult(path string, result buildResult) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".result-*")
	if err != nil {
		return fmt.Errorf("create core result descriptor: %w", err)
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if err := json.NewEncoder(file).Encode(result); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("publish core result descriptor: %w", err)
	}
	return nil
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
