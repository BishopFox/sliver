//go:build windows && (386 || amd64)

package bofloader

func resolveWindowsCRTCompatibility(_, _ string) (uintptr, bool, error) {
	return 0, false, nil
}
