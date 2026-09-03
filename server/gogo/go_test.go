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
	"runtime"
	"strings"
	"testing"
)

func TestGoToolExecutableName(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		hostGOOS string
		want     string
	}{
		{
			name:     "windows go binary",
			toolName: "go",
			hostGOOS: "windows",
			want:     "go.exe",
		},
		{
			name:     "windows garble binary",
			toolName: "garble",
			hostGOOS: "windows",
			want:     "garble.exe",
		},
		{
			name:     "unix go binary",
			toolName: "go",
			hostGOOS: "linux",
			want:     "go",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := goToolExecutableName(test.toolName, test.hostGOOS); got != test.want {
				t.Fatalf("goToolExecutableName(%q, %q) = %q, want %q", test.toolName, test.hostGOOS, got, test.want)
			}
		})
	}
}

func TestWithDarwinNoCGOLinknameCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		config  GoConfig
		ldflags []string
		want    []string
	}{
		{
			name:   "darwin amd64",
			config: GoConfig{GOOS: "darwin", GOARCH: "amd64", CGO: "0"},
			want:   []string{darwinNoCGOCheckLinknameLDFlag},
		},
		{
			name:    "darwin arm64 preserves existing flags",
			config:  GoConfig{GOOS: "darwin", GOARCH: "arm64", CGO: "0"},
			ldflags: []string{"-s -w"},
			want:    []string{"-s -w " + darwinNoCGOCheckLinknameLDFlag},
		},
		{
			name:    "coalesces multiple linker flag arguments",
			config:  GoConfig{GOOS: "darwin", GOARCH: "amd64", CGO: "0"},
			ldflags: []string{"-s", "-w"},
			want:    []string{"-s -w " + darwinNoCGOCheckLinknameLDFlag},
		},
		{
			name:    "does not duplicate compatibility flag",
			config:  GoConfig{GOOS: "darwin", GOARCH: "amd64", CGO: "0"},
			ldflags: []string{"-s " + darwinNoCGOCheckLinknameLDFlag},
			want:    []string{"-s " + darwinNoCGOCheckLinknameLDFlag},
		},
		{
			name:    "darwin with cgo",
			config:  GoConfig{GOOS: "darwin", GOARCH: "amd64", CGO: "1"},
			ldflags: []string{"-s"},
			want:    []string{"-s"},
		},
		{
			name:    "non-darwin",
			config:  GoConfig{GOOS: "linux", GOARCH: "amd64", CGO: "0"},
			ldflags: []string{"-s"},
			want:    []string{"-s"},
		},
		{
			name:    "unsupported darwin architecture",
			config:  GoConfig{GOOS: "darwin", GOARCH: "386", CGO: "0"},
			ldflags: []string{"-s"},
			want:    []string{"-s"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := append([]string(nil), test.ldflags...)
			got := withDarwinNoCGOLinknameCompatibility(test.config, input)
			if strings.Join(got, "|") != strings.Join(test.want, "|") {
				t.Fatalf("withDarwinNoCGOLinknameCompatibility() = %q, want %q", got, test.want)
			}
			if strings.Join(input, "|") != strings.Join(test.ldflags, "|") {
				t.Fatalf("input ldflags mutated: got %q, want %q", input, test.ldflags)
			}
		})
	}
}

func TestGoVersionUsesHostToolchain(t *testing.T) {
	goRoot := runtime.GOROOT()
	if goRoot == "" {
		t.Fatal("runtime.GOROOT() is empty")
	}
	// The configured GOROOT must win over any inherited toolchain selection.
	// A non-local selection here would fail before invoking the configured go
	// binary, while GOTOOLCHAIN=local keeps host and embedded compilers pinned.
	t.Setenv("GOTOOLCHAIN", "go1.999.0+path")

	config := GoConfig{
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
		GOROOT: goRoot,
	}

	output, err := GoVersion(config)
	if err != nil {
		t.Fatalf("GoVersion failed: %v", err)
	}
	if !strings.Contains(string(output), "go version") {
		t.Fatalf("unexpected go version output: %q", string(output))
	}
}
