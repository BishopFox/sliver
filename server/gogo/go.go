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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
	utilAssets "github.com/bishopfox/sliver/util/assets"
)

const (
	goDirName                        = "go"
	garbleExperimentalControlFlowEnv = "GARBLE_EXPERIMENTAL_CONTROLFLOW"
	garbleDebugDirEnv                = "SLIVER_GARBLE_DEBUG_DIR"
	garbleRandomSeedPrefix           = "-seed chosen at random: "
	commandErrorOutputLimit          = 8 * 1024
)

var (
	// ErrGarbleAssetVerification indicates that a control-flow build did not
	// use the exact host Garble artifact pinned by Sliver.
	ErrGarbleAssetVerification = errors.New("garble asset verification failed")
	gogoLog                    = log.NamedLogger("gogo", "compiler")
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

	Obfuscation            bool
	ObfuscationControlFlow bool
	GOGARBLE               string
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

func joinPathList(paths ...string) string {
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		filtered = append(filtered, path)
	}
	return strings.Join(filtered, string(os.PathListSeparator))
}

func goToolExecutableName(name, hostGOOS string) string {
	if hostGOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func goToolPath(goRoot string, toolName string) string {
	return filepath.Join(goRoot, "bin", goToolExecutableName(toolName, runtime.GOOS))
}

func envKeyEqual(first, second string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(first, second)
	}
	return first == second
}

func setCommandEnv(env []string, name, value string) []string {
	updated := make([]string, 0, len(env)+1)
	for _, envVar := range env {
		key, _, _ := strings.Cut(envVar, "=")
		if envKeyEqual(key, name) {
			continue
		}
		updated = append(updated, envVar)
	}
	return append(updated, fmt.Sprintf("%s=%s", name, value))
}

func setGoDebugSetting(settings, name, value string) string {
	updated := make([]string, 0, strings.Count(settings, ",")+1)
	for _, setting := range strings.Split(settings, ",") {
		setting = strings.TrimSpace(setting)
		if setting == "" {
			continue
		}
		key, _, _ := strings.Cut(setting, "=")
		if key == name {
			continue
		}
		updated = append(updated, setting)
	}
	updated = append(updated, name+"="+value)
	return strings.Join(updated, ",")
}

func buildCommandEnv(config GoConfig, extra map[string]string) []string {
	goBinDir := ""
	if strings.TrimSpace(config.GOROOT) != "" {
		goBinDir = filepath.Join(config.GOROOT, "bin")
	}

	env := append([]string{}, os.Environ()...)
	overrides := []struct {
		name  string
		value string
	}{
		{name: "CC", value: config.CC},
		{name: "CGO_ENABLED", value: config.CGO},
		{name: "GOOS", value: config.GOOS},
		{name: "GOARCH", value: config.GOARCH},
		{name: "GOROOT", value: config.GOROOT},
		{name: "GOPATH", value: config.ProjectDir},
		{name: "GOCACHE", value: config.GOCACHE},
		{name: "GOMODCACHE", value: config.GOMODCACHE},
		{name: "GOPROXY", value: config.GOPROXY},
		{name: "HTTP_PROXY", value: config.HTTPPROXY},
		{name: "HTTPS_PROXY", value: config.HTTPSPROXY},
		{name: "PATH", value: joinPathList(goBinDir, assets.GetZigDir(), os.Getenv("PATH"))},
		{name: "HOME", value: getHomeDir()},
	}

	extraNames := make([]string, 0, len(extra))
	for name := range extra {
		extraNames = append(extraNames, name)
	}
	sort.Strings(extraNames)
	for _, name := range extraNames {
		overrides = append(overrides, struct {
			name  string
			value string
		}{name: name, value: extra[name]})
	}

	for _, override := range overrides {
		env = setCommandEnv(env, override.name, override.value)
	}
	return env
}

func garbleCommandEnv(config GoConfig) []string {
	controlFlow := "0"
	if config.ObfuscationControlFlow {
		controlFlow = "1"
	}
	return buildCommandEnv(config, map[string]string{
		garbleExperimentalControlFlowEnv: controlFlow,
		"GOGARBLE":                       config.GOGARBLE,
		"GODEBUG":                        setGoDebugSetting(os.Getenv("GODEBUG"), "randautoseed", "0"),
	})
}

func validateObfuscationConfig(config GoConfig) error {
	if config.ObfuscationControlFlow && !config.Obfuscation {
		return fmt.Errorf("control-flow obfuscation requires symbol obfuscation")
	}
	return nil
}

func garbleRandomSeed(stderr string) string {
	for _, line := range strings.Split(stderr, "\n") {
		if seed, ok := strings.CutPrefix(strings.TrimSpace(line), garbleRandomSeedPrefix); ok {
			return strings.TrimSpace(seed)
		}
	}
	return ""
}

func sanitizeCommandErrorOutput(output string) string {
	output = strings.Map(func(character rune) rune {
		switch character {
		case '\n', '\r', '\t':
			return character
		default:
			if character < 0x20 || character == 0x7f {
				return -1
			}
			return character
		}
	}, strings.TrimSpace(output))
	if len(output) <= commandErrorOutputLimit {
		return output
	}
	headLength := commandErrorOutputLimit / 4
	tailLength := commandErrorOutputLimit - headLength
	return output[:headLength] + "\n... output truncated ...\n" + output[len(output)-tailLength:]
}

func garbleCommandError(err error, stderr string) error {
	details := sanitizeCommandErrorOutput(stderr)
	if details == "" {
		return fmt.Errorf("garble command failed: %w", err)
	}
	return fmt.Errorf("garble command failed: %w: %s", err, details)
}

// GarbleCmd - Execute a go command
func GarbleCmd(config GoConfig, cwd string, command []string) ([]byte, error) {
	if err := validateObfuscationConfig(config); err != nil {
		return nil, err
	}
	target := fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH)
	if _, ok := ValidCompilerTargets(config)[target]; !ok {
		return nil, fmt.Errorf("%s", fmt.Sprintf("Invalid compiler target: %s", target))
	}
	garbleBinPath := goToolPath(config.GOROOT, "garble")
	if config.ObfuscationControlFlow {
		if err := utilAssets.VerifyGarbleBinary(garbleBinPath, runtime.GOOS, runtime.GOARCH); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrGarbleAssetVerification, err)
		}
	}
	garbleFlags := []string{"-seed=random", "-literals", "-tiny"}
	if debugDir := strings.TrimSpace(os.Getenv(garbleDebugDirEnv)); debugDir != "" {
		garbleFlags = append(garbleFlags, "-debugdir="+debugDir)
	}
	command = append(garbleFlags, command...)
	cmd := exec.Command(garbleBinPath, command...)
	cmd.Dir = cwd
	cmd.Env = garbleCommandEnv(config)
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
	if seed := garbleRandomSeed(stderr.String()); seed != "" {
		gogoLog.Infof("garble random seed: %s", seed)
	}
	if err != nil {
		gogoLog.Debugf("--- env ---\n")
		for _, envVar := range cmd.Env {
			gogoLog.Debugf("%s\n", envVar)
		}
		gogoLog.Errorf("--- stdout ---\n%s\n", stdout.String())
		gogoLog.Errorf("--- stderr ---\n%s\n", stderr.String())
		gogoLog.Error(err)
		return stdout.Bytes(), garbleCommandError(err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// GoCmd - Execute a go command
func GoCmd(config GoConfig, cwd string, command []string) ([]byte, error) {
	goBinPath := goToolPath(config.GOROOT, "go")
	cmd := exec.Command(goBinPath, command...)
	cmd.Dir = cwd
	cmd.Env = buildCommandEnv(config, nil)
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
	if err := validateObfuscationConfig(config); err != nil {
		return nil, err
	}
	target := fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH)
	if _, ok := ValidCompilerTargets(config)[target]; !ok {
		return nil, fmt.Errorf("%s", fmt.Sprintf("Invalid compiler target: %s", target))
	}
	var goCommand = []string{"build"}

	goCommand = append(goCommand, "-trimpath") // remove absolute paths from any compiled binary
	goCommand = append(goCommand, "-mod=vendor")

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
