//go:build windows && arm64

package c2

import "golang.zx2c4.com/wireguard/conn"

// Use the portable socket bind on Windows ARM64 so the listener continues to
// accept handshakes when short-lived beacon devices change their UDP endpoint.
func newWGUDPBind() conn.Bind {
	return conn.NewStdNetBind()
}
