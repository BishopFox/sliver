//go:build server

package assets

import "embed"

var (
	//go:embed fs/*.txt fs/*.zip fs/windows/arm64/*
	assetsFs embed.FS
)
