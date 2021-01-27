package implant

import "embed"

var (
	// FS - Embedded FS access to sliver implant code
	//go:embed sliver/**
	FS embed.FS
)
