//go:build server && !sliver_lint

package assets

import "embed"

var (
	//go:embed fs/*.txt fs/*.zip fs/windows/amd64/*
	assetsFs embed.FS
)
