//go:build (windows && !arm64) || darwin || linux

package wireguard

import "golang.zx2c4.com/wireguard/conn"

func newWGUDPBind() conn.Bind {
	return conn.NewDefaultBind()
}
