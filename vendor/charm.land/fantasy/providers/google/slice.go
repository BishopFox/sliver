package google

func depointerSlice[T any](s []*T) []T {
	result := make([]T, 0, len(s))
	for _, v := range s {
		if v != nil {
			result = append(result, *v)
		}
	}
	return result
}
