package util

// SliceContains checks if a slice contains a certain element
func SliceContains[T comparable](values []T, value T) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// IndexOf returns an elements position in a slice
func IndexOf[T comparable](values []T, value T) int {
	for i, v := range values {
		if v == value {
			return i
		}
	}
	return -1
}
