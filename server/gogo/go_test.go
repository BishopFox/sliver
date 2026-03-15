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

func TestGoVersionUsesHostToolchain(t *testing.T) {
	goRoot := runtime.GOROOT()
	if goRoot == "" {
		t.Fatal("runtime.GOROOT() is empty")
	}

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
