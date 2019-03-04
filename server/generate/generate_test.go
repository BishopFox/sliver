package generate

import (
	"fmt"
	"testing"
)

func TestSliverExecutableWindows64Bit(t *testing.T) {

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

}

func TestSliverExecutableLinux(t *testing.T) {

	// mTLS C2
	mtls(t, "linux", "amd64", false)
	mtls(t, "linux", "386", false)
	mtls(t, "linux", "amd64", true)
	mtls(t, "linux", "386", true)

	// DNS C2
	dns(t, "linux", "amd64", false)
	dns(t, "linux", "386", false)
	dns(t, "linux", "amd64", true)
	dns(t, "linux", "386", true)

}

func TestSliverExecutableDarwin(t *testing.T) {

	// mTLS C2
	mtls(t, "darwin", "amd64", false)
	mtls(t, "darwin", "386", false)
	mtls(t, "darwin", "amd64", true)
	mtls(t, "darwin", "386", true)

	// DNS C2
	dns(t, "darwin", "amd64", false)
	dns(t, "darwin", "386", false)
	dns(t, "darwin", "amd64", true)
	dns(t, "darwin", "386", true)

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
