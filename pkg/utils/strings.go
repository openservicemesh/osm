package utils

import "strings"

// GetLastNOfDotted splits a string by period and returns the last N chunks.
func GetLastNOfDotted(s string, n int) string {
	split := strings.Split(s, ".")
	var pieces []string
	startAt := len(split) - n
	for i := startAt; i < len(split); i++ {
		pieces = append(pieces, split[i])
	}
	if len(pieces) == 0 {
		return s
	}
	return strings.Join(pieces, ".")
}

// GetFirstNOfDotted splits a string by period and returns the first chunk.
func GetFirstNOfDotted(s string, n int) string {
	split := strings.Split(s, ".")
	var pieces []string

	if len(split) == 1 {
		return s
	}
	startAt := len(split) - n
	for i := startAt; i >= 0; i-- {
		pieces = append(pieces, split[i])
	}
	return strings.Join(pieces, "/")
}
