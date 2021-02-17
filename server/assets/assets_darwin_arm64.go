// +build server

package assets

import "embed"

var (
	//go:embed fs/*.txt fs/*.zip fs/dll/*.dll fs/darwin/arm64/go.zip
	assetsFs embed.FS
)
