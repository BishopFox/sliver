package assets

import "embed"

var (
	//go:embed traffic-encoders/*.wasm
	DefaultTrafficEncoders embed.FS
)
