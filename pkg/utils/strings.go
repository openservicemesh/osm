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
	return strings.Join(pieces, ".")
}
