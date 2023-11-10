package util

func RemoveElement(slice []uint64, value uint64) []uint64 {
	// Create a new slice to hold the result
	result := []uint64{}

	// Iterate through the original slice
	for _, item := range slice {
		if item != value {
			// Append the items that are not equal to the value
			result = append(result, item)
		}
	}

	return result
}
