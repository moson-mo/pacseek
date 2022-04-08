package pacseek

// checks if a slice of strings contains a string
func stringSliceContains(slice []string, item string) bool {
	for _, str := range slice {
		if str == item {
			return true
		}
	}
	return false
}
