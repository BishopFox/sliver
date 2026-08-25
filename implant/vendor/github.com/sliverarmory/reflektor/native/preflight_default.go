//go:build !darwin || ios

package native

import (
	"bytes"
	"debug/buildinfo"
)

func imageHasGoBuildInfo(data []byte) bool {
	_, err := buildinfo.Read(bytes.NewReader(data))
	return err == nil
}
