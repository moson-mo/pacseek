package util

import (
	"os"
	"regexp"
)

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

// Shell returns the users default shell
func Shell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh" // fallback
	}
	return shell
}

// MaxLenMapKey returns the length of the longest key string in a map of strings
func MaxLenMapKey(strMap map[string]string) int {
	maxLen := 0
	for k := range strMap {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}
	return maxLen
}

// UniqueStrings merges string slices
func UniqueStrings(strSlices ...[]string) []string {
	uniqueMap := map[string]bool{}

	for _, strSlice := range strSlices {
		for _, str := range strSlice {
			uniqueMap[str] = true
		}
	}

	result := make([]string, 0, len(uniqueMap))

	for key := range uniqueMap {
		result = append(result, key)
	}

	return result
}

// Check if string matches regex
func IsMatch(str string, expr string) bool {
	re, err := regexp.Compile(expr)
	if err != nil || !re.Match([]byte(str)) {
		return false
	}
	return true
}
