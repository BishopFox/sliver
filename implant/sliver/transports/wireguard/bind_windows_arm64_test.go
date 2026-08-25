//go:build windows && arm64

package wireguard

import (
	"testing"

	"golang.zx2c4.com/wireguard/conn"
)

func TestWindowsARM64WGUDPBindUsesStandardSockets(t *testing.T) {
	if _, ok := newWGUDPBind().(*conn.StdNetBind); !ok {
		t.Fatal("Windows ARM64 WireGuard bind did not use standard sockets")
	}
}
