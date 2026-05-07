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
	"encoding/xml"
	"errors"
	"fmt"
	//{{if .Config.Debug}}
	"log"
	//{{end}}
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

type plist struct {
	XMLName xml.Name `xml:"plist"`
	Dict    dict     `xml:"dict"`
}

type dict struct {
	Key    []string `xml:"key"`
	String []string `xml:"string"`
}

func getString(input []byte) string {
	ver := string(input)
	if i := strings.Index(ver, "\x00"); i != -1 {
		ver = ver[:i]
	}
	return ver
}

func readOSRelease() string {
	paths := []string{
		"/System/Library/CoreServices/SystemVersion.plist",
		"/System/Library/CoreServices/ServerVersion.plist",
	}
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		properties, err := parsePlistFile(file)
		_ = file.Close()
		if err != nil {
			//{{if .Config.Debug}}
			log.Printf("error parsing %s: %v", path, err)
			//{{end}}
			continue
		}
		if osRelease := buildOSRelease(properties); osRelease != "" {
			return osRelease
		}
	}
	return ""
}

func parsePlistFile(file *os.File) (map[string]string, error) {
	var v plist
	if err := xml.NewDecoder(file).Decode(&v); err != nil {
		return nil, err
	}
	if len(v.Dict.Key) != len(v.Dict.String) {
		return nil, errors.New("invalid plist content")
	}
	properties := make(map[string]string, len(v.Dict.Key))
	for i, key := range v.Dict.Key {
		properties[key] = v.Dict.String[i]
	}
	return properties, nil
}

func buildOSRelease(properties map[string]string) string {
	productName := properties["ProductName"]
	productVersion := properties["ProductVersion"]
	productBuildVersion := properties["ProductBuildVersion"]
	if productName == "" || productVersion == "" || productBuildVersion == "" {
		return ""
	}
	return fmt.Sprintf("%s %s (%s)", productName, productVersion, productBuildVersion)
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

func GetVersion() string {
	osRelease := readOSRelease()
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		//{{if .Config.Debug}}
		log.Printf("error getting OS version: %v", err)
		//{{end}}
		return osRelease
	}
	if osRelease == "" {
		osRelease = getString(uname.Sysname[:])
	}
	kernel := getString(uname.Release[:])
	arch := normalizeArch(getString(uname.Machine[:]))
	return formatDetailedVersion(osRelease, kernel, arch)
}
