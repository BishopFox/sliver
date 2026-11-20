package generate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// writeRetainedBuildScript adds a human-run Mage command to a retained Sliver
// workspace. It never invokes the script; normal builds use the Mage runner.
func writeRetainedBuildScript(workspacePath, hostOS string) error {
	root, err := os.OpenRoot(workspacePath)
	if err != nil {
		return fmt.Errorf("open retained workspace: %w", err)
	}
	defer root.Close()

	inputsInfo, err := root.Lstat("inputs")
	if err != nil {
		return fmt.Errorf("inspect retained build inputs: %w", err)
	}
	if !inputsInfo.IsDir() {
		return errors.New("retained build inputs are not a directory")
	}
	inputs, err := root.Open("inputs")
	if err != nil {
		return fmt.Errorf("open retained build inputs: %w", err)
	}
	entries, readErr := inputs.ReadDir(-1)
	closeErr := inputs.Close()
	if readErr != nil {
		return fmt.Errorf("list retained build inputs: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close retained build inputs: %w", closeErr)
	}

	var manifest string
	for _, entry := range entries {
		name := entry.Name()
		if !isGoBuildManifestName(name) {
			continue
		}
		info, err := root.Lstat("inputs/" + name)
		if err != nil {
			return fmt.Errorf("inspect retained Mage manifest: %w", err)
		}
		if !info.Mode().IsRegular() {
			return errors.New("retained Mage manifest is not a regular file")
		}
		if manifest != "" {
			return errors.New("retained workspace has multiple Mage manifests")
		}
		manifest = name
	}
	if manifest == "" {
		return errors.New("retained workspace has no Mage manifest")
	}

	name, mode, contents := retainedBuildScript(manifest, hostOS)
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create retained build script: %w", err)
	}
	if _, err := io.WriteString(file, contents); err != nil {
		file.Close()
		root.Remove(name)
		return fmt.Errorf("write retained build script: %w", err)
	}
	if err := file.Chmod(mode); err != nil {
		file.Close()
		root.Remove(name)
		return fmt.Errorf("protect retained build script: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		root.Remove(name)
		return fmt.Errorf("sync retained build script: %w", err)
	}
	if err := file.Close(); err != nil {
		root.Remove(name)
		return fmt.Errorf("close retained build script: %w", err)
	}
	return nil
}

func isGoBuildManifestName(name string) bool {
	const prefix, suffix = "go-build-", ".json"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
		return false
	}
	random := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	if random == "" {
		return false
	}
	for _, digit := range random {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func retainedBuildScript(manifest, hostOS string) (name string, mode os.FileMode, contents string) {
	if hostOS == "windows" {
		return "build.ps1", 0600, fmt.Sprintf(`$ErrorActionPreference = 'Stop'
# Rebuilds the staged core. Sliver's saved build and later transforms are not updated.
$workspace = $PSScriptRoot
& mage -d (Join-Path $workspace 'magefiles') -w $workspace build (Join-Path $workspace 'inputs/%s')
exit $LASTEXITCODE
`, manifest)
	}
	return "build.sh", 0700, fmt.Sprintf(`#!/bin/sh
set -eu
# Rebuilds the staged core. Sliver's saved build and later transforms are not updated.
workspace=$(CDPATH= cd "$(dirname "$0")" && pwd -P)
exec mage -d "$workspace/magefiles" -w "$workspace" build "$workspace/inputs/%s"
`, manifest)
}
