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
	"runtime"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
)

var (
	buildTestLog = log.NamedLogger("generate", "testbuild")
	nonce        = 0
)

func TestSliverExecutableWindows(t *testing.T) {

	// mTLS C2
	mtlsExe(t, "windows", "amd64", false)
	mtlsExe(t, "windows", "386", false)
	mtlsExe(t, "windows", "amd64", true)
	mtlsExe(t, "windows", "386", true)

	// DNS C2
	dnsExe(t, "windows", "amd64", false)
	dnsExe(t, "windows", "amd64", true)

	// HTTP C2
	httpExe(t, "windows", "amd64", false)
	httpExe(t, "windows", "amd64", true)

	// PIVOT TCP C2
	tcpPivotExe(t, "windows", "amd64", false)
	tcpPivotExe(t, "windows", "amd64", true)

	// Named Pipe C2
	namedPipeExe(t, "windows", "amd64", false)
	namedPipeExe(t, "windows", "amd64", true)

	// Multiple C2s
	multiExe(t, "windows", "amd64", true)
	multiExe(t, "windows", "amd64", false)
	multiExe(t, "windows", "386", false)
	multiExe(t, "windows", "386", false)

	// Service
	multiWindowsService(t, "windows", "amd64", true)
	multiWindowsService(t, "windows", "amd64", false)
}

func TestSliverSharedLibWindows(t *testing.T) {
	multiLibrary(t, "windows", "amd64", true)
	multiLibrary(t, "windows", "amd64", false)
	multiLibrary(t, "windows", "386", true)
	multiLibrary(t, "windows", "386", false)
}

func TestSliverExecutableLinux(t *testing.T) {
	multiExe(t, "linux", "amd64", true)
	multiExe(t, "linux", "amd64", false)
	tcpPivotExe(t, "linux", "amd64", false)
}

func TestSliverSharedLibraryLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		multiLibrary(t, "linux", "amd64", true)
		multiLibrary(t, "linux", "amd64", false)
		multiLibrary(t, "linux", "386", true)
		multiLibrary(t, "linux", "386", false)
	}
}

func TestSliverExecutableDarwin(t *testing.T) {
	multiExe(t, "darwin", "amd64", true)
	multiExe(t, "darwin", "amd64", false)
	tcpPivotExe(t, "darwin", "amd64", false)
}

func TestSliverDefaultBuild(t *testing.T) {
	mtlsExe(t, "linux", "arm", true)
	mtlsExe(t, "linux", "arm", false)
	httpExe(t, "freebsd", "amd64", false)
	httpExe(t, "freebsd", "amd64", true)
	dnsExe(t, "plan9", "amd64", false)
	dnsExe(t, "plan9", "amd64", true)
}

func TestSymbolObfuscation(t *testing.T) {

	// Supported platforms
	symbolObfuscation(t, "windows", "amd64")
	symbolObfuscation(t, "linux", "amd64")
	symbolObfuscation(t, "linux", "386")
	symbolObfuscation(t, "darwin", "amd64")
	symbolObfuscation(t, "darwin", "arm64")

	// Test an "unsupported" platform
	symbolObfuscation(t, "freebsd", "amd64")
}

func mtlsExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[mtls] EXE %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []models.ImplantC2{
			{URL: "mtls://1.example.com"},
		},
		MTLSc2Enabled:    true,
		Debug:            debug,
		ObfuscateSymbols: false,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("mtls_test%d", nonce), config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func dnsExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[dns] EXE %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []models.ImplantC2{
			{URL: "dns://3.example.com"},
		},
		DNSc2Enabled:     true,
		Debug:            debug,
		ObfuscateSymbols: false,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("dns_test%d", nonce), config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func httpExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[http] EXE %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []models.ImplantC2{
			{
				Priority: 1,
				URL:      "http://4.example.com",
				Options:  "asdf",
			},
		},
		HTTPc2Enabled:    true,
		Debug:            debug,
		ObfuscateSymbols: false,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("http_test%d", nonce), config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func multiExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[multi] %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,

		C2: []models.ImplantC2{
			{URL: "mtls://1.example.com"},
			{URL: "mtls://2.example.com", Options: "asdf"},
			{URL: "https://3.example.com"},
			{Priority: 3, URL: "dns://4.example.com"},
		},
		MTLSc2Enabled:    true,
		HTTPc2Enabled:    true,
		DNSc2Enabled:     true,
		Debug:            debug,
		ObfuscateSymbols: false,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("multi_test%d", nonce), config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func multiWindowsService(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[multi] %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		Format: clientpb.ImplantConfig_SERVICE,

		C2: []models.ImplantC2{
			{URL: "mtls://1.example.com"},
			{URL: "mtls://2.example.com", Options: "asdf"},
			{URL: "https://3.example.com"},
			{Priority: 3, URL: "dns://4.example.com"},
		},
		MTLSc2Enabled:    true,
		HTTPc2Enabled:    true,
		DNSc2Enabled:     true,
		Debug:            debug,
		ObfuscateSymbols: false,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("service_test%d", nonce), config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func tcpPivotExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[tcppivot] EXE %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []models.ImplantC2{
			{
				Priority: 1,
				URL:      "tcppivot://127.0.0.1:8080",
				Options:  "asdf",
			},
		},
		NamePipec2Enabled: true,
		Debug:             debug,
		ObfuscateSymbols:  false,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("tcpPivot_test%d", nonce), config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func namedPipeExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[namedpipe] EXE %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []models.ImplantC2{
			{
				Priority: 1,
				URL:      "namedpipe://./pipe/test",
				Options:  "asdf",
			},
		},
		NamePipec2Enabled: true,
		Debug:             debug,
		ObfuscateSymbols:  false,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("namedpipe_test%d", nonce), config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func multiLibrary(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[multi] LIB %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,

		C2: []models.ImplantC2{
			{URL: "mtls://1.example.com"},
			{Priority: 2, URL: "mtls://2.example.com"},
			{URL: "https://3.example.com"},
			{URL: "dns://4.example.com", Options: "asdf"},
		},

		Debug:            debug,
		ObfuscateSymbols: false,
		Format:           clientpb.ImplantConfig_SHARED_LIB,
		IsSharedLib:      true,
	}
	nonce++
	_, err := SliverSharedLibrary(fmt.Sprintf("multilibrary_test%d", nonce), config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func symbolObfuscation(t *testing.T, goos string, goarch string) {
	t.Logf("[symbol obfuscation] %s/%s ...", goos, goarch)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,

		C2: []models.ImplantC2{
			{URL: "mtls://1.example.com"},
			{Priority: 2, URL: "mtls://2.example.com"},
			{URL: "https://3.example.com"},
			{URL: "dns://4.example.com", Options: "asdf"},
		},

		Debug:            false,
		ObfuscateSymbols: true,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("symbol_test%d", nonce), config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}
