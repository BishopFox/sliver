//go:build amd64 || 386

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
	//{{if .Config.Debug}}
	"log"
	//{{end}}
	"os"
	"strings"
	"syscall"
)

func getString(input [65]int8) string {
	var buf [65]byte
	for i, b := range input {
		buf[i] = byte(b)
	}
	ver := string(buf[:])
	if i := strings.Index(ver, "\x00"); i != -1 {
		ver = ver[:i]
	}
	return ver
}

func readOSRelease() string {
	paths := []string{"/etc/os-release", "/usr/lib/os-release"}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return buildOSRelease(parseOSRelease(string(data)))
	}
	return ""
}

func parseOSRelease(content string) map[string]string {
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
		value = unescape(unquote(strings.TrimSpace(value)))
		values[key] = value
	}
	return values
}

func unquote(s string) string {
	if len(s) < 2 {
		return s
	}
	if (s[0] == '"' || s[0] == '\'') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return s
}

func unescape(s string) string {
	return strings.NewReplacer(
		`\$`, `$`,
		`\"`, `"`,
		`\'`, `'`,
		`\\`, `\`,
		"\\`", "`",
	).Replace(s)
}

func buildOSRelease(values map[string]string) string {
	name := values["NAME"]
	version := values["VERSION"]
	if version == "" {
		version = values["VERSION_ID"]
	}
	if name != "" && version != "" {
		return fmt.Sprintf("%s %s", name, version)
	}
	return values["PRETTY_NAME"]
}

func normalizeArch(arch string) string {
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

func formatDetailedVersion(osRelease, kernel, arch string) string {
	parts := make([]string, 0, 3)
	if osRelease != "" {
		parts = append(parts, osRelease)
	}
	if kernel != "" {
		parts = append(parts, fmt.Sprintf("kernel %s", kernel))
	}
	if arch != "" {
		parts = append(parts, arch)
	}
	return strings.Join(parts, " ")
}

// GetVersion returns the os version information
func GetVersion() string {
	osRelease := readOSRelease()
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		//{{if .Config.Debug}}
		log.Printf("error getting OS version: %v", err)
		//{{end}}
		return osRelease
	}
	if osRelease == "" {
		osRelease = getString(uname.Sysname)
	}
	kernel := getString(uname.Release)
	arch := normalizeArch(getString(uname.Machine))
	return formatDetailedVersion(osRelease, kernel, arch)
}
