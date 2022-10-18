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
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/db/models"
)

const (
	otpTestSecret = "12345678901234567890"
)

var (
	nonce = 0
)

func TestSliverExecutableWindows(t *testing.T) {

	// Wireguard C2
	wireguardExe(t, "windows", "amd64", false, false)
	wireguardExe(t, "windows", "386", false, false)
	wireguardExe(t, "windows", "amd64", false, true)
	wireguardExe(t, "windows", "386", false, true)
	// Wireguard beacon
	wireguardExe(t, "windows", "amd64", true, false)
	wireguardExe(t, "windows", "386", true, false)
	wireguardExe(t, "windows", "amd64", true, true)
	wireguardExe(t, "windows", "386", true, true)

	// mTLS C2
	mtlsExe(t, "windows", "amd64", false, false)
	mtlsExe(t, "windows", "386", false, false)
	mtlsExe(t, "windows", "amd64", false, true)
	mtlsExe(t, "windows", "386", false, true)
	// mTLS Beacon
	mtlsExe(t, "windows", "amd64", true, false)
	mtlsExe(t, "windows", "386", true, false)
	mtlsExe(t, "windows", "amd64", true, true)
	mtlsExe(t, "windows", "386", true, true)

	// DNS C2
	dnsExe(t, "windows", "amd64", false, false)
	dnsExe(t, "windows", "amd64", false, true)
	// DNS Beacon
	dnsExe(t, "windows", "amd64", true, false)
	dnsExe(t, "windows", "amd64", true, true)

	// HTTP C2
	httpExe(t, "windows", "amd64", false, false)
	httpExe(t, "windows", "amd64", false, true)
	// HTTP Beacon
	httpExe(t, "windows", "amd64", true, false)
	httpExe(t, "windows", "amd64", true, true)

	// PIVOT TCP C2
	tcpPivotExe(t, "windows", "amd64", false)
	tcpPivotExe(t, "windows", "amd64", true)

	// Named Pipe C2
	namedPipeExe(t, "windows", "amd64", false)
	namedPipeExe(t, "windows", "amd64", true)

	// Multiple C2s
	multiExe(t, "windows", "amd64", false, true)
	multiExe(t, "windows", "amd64", false, false)
	multiExe(t, "windows", "386", false, false)
	multiExe(t, "windows", "386", false, false)

	// Multiple Beacons
	multiExe(t, "windows", "amd64", true, true)
	multiExe(t, "windows", "amd64", true, false)
	multiExe(t, "windows", "386", true, false)
	multiExe(t, "windows", "386", true, false)

	// Service
	multiWindowsService(t, "windows", "amd64", false, true)
	multiWindowsService(t, "windows", "amd64", false, false)
}

func TestSliverSharedLibWindows(t *testing.T) {
	multiLibrary(t, "windows", "amd64", true)
	multiLibrary(t, "windows", "amd64", false)
	multiLibrary(t, "windows", "386", true)
	multiLibrary(t, "windows", "386", false)
}

func TestSliverExecutableLinux(t *testing.T) {
	multiExe(t, "linux", "amd64", false, true)
	multiExe(t, "linux", "amd64", false, false)

	multiExe(t, "linux", "amd64", true, true)
	multiExe(t, "linux", "amd64", true, false)

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
	multiExe(t, "darwin", "amd64", false, true)
	multiExe(t, "darwin", "amd64", false, false)
	multiExe(t, "darwin", "amd64", true, true)
	multiExe(t, "darwin", "amd64", true, false)

	tcpPivotExe(t, "darwin", "amd64", false)
}

func TestSliverDefaultBuild(t *testing.T) {
	mtlsExe(t, "linux", "arm", false, true)
	mtlsExe(t, "linux", "arm", false, false)
	httpExe(t, "freebsd", "amd64", false, false)
	httpExe(t, "freebsd", "amd64", false, true)
	dnsExe(t, "plan9", "amd64", false, false)
	dnsExe(t, "plan9", "amd64", false, true)

	mtlsExe(t, "linux", "arm", true, true)
	mtlsExe(t, "linux", "arm", true, false)
	httpExe(t, "freebsd", "amd64", true, false)
	httpExe(t, "freebsd", "amd64", true, true)
	dnsExe(t, "plan9", "amd64", true, false)
	dnsExe(t, "plan9", "amd64", true, true)
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

func mtlsExe(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
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
		IsBeacon:         beacon,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("mtls_test%d", nonce), otpTestSecret, config, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func dnsExe(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
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
		IsBeacon:         beacon,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("dns_test%d", nonce), otpTestSecret, config, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func httpExe(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
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
		IsBeacon:         beacon,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("http_test%d", nonce), otpTestSecret, config, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func multiExe(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
	t.Logf("[multi] %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,

		C2: []models.ImplantC2{
			{URL: "mtls://1.example.com"},
			{URL: "mtls://2.example.com", Options: "asdf"},
			{URL: "https://3.example.com"},
			{Priority: 3, URL: "dns://4.example.com"},
			{Priority: 4, URL: "wg://5.example.com"},
		},
		MTLSc2Enabled:    true,
		HTTPc2Enabled:    true,
		DNSc2Enabled:     true,
		WGc2Enabled:      true,
		Debug:            debug,
		ObfuscateSymbols: false,
		IsBeacon:         beacon,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("multi_test%d", nonce), otpTestSecret, config, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func multiWindowsService(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
	t.Logf("[multi] %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		Format: clientpb.OutputFormat_SERVICE,

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
		IsBeacon:         beacon,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("service_test%d", nonce), otpTestSecret, config, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

// Pivots do not support beacon mode
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
	_, err := SliverExecutable(fmt.Sprintf("tcpPivot_test%d", nonce), otpTestSecret, config, true)
	if err != nil {
		t.Fatalf("%v", err)
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
	_, err := SliverExecutable(fmt.Sprintf("namedpipe_test%d", nonce), otpTestSecret, config, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func wireguardExe(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
	t.Logf("[wireguard] EXE %s/%s - debug: %v", goos, goarch, debug)
	config := &models.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []models.ImplantC2{
			{
				Priority: 1,
				URL:      "wg://1.example.com:8000",
				Options:  "asdf",
			},
		},
		WGc2Enabled:       true,
		Debug:             debug,
		ObfuscateSymbols:  false,
		WGImplantPrivKey:  "153be871d7e54545c01a9700880f86fc83087275669c9237b9bcd617ddbfa43f",
		WGServerPubKey:    "153be871d7e54545c01a9700880f86fc83087275669c9237b9bcd617ddbfa43f",
		WGPeerTunIP:       "100.64.0.2",
		WGKeyExchangePort: 1234,
		WGTcpCommsPort:    5678,
		IsBeacon:          beacon,
	}
	nonce++
	certs.SetupWGKeys()
	_, err := SliverExecutable(fmt.Sprintf("wireguard_test%d", nonce), otpTestSecret, config, true)
	if err != nil {
		t.Fatalf("%v", err)
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
			{URL: "wg://5.example.com", Options: "asdf"},
		},

		Debug:             debug,
		ObfuscateSymbols:  false,
		Format:            clientpb.OutputFormat_SHARED_LIB,
		IsSharedLib:       true,
		WGc2Enabled:       true,
		WGImplantPrivKey:  "153be871d7e54545c01a9700880f86fc83087275669c9237b9bcd617ddbfa43f",
		WGServerPubKey:    "153be871d7e54545c01a9700880f86fc83087275669c9237b9bcd617ddbfa43f",
		WGPeerTunIP:       "100.64.0.2",
		WGKeyExchangePort: 1234,
		WGTcpCommsPort:    5678,
	}
	nonce++
	_, err := SliverSharedLibrary(fmt.Sprintf("multilibrary_test%d", nonce), otpTestSecret, config, true)
	if err != nil {
		t.Fatalf("%v", err)
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
		MTLSc2Enabled: true,
		HTTPc2Enabled: true,
		DNSc2Enabled:  true,

		Debug:            false,
		ObfuscateSymbols: true,
	}
	nonce++
	_, err := SliverExecutable(fmt.Sprintf("symbol_test%d", nonce), otpTestSecret, config, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
}
