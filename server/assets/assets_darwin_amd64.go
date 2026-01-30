//go:build server

package assets

import "embed"

var (
	//go:embed fs/*.txt fs/*.zip fs/darwin/amd64/*
	assetsFs embed.FS
)
