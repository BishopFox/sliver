package generate

import (
	"fmt"
	"testing"
)

func TestSliverC2s(t *testing.T) {

	// mTLS C2
	mtls(t, "windows", "amd64", false)
	mtls(t, "windows", "386", false)
	mtls(t, "windows", "amd64", true)
	mtls(t, "windows", "386", true)

	// DNS C2
	dns(t, "windows", "amd64", false)
	dns(t, "windows", "386", false)
	dns(t, "windows", "amd64", true)
	dns(t, "windows", "386", true)

	// HTTP C2
	http(t, "windows", "amd64", false)
	http(t, "windows", "386", false)
	http(t, "windows", "amd64", true)
	http(t, "windows", "386", true)

	// Multiple C2s
	multiC2(t, "windows", "amd64", true)
	multiC2(t, "windows", "amd64", false)
}

func TestSliverExecutableLinux(t *testing.T) {
	multiC2(t, "linux", "amd64", true)
	multiC2(t, "linux", "amd64", false)
}

func TestSliverExecutableDarwin(t *testing.T) {
	multiC2(t, "darwin", "amd64", true)
	multiC2(t, "darwin", "amd64", false)
}

func mtls(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[mtls] %s/%s - debug: %v", goos, goarch, debug)
	config := &SliverConfig{
		GOOS:       goos,
		GOARCH:     goarch,
		MTLSServer: "localhost",
		MTLSLPort:  443,
		Debug:      debug,
	}
	_, err := SliverExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func dns(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[dns] %s/%s - debug: %v", goos, goarch, debug)
	config := &SliverConfig{
		GOOS:      goos,
		GOARCH:    goarch,
		DNSParent: "example.com",
		Debug:     debug,
	}
	_, err := SliverExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func http(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[http] %s/%s - debug: %v", goos, goarch, debug)
	config := &SliverConfig{
		GOOS:       goos,
		GOARCH:     goarch,
		HTTPServer: "example.com",
		Debug:      debug,
	}
	_, err := SliverExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}

func multiC2(t *testing.T, goos string, goarch string, debug bool) {
	t.Logf("[multi] %s/%s - debug: %v", goos, goarch, debug)
	config := &SliverConfig{
		GOOS:   goos,
		GOARCH: goarch,

		MTLSServer: "1.example.com",
		MTLSLPort:  1337,
		HTTPServer: "2.example.com",
		DNSParent:  "3.example.com",
		Debug:      debug,
	}
	_, err := SliverExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}
}
