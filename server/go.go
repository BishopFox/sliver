package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
)

const (
	goDirName     = "go"
	goPathDirName = "gopath"
)

var (
	validCompilerTargets = map[string]bool{
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
}

// GetGoRootDir - Get the path to GOROOT
func GetGoRootDir() string {
	appDir := GetRootAppDir()
	return path.Join(appDir, goDirName)
}

// GetGoPathDir - Get the path to GOPATH
func GetGoPathDir() string {
	appDir := GetRootAppDir()
	return path.Join(appDir, goPathDirName)
}

// GoCmd - Execute a go command
func GoCmd(config GoConfig, cwd string, command []string) ([]byte, error) {
	target := fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH)
	if _, ok := validCompilerTargets[target]; !ok {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid compiler target: %s", target))
	}
	goBinPath := path.Join(config.GOROOT, "bin", "go")
	cmd := exec.Command(goBinPath, command...)
	cmd.Dir = cwd
	cmd.Env = []string{
		"CGO_ENABLED=0",
		fmt.Sprintf("GOOS=%s", config.GOOS),
		fmt.Sprintf("GOARCH=%s", config.GOARCH),
		fmt.Sprintf("GOROOT=%s", config.GOROOT),
		fmt.Sprintf("GOPATH=%s", config.GOPATH),
		fmt.Sprintf("PATH=%s/bin", config.GOROOT),
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("go cmd: '%v'", cmd)
	err := cmd.Run()
	if err != nil {
		log.Printf("--- stdout ---\n%s\n", stdout.String())
		log.Printf("--- stderr ---\n%s\n", stderr.String())
		log.Print(err)
	}

	return stdout.Bytes(), err
}

// GoBuild - Execute a go build command, returns stdout/error
func GoBuild(config GoConfig, src string, dest string, tags []string, ldflags []string) ([]byte, error) {
	var goCommand = []string{"build"}
	if 0 < len(tags) {
		goCommand = append(goCommand, "-tags")
		goCommand = append(goCommand, tags...)
	}
	if 0 < len(ldflags) {
		goCommand = append(goCommand, "-ldflags")
		goCommand = append(goCommand, ldflags...)
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
