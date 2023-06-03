//go:build server

package assets

import "embed"

var (
	//go:embed fs/sliver.asc fs/*.txt fs/*.zip fs/darwin/arm64/*
	assetsFs embed.FS
)
