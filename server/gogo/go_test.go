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
	"os"
	"runtime"
	"strings"
	"testing"
)

func commandEnvValues(env []string, name string) []string {
	values := []string{}
	for _, envVar := range env {
		key, value, found := strings.Cut(envVar, "=")
		if found && envKeyEqual(key, name) {
			values = append(values, value)
		}
	}
	return values
}

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

func TestGarbleCommandEnvOverridesAmbientControlFlow(t *testing.T) {
	tests := []struct {
		name        string
		ambient     string
		controlFlow bool
		want        string
	}{
		{
			name:        "enable overrides ambient disabled",
			ambient:     "0",
			controlFlow: true,
			want:        "1",
		},
		{
			name:        "disable overrides ambient enabled",
			ambient:     "1",
			controlFlow: false,
			want:        "0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(garbleExperimentalControlFlowEnv, test.ambient)
			config := GoConfig{
				Obfuscation:            true,
				ObfuscationControlFlow: test.controlFlow,
			}

			values := commandEnvValues(garbleCommandEnv(config), garbleExperimentalControlFlowEnv)
			if len(values) != 1 {
				t.Fatalf("%s occurs %d times in Garble environment, want exactly once: %q", garbleExperimentalControlFlowEnv, len(values), values)
			}
			if values[0] != test.want {
				t.Fatalf("%s = %q, want %q", garbleExperimentalControlFlowEnv, values[0], test.want)
			}
			if ambient := os.Getenv(garbleExperimentalControlFlowEnv); ambient != test.ambient {
				t.Fatalf("building Garble environment changed process %s to %q, want ambient value %q", garbleExperimentalControlFlowEnv, ambient, test.ambient)
			}
		})
	}
}

func TestGarbleCommandEnvMakesGlobalRandDeterministic(t *testing.T) {
	ambient := "asyncpreemptoff=1,randautoseed=1,panicnil=1,randautoseed=2"
	t.Setenv("GODEBUG", ambient)

	values := commandEnvValues(garbleCommandEnv(GoConfig{}), "GODEBUG")
	if len(values) != 1 {
		t.Fatalf("GODEBUG occurs %d times in Garble environment, want exactly once: %q", len(values), values)
	}
	want := "asyncpreemptoff=1,panicnil=1,randautoseed=0"
	if values[0] != want {
		t.Fatalf("GODEBUG = %q, want %q", values[0], want)
	}
	if got := os.Getenv("GODEBUG"); got != ambient {
		t.Fatalf("building Garble environment changed process GODEBUG to %q, want %q", got, ambient)
	}
}

func TestGoBuildRejectsControlFlowWithoutObfuscation(t *testing.T) {
	config := GoConfig{ObfuscationControlFlow: true}

	_, err := GoBuild(config, "", "", "", nil, nil, "", "")
	if err == nil {
		t.Fatal("GoBuild accepted control-flow obfuscation without symbol obfuscation")
	}
	if !strings.Contains(err.Error(), "control-flow obfuscation requires symbol obfuscation") {
		t.Fatalf("GoBuild returned unexpected error: %v", err)
	}
}

func TestGarbleRandomSeed(t *testing.T) {
	stderr := "-seed chosen at random: AAAAAAAAAAA\nsome other output\n"
	if seed := garbleRandomSeed(stderr); seed != "AAAAAAAAAAA" {
		t.Fatalf("garbleRandomSeed() = %q, want %q", seed, "AAAAAAAAAAA")
	}
	if seed := garbleRandomSeed("no seed here"); seed != "" {
		t.Fatalf("garbleRandomSeed() = %q, want empty", seed)
	}
}

func TestGarbleCommandErrorIsBoundedAndSanitized(t *testing.T) {
	stderr := garbleRandomSeedPrefix + "AAAAAAAAAAA\n" + strings.Repeat("x", commandErrorOutputLimit*2) + "\x00failure detail"
	err := garbleCommandError(os.ErrInvalid, stderr)
	if err == nil {
		t.Fatal("garbleCommandError() returned nil")
	}
	message := err.Error()
	if len(message) > commandErrorOutputLimit+512 {
		t.Fatalf("garbleCommandError() returned %d bytes, expected bounded output", len(message))
	}
	if strings.ContainsRune(message, '\x00') {
		t.Fatal("garbleCommandError() retained a NUL byte")
	}
	if !strings.Contains(message, "AAAAAAAAAAA") || !strings.Contains(message, "failure detail") {
		t.Fatalf("garbleCommandError() lost seed or tail detail: %q", message)
	}
}
