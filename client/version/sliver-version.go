package version

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
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	normal = "\033[0m"
	bold   = "\033[1m"

	defaultVersion     = "devel"
	defaultReleasesURL = "https://api.github.com/repos/BishopFox/sliver/releases"
)

var (
	// Version - The semantic version in string form
	Version string

	// GoVersion - Go compiler version
	GoVersion string

	// GitCommit - The commit id at compile time
	GitCommit string

	// GitDirty - Was the commit dirty at compile time
	GitDirty string

	// CompiledAt - When was this binary compiled
	CompiledAt string
)

func init() {
	if GithubReleasesURL == "" {
		GithubReleasesURL = defaultReleasesURL
	}
	applyBuildInfo()
}

func applyBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		ensureDefaults()
		return
	}

	if GoVersion == "" {
		if info.GoVersion != "" {
			GoVersion = info.GoVersion
		} else {
			GoVersion = runtime.Version()
		}
	}

	if Version == "" {
		Version = normalizeVersion(info.Main.Version)
	}

	settings := buildSettings(info.Settings)
	if GitCommit == "" {
		GitCommit = settings["vcs.revision"]
	}
	if GitDirty == "" && settings["vcs.modified"] == "true" {
		GitDirty = "Dirty"
	}
	if CompiledAt == "" {
		if vcsTime := settings["vcs.time"]; vcsTime != "" {
			if compiled, err := time.Parse(time.RFC3339, vcsTime); err == nil {
				CompiledAt = strconv.FormatInt(compiled.Unix(), 10)
			}
		}
	}

	ensureDefaults()
}

func ensureDefaults() {
	if Version == "" {
		Version = defaultVersion
	}
	if GoVersion == "" {
		GoVersion = runtime.Version()
	}
}

func buildSettings(settings []debug.BuildSetting) map[string]string {
	values := make(map[string]string, len(settings))
	for _, setting := range settings {
		values[setting.Key] = setting.Value
	}
	return values
}

func normalizeVersion(version string) string {
	if version == "" || version == "(devel)" {
		return defaultVersion
	}
	if canonical := canonicalSemver(version); canonical != "" {
		return canonical
	}
	return version
}

func canonicalSemver(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || version == "(devel)" {
		return ""
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return semver.Canonical(version)
}

// SemanticVersion - Get the structured sematic version
func SemanticVersion() []int {
	semVer := make([]int, 3)
	canonical := canonicalSemver(Version)
	if canonical == "" {
		return semVer
	}
	base := strings.TrimPrefix(canonical, "v")
	if idx := strings.Index(base, "-"); idx != -1 {
		base = base[:idx]
	}
	parts := strings.Split(base, ".")
	for i := 0; i < 3 && i < len(parts); i++ {
		semVer[i], _ = strconv.Atoi(parts[i])
	}
	return semVer
}

// Compiled - Get time this binary was compiled
func Compiled() (time.Time, error) {
	if CompiledAt == "" {
		return time.Unix(0, 0), errors.New("compiled time not available")
	}
	compiled, err := strconv.ParseInt(CompiledAt, 10, 64)
	if err != nil {
		return time.Unix(0, 0), err
	}
	return time.Unix(compiled, 0), nil
}

// FullVersion - Full version string
func FullVersion() string {
	ver := fmt.Sprintf("%s", Version)
	compiled, err := Compiled()
	if err == nil {
		ver += fmt.Sprintf(" - Compiled %s", compiled.String())
	}
	if GitCommit != "" {
		ver += fmt.Sprintf(" - %s", GitCommit)
	}
	return ver
}
