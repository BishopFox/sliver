//go:build linux && !android && arm && arm.7

package bofloader

import (
	"fmt"
	"runtime/debug"
)

func validateRuntimeVariant() error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("bofloader: linux/arm BOFs require GOARM=7 hard-float; executable build settings are unavailable")
	}
	for _, setting := range info.Settings {
		if setting.Key != "GOARM" {
			continue
		}
		if setting.Value == "7" || setting.Value == "7,hardfloat" {
			return nil
		}
		return fmt.Errorf("bofloader: linux/arm BOFs require GOARM=7 hard-float; executable uses GOARM=%s", setting.Value)
	}
	return fmt.Errorf("bofloader: linux/arm BOFs require GOARM=7 hard-float; executable does not record GOARM")
}
