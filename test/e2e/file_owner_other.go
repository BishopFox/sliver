//go:build !darwin && !linux

package e2e

import (
	"errors"
	"os"
)

func fileOwnerIDs(os.FileInfo) (string, string, error) {
	return "", "", errors.New("numeric file ownership is unsupported on this platform")
}
