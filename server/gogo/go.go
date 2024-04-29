package gogo

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
)

const (
	goDirName = "go"
)

var (
	gogoLog = log.NamedLogger("gogo", "compiler")
)

// GoConfig - Env variables for Go compiler
type GoConfig struct {
	ProjectDir string

	GOOS       string
	GOARCH     string
	GOROOT     string
	GOCACHE    string
	GOMODCACHE string
	GOPROXY    string
	CGO        string
	CC         string
	CXX        string
	HTTPPROXY  string
	HTTPSPROXY string

	Obfuscation bool
	GOGARBLE    string
}

// GetGoRootDir - Get the path to GOROOT
func GetGoRootDir(appDir string) string {
	return filepath.Join(appDir, goDirName)
}

// GetGoCache - Get the OS temp dir (used for GOCACHE)
func GetGoCache(appDir string) string {
	cachePath := filepath.Join(GetGoRootDir(appDir), "cache")
	os.MkdirAll(cachePath, 0700)
	return cachePath
}

// GetGoModCache - Get the GoMod cache dir
func GetGoModCache(appDir string) string {
	cachePath := filepath.Join(GetGoRootDir(appDir), "modcache")
	os.MkdirAll(cachePath, 0700)
	return cachePath
}

// Garble requires $HOME to be defined, if it's not set we use the os temp dir
func getHomeDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		return os.TempDir()
	}
	return home
}

// GarbleCmd - Execute a go command
func GarbleCmd(config GoConfig, cwd string, command []string) ([]byte, error) {
	target := fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH)
	if _, ok := ValidCompilerTargets(config)[target]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid compiler target: %s", target))
	}
	garbleBinPath := filepath.Join(config.GOROOT, "bin", "garble")
	garbleFlags := []string{"-seed=random", "-literals", "-tiny"}
	command = append(garbleFlags, command...)
	cmd := exec.Command(garbleBinPath, command...)
	cmd.Dir = cwd
	cmd.Env = []string{
		fmt.Sprintf("CC=%s", config.CC),
		fmt.Sprintf("CGO_ENABLED=%s", config.CGO),
		fmt.Sprintf("GOOS=%s", config.GOOS),
		fmt.Sprintf("GOARCH=%s", config.GOARCH),
		fmt.Sprintf("GOPATH=%s", config.ProjectDir),
		fmt.Sprintf("GOCACHE=%s", config.GOCACHE),
		fmt.Sprintf("GOMODCACHE=%s", config.GOMODCACHE),
		fmt.Sprintf("GOPROXY=%s", config.GOPROXY),
		fmt.Sprintf("HTTP_PROXY=%s", config.HTTPPROXY),
		fmt.Sprintf("HTTPS_PROXY=%s", config.HTTPSPROXY),
		fmt.Sprintf("PATH=%s:%s:%s", filepath.Join(config.GOROOT, "bin"), assets.GetZigDir(), os.Getenv("PATH")),
		fmt.Sprintf("GOGARBLE=%s", config.GOGARBLE),
		fmt.Sprintf("HOME=%s", getHomeDir()),
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	gogoLog.Debugf("--- env ---\n")
	for _, envVar := range cmd.Env {
		gogoLog.Debugf("%s\n", envVar)
	}
	gogoLog.Infof("garble cmd: '%v'", cmd)
	err := cmd.Run()
	if err != nil {
		gogoLog.Debugf("--- env ---\n")
		for _, envVar := range cmd.Env {
			gogoLog.Debugf("%s\n", envVar)
		}
		gogoLog.Errorf("--- stdout ---\n%s\n", stdout.String())
		gogoLog.Errorf("--- stderr ---\n%s\n", stderr.String())
		gogoLog.Error(err)
	}

	return stdout.Bytes(), err
}

// GoCmd - Execute a go command
func GoCmd(config GoConfig, cwd string, command []string) ([]byte, error) {
	goBinPath := filepath.Join(config.GOROOT, "bin", "go")
	cmd := exec.Command(goBinPath, command...)
	cmd.Dir = cwd
	cmd.Env = []string{
		fmt.Sprintf("CC=%s", config.CC),
		fmt.Sprintf("CGO_ENABLED=%s", config.CGO),
		fmt.Sprintf("GOOS=%s", config.GOOS),
		fmt.Sprintf("GOARCH=%s", config.GOARCH),
		fmt.Sprintf("GOPATH=%s", config.ProjectDir),
		fmt.Sprintf("GOCACHE=%s", config.GOCACHE),
		fmt.Sprintf("GOMODCACHE=%s", config.GOMODCACHE),
		fmt.Sprintf("GOPROXY=%s", config.GOPROXY),
		fmt.Sprintf("HTTP_PROXY=%s", config.HTTPPROXY),
		fmt.Sprintf("HTTPS_PROXY=%s", config.HTTPSPROXY),
		fmt.Sprintf("PATH=%s:%s:%s", filepath.Join(config.GOROOT, "bin"), assets.GetZigDir(), os.Getenv("PATH")),
		fmt.Sprintf("HOME=%s", getHomeDir()),
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	gogoLog.Infof("go cmd: '%v'", cmd)
	err := cmd.Run()
	if err != nil {
		gogoLog.Infof("--- env ---\n")
		for _, envVar := range cmd.Env {
			gogoLog.Infof("%s\n", envVar)
		}
		gogoLog.Infof("--- stdout ---\n%s\n", stdout.String())
		gogoLog.Infof("--- stderr ---\n%s\n", stderr.String())
		gogoLog.Info(err)
	}

	return stdout.Bytes(), err
}

// GoBuild - Execute a go build command, returns stdout/error
func GoBuild(config GoConfig, src string, dest string, buildmode string, tags []string, ldflags []string, gcflags, asmflags string) ([]byte, error) {
	target := fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH)
	if _, ok := ValidCompilerTargets(config)[target]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid compiler target: %s", target))
	}
	var goCommand = []string{"build"}

	goCommand = append(goCommand, "-trimpath") // remove absolute paths from any compiled binary

	if 0 < len(tags) {
		goCommand = append(goCommand, "-tags")
		goCommand = append(goCommand, tags...)
	}
	if 0 < len(ldflags) {
		goCommand = append(goCommand, "-ldflags")
		goCommand = append(goCommand, ldflags...)
	}
	if 0 < len(gcflags) {
		goCommand = append(goCommand, fmt.Sprintf("-gcflags=%s", gcflags))
	}
	if 0 < len(asmflags) {
		goCommand = append(goCommand, fmt.Sprintf("-asmflags=%s", asmflags))
	}
	if 0 < len(buildmode) {
		goCommand = append(goCommand, fmt.Sprintf("-buildmode=%s", buildmode))
	}
	goCommand = append(goCommand, []string{"-o", dest, "."}...)
	if config.Obfuscation {
		return GarbleCmd(config, src, goCommand)
	}
	return GoCmd(config, src, goCommand)
}

// GoMod - Execute go module commands in src dir
func GoMod(config GoConfig, src string, args []string) ([]byte, error) {
	goCommand := []string{"mod"}
	goCommand = append(goCommand, args...)
	return GoCmd(config, src, goCommand)
}

// GoVersion - Execute a go version command, returns stdout/error
func GoVersion(config GoConfig) ([]byte, error) {
	var goCommand = []string{"version"}
	wd, _ := os.Getwd()
	return GoCmd(config, wd, goCommand)
}

// ValidCompilerTargets - Returns a map of valid compiler targets
func ValidCompilerTargets(config GoConfig) map[string]bool {
	validTargets := make(map[string]bool)
	for _, target := range GoToolDistList(config) {
		validTargets[target] = true
	}
	return validTargets
}

// GoToolDistList - Get a list of supported GOOS/GOARCH pairs
func GoToolDistList(config GoConfig) []string {
	var goCommand = []string{"tool", "dist", "list"}
	wd, _ := os.Getwd()
	data, err := GoCmd(config, wd, goCommand)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	return lines
}
