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
	"fmt"
	"testing"

	"github.com/bishopfox/sliver/server/assets"
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

func TestGoGoVersion(t *testing.T) {
	appDir := assets.GetRootAppDir()
	winConfig := GoConfig{
		GOOS:   "windows",
		GOARCH: "amd64",
		GOROOT: GetGoRootDir(appDir),
	}
	_, err := GoVersion(winConfig)
	if err != nil {
		t.Errorf("%s", fmt.Sprintf("version cmd failed %v", err))
	}
}
