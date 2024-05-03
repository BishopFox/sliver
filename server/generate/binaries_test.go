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
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
)

var (
	nonce = 0
)

func TestSliverExecutableWindows(t *testing.T) {

	// Wireguard C2
	wireguardExe(t, "windows", "amd64", false, false)
	wireguardExe(t, "windows", "arm64", false, false)
	wireguardExe(t, "windows", "386", false, false)
	wireguardExe(t, "windows", "amd64", false, true)
	wireguardExe(t, "windows", "arm64", false, true)
	wireguardExe(t, "windows", "386", false, true)

	// Wireguard beacon
	wireguardExe(t, "windows", "amd64", true, false)
	wireguardExe(t, "windows", "arm64", true, false)
	wireguardExe(t, "windows", "386", true, false)
	wireguardExe(t, "windows", "amd64", true, true)
	wireguardExe(t, "windows", "arm64", true, true)
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
	multiExe(t, "windows", "arm64", false, true)
	multiExe(t, "windows", "arm64", false, false)
	multiExe(t, "windows", "386", false, false)
	multiExe(t, "windows", "386", false, false)

	// Multiple Beacons
	multiExe(t, "windows", "amd64", true, true)
	multiExe(t, "windows", "amd64", true, false)
	multiExe(t, "windows", "arm64", true, true)
	multiExe(t, "windows", "arm64", true, false)
	multiExe(t, "windows", "386", true, false)
	multiExe(t, "windows", "386", true, false)

	// Service
	multiWindowsService(t, "windows", "amd64", false, true)
	multiWindowsService(t, "windows", "amd64", false, false)
}

func TestSliverSharedLibWindows(t *testing.T) {
	// amd64
	multiLibrary(t, "windows", "amd64", true)
	multiLibrary(t, "windows", "amd64", false)

	// arm64
	multiLibrary(t, "windows", "arm64", true)
	multiLibrary(t, "windows", "arm64", false)

	// 386
	multiLibrary(t, "windows", "386", true)
	multiLibrary(t, "windows", "386", false)
}

func TestSliverExecutableLinux(t *testing.T) {
	// amd64
	multiExe(t, "linux", "amd64", false, true)
	multiExe(t, "linux", "amd64", false, false)
	multiExe(t, "linux", "amd64", true, true)
	multiExe(t, "linux", "amd64", true, false)
	tcpPivotExe(t, "linux", "amd64", false)

	// arm64
	multiExe(t, "linux", "arm64", false, true)
	multiExe(t, "linux", "arm64", false, false)
	multiExe(t, "linux", "arm64", true, true)
	multiExe(t, "linux", "arm64", true, false)
	tcpPivotExe(t, "linux", "arm64", false)

	// 386
	multiExe(t, "linux", "386", false, true)
	multiExe(t, "linux", "386", false, false)
	multiExe(t, "linux", "386", true, true)
	multiExe(t, "linux", "386", true, false)
	tcpPivotExe(t, "linux", "386", false)
}

func TestSliverSharedLibraryLinux(t *testing.T) {
	// amd64
	multiLibrary(t, "linux", "amd64", true)
	multiLibrary(t, "linux", "amd64", false)

	// arm64
	multiLibrary(t, "linux", "arm64", true)
	multiLibrary(t, "linux", "arm64", false)

	// 386
	multiLibrary(t, "linux", "386", true)
	multiLibrary(t, "linux", "386", false)
}

func TestSliverExecutableDarwin(t *testing.T) {
	// amd64
	multiExe(t, "darwin", "amd64", false, true)
	multiExe(t, "darwin", "amd64", false, false)
	multiExe(t, "darwin", "amd64", true, true)
	multiExe(t, "darwin", "amd64", true, false)
	tcpPivotExe(t, "darwin", "amd64", false)

	// arm64
	multiExe(t, "darwin", "arm64", false, true)
	multiExe(t, "darwin", "arm64", false, false)
	multiExe(t, "darwin", "arm64", true, true)
	multiExe(t, "darwin", "arm64", true, false)
	tcpPivotExe(t, "darwin", "arm64", false)
}

func TestSliverDefaultBuild(t *testing.T) {
	httpExe(t, "freebsd", "amd64", false, false)
	httpExe(t, "freebsd", "amd64", false, true)
	dnsExe(t, "plan9", "amd64", false, false)
	dnsExe(t, "plan9", "amd64", false, true)
	httpExe(t, "freebsd", "amd64", true, false)
	httpExe(t, "freebsd", "amd64", true, true)
	dnsExe(t, "plan9", "amd64", true, false)
	dnsExe(t, "plan9", "amd64", true, true)
}

func TestSymbolObfuscation(t *testing.T) {
	// Supported platforms
	symbolObfuscation(t, "windows", "amd64")
	symbolObfuscation(t, "windows", "arm64")
	symbolObfuscation(t, "linux", "amd64")
	symbolObfuscation(t, "linux", "arm64")
	symbolObfuscation(t, "linux", "386")
	symbolObfuscation(t, "darwin", "amd64")
	symbolObfuscation(t, "darwin", "arm64")

	// Test an "unsupported" platform
	symbolObfuscation(t, "freebsd", "amd64")
}

func TestTrafficEncoders(t *testing.T) {
	// Supported platforms
	trafficEncodersExecutable(t, "windows", "amd64")
	trafficEncodersExecutable(t, "linux", "amd64")
	trafficEncodersExecutable(t, "linux", "386")
	trafficEncodersExecutable(t, "darwin", "amd64")
	trafficEncodersExecutable(t, "darwin", "arm64")

	// Test an "unsupported" platform
	trafficEncodersExecutable(t, "freebsd", "amd64")
}

func trafficEncodersExecutable(t *testing.T, goos string, goarch string) {
	t.Logf("[trafficEncoders] %s/%s", goos, goarch)
	name := fmt.Sprintf("trafficEncodersDebug_test%d", nonce)
	debugConfig := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []*clientpb.ImplantC2{
			{URL: "http://1.example.com"},
		},
		Debug:                  true,
		ObfuscateSymbols:       false,
		IsBeacon:               false,
		TrafficEncodersEnabled: true,
		IncludeHTTP:            true,
	}

	debugHttpC2Config := configs.GenerateDefaultHTTPC2Config()
	build, _ := GenerateConfig(name, debugConfig)
	nonce++
	_, err := SliverExecutable(name, build, debugConfig, debugHttpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
	name = fmt.Sprintf("trafficEncodersProd_test%d", nonce)
	prodConfig := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []*clientpb.ImplantC2{
			{URL: "http://2.example.com"},
		},
		Debug:                  false,
		ObfuscateSymbols:       true,
		IsBeacon:               false,
		TrafficEncodersEnabled: true,
		IncludeHTTP:            true,
	}
	build, _ = GenerateConfig(name, prodConfig)
	nonce++
	_, err = SliverExecutable(name, build, prodConfig, debugHttpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func mtlsExe(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
	t.Logf("[mtls] EXE %s/%s - debug: %v", goos, goarch, debug)
	name := fmt.Sprintf("mtls_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []*clientpb.ImplantC2{
			{URL: "mtls://1.example.com"},
		},
		Debug:            debug,
		ObfuscateSymbols: false,
		IsBeacon:         beacon,
		IncludeMTLS:      true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	build, _ := GenerateConfig(name, config)
	nonce++
	_, err := SliverExecutable(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func dnsExe(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
	t.Logf("[dns] EXE %s/%s - debug: %v", goos, goarch, debug)
	name := fmt.Sprintf("dns_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []*clientpb.ImplantC2{
			{URL: "dns://3.example.com"},
		},
		Debug:            debug,
		ObfuscateSymbols: false,
		IsBeacon:         beacon,
		IncludeDNS:       true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	build, _ := GenerateConfig(name, config)
	nonce++
	_, err := SliverExecutable(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func httpExe(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
	t.Logf("[http] EXE %s/%s - debug: %v", goos, goarch, debug)
	name := fmt.Sprintf("http_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []*clientpb.ImplantC2{
			{
				Priority: 1,
				URL:      "http://4.example.com",
				Options:  "asdf",
			},
		},
		Debug:            debug,
		ObfuscateSymbols: false,
		IsBeacon:         beacon,
		IncludeHTTP:      true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	build, _ := GenerateConfig(name, config)
	nonce++
	_, err := SliverExecutable(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func multiExe(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
	t.Logf("[multi] %s/%s - debug: %v", goos, goarch, debug)
	name := fmt.Sprintf("multi_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,

		C2: []*clientpb.ImplantC2{
			{URL: "mtls://1.example.com"},
			{URL: "mtls://2.example.com", Options: "asdf"},
			{URL: "https://3.example.com"},
			{Priority: 3, URL: "dns://4.example.com"},
			{Priority: 4, URL: "wg://5.example.com"},
		},
		Debug:            debug,
		ObfuscateSymbols: false,
		IsBeacon:         beacon,
		IncludeMTLS:      true,
		IncludeHTTP:      true,
		IncludeWG:        true,
		IncludeDNS:       true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	build, _ := GenerateConfig(name, config)
	nonce++
	_, err := SliverExecutable(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func multiWindowsService(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
	t.Logf("[multi] %s/%s - debug: %v", goos, goarch, debug)
	name := fmt.Sprintf("service_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		Format: clientpb.OutputFormat_SERVICE,

		C2: []*clientpb.ImplantC2{
			{URL: "mtls://1.example.com"},
			{URL: "mtls://2.example.com", Options: "asdf"},
			{URL: "https://3.example.com"},
			{Priority: 3, URL: "dns://4.example.com"},
		},
		Debug:            debug,
		ObfuscateSymbols: false,
		IsBeacon:         beacon,
		IncludeMTLS:      true,
		IncludeHTTP:      true,
		IncludeDNS:       true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	build, _ := GenerateConfig(name, config)
	nonce++
	_, err := SliverExecutable(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

// Pivots do not support beacon mode
func tcpPivotExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[tcppivot] EXE %s/%s - debug: %v", goos, goarch, debug)
	name := fmt.Sprintf("tcpPivot_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []*clientpb.ImplantC2{
			{
				Priority: 1,
				URL:      "tcppivot://127.0.0.1:8080",
				Options:  "asdf",
			},
		},
		Debug:            debug,
		ObfuscateSymbols: false,
		IncludeTCP:       true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	build, _ := GenerateConfig(name, config)
	nonce++
	_, err := SliverExecutable(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func namedPipeExe(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[namedpipe] EXE %s/%s - debug: %v", goos, goarch, debug)
	name := fmt.Sprintf("namedpipe_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []*clientpb.ImplantC2{
			{
				Priority: 1,
				URL:      "namedpipe://./pipe/test",
				Options:  "asdf",
			},
		},
		Debug:            debug,
		ObfuscateSymbols: false,
		IncludeNamePipe:  true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	build, _ := GenerateConfig(name, config)
	nonce++
	_, err := SliverExecutable(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func wireguardExe(t *testing.T, goos string, goarch string, beacon bool, debug bool) {
	t.Logf("[wireguard] EXE %s/%s - debug: %v", goos, goarch, debug)
	name := fmt.Sprintf("wireguard_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,
		C2: []*clientpb.ImplantC2{
			{
				Priority: 1,
				URL:      "wg://1.example.com:8000",
				Options:  "asdf",
			},
		},
		Debug:             debug,
		ObfuscateSymbols:  false,
		WGPeerTunIP:       "100.64.0.2",
		WGKeyExchangePort: 1234,
		WGTcpCommsPort:    5678,
		IsBeacon:          beacon,
		IncludeWG:         true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	nonce++
	certs.SetupWGKeys()
	build, _ := GenerateConfig(name, config)
	_, err := SliverExecutable(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func multiLibrary(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[multi] LIB %s/%s - debug: %v", goos, goarch, debug)
	name := fmt.Sprintf("multilibrary_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,

		C2: []*clientpb.ImplantC2{
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
		WGPeerTunIP:       "100.64.0.2",
		WGKeyExchangePort: 1234,
		WGTcpCommsPort:    5678,
		IncludeMTLS:       true,
		IncludeHTTP:       true,
		IncludeWG:         true,
		IncludeDNS:        true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	nonce++
	build, _ := GenerateConfig(name, config)
	_, err := SliverSharedLibrary(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func symbolObfuscation(t *testing.T, goos string, goarch string) {
	t.Logf("[symbol obfuscation] %s/%s ...", goos, goarch)
	name := fmt.Sprintf("symbol_test%d", nonce)
	config := &clientpb.ImplantConfig{
		GOOS:   goos,
		GOARCH: goarch,

		C2: []*clientpb.ImplantC2{
			{URL: "mtls://1.example.com"},
			{Priority: 2, URL: "mtls://2.example.com"},
			{URL: "https://3.example.com"},
			{URL: "dns://4.example.com", Options: "asdf"},
		},

		Debug:            false,
		ObfuscateSymbols: true,
		IncludeMTLS:      true,
		IncludeHTTP:      true,
		IncludeDNS:       true,
	}
	httpC2Config := configs.GenerateDefaultHTTPC2Config()
	nonce++
	build, _ := GenerateConfig(name, config)
	_, err := SliverExecutable(name, build, config, httpC2Config.ImplantConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}
}
