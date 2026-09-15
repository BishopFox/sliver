//go:build !windows || !arm64

package c2

import "golang.zx2c4.com/wireguard/conn"

func newWGUDPBind() conn.Bind {
	return conn.NewDefaultBind()
}
