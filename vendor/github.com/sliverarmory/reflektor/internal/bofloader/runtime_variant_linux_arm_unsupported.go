//go:build linux && !android && arm && !arm.7

package bofloader

import "fmt"

func validateRuntimeVariant() error {
	return fmt.Errorf("bofloader: linux/arm BOFs require GOARM=7 hard-float")
}
