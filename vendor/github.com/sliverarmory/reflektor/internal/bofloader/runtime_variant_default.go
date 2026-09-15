//go:build !(linux && !android && arm)

package bofloader

func validateRuntimeVariant() error {
	return nil
}
