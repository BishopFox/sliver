//go:build windows && arm64

package wireguard

import "golang.zx2c4.com/wireguard/conn"

// WinRingBind can stop exchanging handshakes after rapid userspace-device
// recreation on Windows ARM64. Beacon callbacks recreate their device at each
// interval, so use wireguard-go's portable socket bind on this target.
func newWGUDPBind() conn.Bind {
	return conn.NewStdNetBind()
}
