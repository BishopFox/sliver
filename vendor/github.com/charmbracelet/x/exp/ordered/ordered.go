package ordered

import "cmp"

// Min returns the smaller of a and b.
func Min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of a and b.
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Clamp returns a value clamped between the given low and high values.
func Clamp[T cmp.Ordered](n, low, high T) T {
	if low > high {
		low, high = high, low
	}
	return Min(high, Max(low, n))
}

// First returns the first non-default value of a fixed number of
// arguments of [cmp.Ordered] types.
func First[T cmp.Ordered](x T, y ...T) T {
	var empty T
	if x != empty {
		return x
	}
	for _, s := range y {
		if s != empty {
			return s
		}
	}
	return empty
}
