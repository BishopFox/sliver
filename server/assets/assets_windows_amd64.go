// +build server

package assets

import "embed"

var (
	//go:embed fs/*.txt fs/*.zip fs/dll/*.dll fs/windows/amd64/*
	assetsFs embed.FS
)
