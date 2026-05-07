package beignet

import _ "embed"

//go:embed internal/loader/beignet_loader.c
var loaderCSource string

// LoaderCSource returns the embedded darwin loader C source code.
func LoaderCSource() string {
	return loaderCSource
}
