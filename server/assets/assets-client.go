// +build client

package assets

import "embed"

var (
	//go:embed fs/english.txt
	assetsFs embed.FS
)
