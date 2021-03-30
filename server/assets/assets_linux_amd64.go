// +build server

package assets

import "embed"

var (
	//go:embed fs/*.txt fs/*.zip fs/dll/*.dll fs/linux/amd64/*
	assetsFs embed.FS
)
