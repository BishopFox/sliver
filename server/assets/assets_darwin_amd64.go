//go:build server

package assets

import "embed"

var (
	//go:embed fs/sliver.asc fs/*.txt fs/*.zip fs/darwin/amd64/* fs/lib/*.a
	assetsFs embed.FS
)
