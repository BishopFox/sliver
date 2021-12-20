//go:build client

package assets

import "embed"

var (
	//go:embed fs/english.txt fs/sliver.asc
	assetsFs embed.FS
)
