// +build client

package assets

import "embed"

var (
	//go:embed fs/empty.txt
	assetsFs embed.FS
)
