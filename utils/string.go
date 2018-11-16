package utils

import "strings"

// IncludesPrefix checks if needle is in haystack
func Includes(haystack []string, needle string) bool {
	for _, name := range haystack {
		if name == needle {
			return true
		}
	}
	return false
}

// IncludesPrefix checks if any item in haystack is a prefix of needle
func IncludesPrefix(haystack []string, needle string) bool {
	for _, prefix := range haystack {
		if strings.HasPrefix(needle, prefix) {
			return true
		}
	}
	return false
}

// Remove removes val from slice
func Remove(s []string, val string) []string {
	for i, v := range s {
		if v == val {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

// JSONPointerPath builds a JSON pointer path according to spec, see
// https://tools.ietf.org/html/draft-ietf-appsawg-json-pointer-07#section-3.
func JSONPointerPath(s string) string {
	pointer := strings.Replace(s, "~", "~0", -1)
	return strings.Replace(pointer, "/", "~1", -1)
}
