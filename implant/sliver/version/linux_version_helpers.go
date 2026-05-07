package version

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"fmt"
	"os"
	"strings"
)

type linuxVersionInfo struct {
	Sysname string
	Release string
	Version string
	Machine string
}

func readLinuxOSRelease() string {
	paths := []string{"/etc/os-release", "/usr/lib/os-release"}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if osRelease := buildLinuxOSRelease(parseLinuxOSRelease(string(data))); osRelease != "" {
			return osRelease
		}
	}
	return ""
}

func parseLinuxOSRelease(content string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			continue
		}
		key = strings.TrimSpace(key)
		value = unescapeLinuxOSReleaseValue(unquoteLinuxOSReleaseValue(strings.TrimSpace(value)))
		values[key] = value
	}
	return values
}

func unquoteLinuxOSReleaseValue(s string) string {
	if len(s) < 2 {
		return s
	}
	if (s[0] == '"' || s[0] == '\'') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return s
}

func unescapeLinuxOSReleaseValue(s string) string {
	return strings.NewReplacer(
		`\$`, `$`,
		`\"`, `"`,
		`\'`, `'`,
		`\\`, `\`,
		"\\`", "`",
	).Replace(s)
}

func buildLinuxOSRelease(values map[string]string) string {
	name := values["NAME"]
	version := values["VERSION"]
	if version == "" {
		version = values["VERSION_ID"]
	}
	if name != "" && version != "" {
		return fmt.Sprintf("%s %s", name, version)
	}
	if values["PRETTY_NAME"] != "" {
		return values["PRETTY_NAME"]
	}
	return name
}

func normalizeLinuxArch(arch string) string {
	arch = strings.TrimSpace(arch)
	if arch == "" {
		return ""
	}
	switch strings.ToLower(arch) {
	case "amd64":
		return "x86_64"
	case "i386", "i486", "i586", "i686", "x86":
		return "x86"
	default:
		return arch
	}
}

func formatLinuxDetailedVersion(osRelease string, versionInfo linuxVersionInfo) string {
	parts := make([]string, 0, 3)
	if osRelease != "" {
		parts = append(parts, osRelease)
	} else if versionInfo.Sysname != "" {
		parts = append(parts, versionInfo.Sysname)
	}

	kernelParts := make([]string, 0, 2)
	if versionInfo.Release != "" {
		kernelParts = append(kernelParts, versionInfo.Release)
	}
	if versionInfo.Version != "" {
		kernelParts = append(kernelParts, versionInfo.Version)
	}
	if len(kernelParts) > 0 {
		parts = append(parts, fmt.Sprintf("kernel %s", strings.Join(kernelParts, " ")))
	}

	if arch := normalizeLinuxArch(versionInfo.Machine); arch != "" {
		parts = append(parts, arch)
	}
	return strings.Join(parts, " ")
}
