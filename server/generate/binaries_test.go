package generate

import (
	"fmt"
	"testing"
)

func TestSliverExecutableWindows64Bit(t *testing.T) {

	t.Log("[mtls] windows/amd64")
	config := &SliverConfig{
		GOOS:       WINDOWS,
		GOARCH:     "amd64",
		MTLSServer: "localhost",
	}
	_, err := SliverExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}

	t.Log("[mtls] windows/amd64 - debug")
	config.Debug = true
	_, err = SliverExecutable(config)
	if err != nil {
		t.Errorf(fmt.Sprintf("%v", err))
	}

}

func TestSliverExecutableLinux(t *testing.T) {

}

func TestSliverExecutableDarwin(t *testing.T) {

}
