package util

// checks if a slice contains a certain element
func SliceContains[T comparable](values []T, value T) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
