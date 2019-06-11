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
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/bishopfox/sliver/server/log"
)

const (
	goDirName     = "go"
	goPathDirName = "gopath"
)

var (
	gogoLog = log.NamedLogger("gogo", "compiler")

	// ValidCompilerTargets - Supported compiler targets
	ValidCompilerTargets = map[string]bool{
		"darwin/386":    true,
		"darwin/amd64":  true,
		"linux/386":     true,
		"linux/amd64":   true,
		"windows/386":   true,
		"windows/amd64": true,
	}
)

// GoConfig - Env variables for Go compiler
type GoConfig struct {
	GOOS   string
	GOARCH string
	GOROOT string
	GOPATH string
	CGO    string
	CC     string
}

// GetGoRootDir - Get the path to GOROOT
func GetGoRootDir(appDir string) string {
	return path.Join(appDir, goDirName)
}

// GetGoPathDir - Get the path to GOPATH
func GetGoPathDir(appDir string) string {
	return path.Join(appDir, goPathDirName)
}

// GetTempDir - Get the OS temp dir (used for GOCACHE)
func GetTempDir() string {
	dir, _ := ioutil.TempDir("", ".sliver_gocache")
	return dir
}

// GoCmd - Execute a go command
func GoCmd(config GoConfig, cwd string, command []string) ([]byte, error) {
	target := fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH)
	if _, ok := ValidCompilerTargets[target]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid compiler target: %s", target))
	}
	goBinPath := path.Join(config.GOROOT, "bin", "go")
	cmd := exec.Command(goBinPath, command...)
	cmd.Dir = cwd
	cmd.Env = []string{
		fmt.Sprintf("CC=%s", config.CC),
		fmt.Sprintf("CGO_ENABLED=%s", config.CGO),
		fmt.Sprintf("GOOS=%s", config.GOOS),
		fmt.Sprintf("GOARCH=%s", config.GOARCH),
		fmt.Sprintf("GOROOT=%s", config.GOROOT),
		fmt.Sprintf("GOPATH=%s", config.GOPATH),
		fmt.Sprintf("GOCACHE=%s", GetTempDir()),
		fmt.Sprintf("PATH=%s/bin", config.GOROOT),
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	gogoLog.Infof("go cmd: '%v'", cmd)
	err := cmd.Run()
	if err != nil {
		gogoLog.Infof("--- stdout ---\n%s\n", stdout.String())
		gogoLog.Infof("--- stderr ---\n%s\n", stderr.String())
		gogoLog.Info(err)
	}

	return stdout.Bytes(), err
}

// GoBuild - Execute a go build command, returns stdout/error
func GoBuild(config GoConfig, src string, dest string, buildmode string, tags []string, ldflags []string, gcflags, asmflags string) ([]byte, error) {
	var goCommand = []string{"build"}
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
	return GoCmd(config, src, goCommand)
}

// GoVersion - Execute a go version command, returns stdout/error
func GoVersion(config GoConfig) ([]byte, error) {
	var goCommand = []string{"version"}
	wd, _ := os.Getwd()
	return GoCmd(config, wd, goCommand)
}
