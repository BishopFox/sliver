package table

import (
	"sort"
)

// btoi converts a boolean to an integer, 1 if true, 0 if false.
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// max returns the greater of two integers.
func max(a, b int) int { //nolint:predeclared
	if a > b {
		return a
	}
	return b
}

// min returns the smaller of two integers.
func min(a, b int) int { //nolint:predeclared
	if a < b {
		return a
	}
	return b
}

// sum returns the sum of all integers in a slice.
func sum(n []int) int {
	var sum int
	for _, i := range n {
		sum += i
	}
	return sum
}

// median returns the median of a slice of integers.
func median(n []int) int {
	sort.Ints(n)

	if len(n) <= 0 {
		return 0
	}
	if len(n)%2 == 0 {
		h := len(n) / 2            //nolint:mnd
		return (n[h-1] + n[h]) / 2 //nolint:mnd
	}
	return n[len(n)/2]
}
