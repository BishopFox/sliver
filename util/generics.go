package util

func Contains[T comparable](elements []T, v T) bool {
	for _, s := range elements {
		if v == s {
			return true
		}
	}
	return false
}
