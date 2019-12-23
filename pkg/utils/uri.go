package utils

import "strings"

// GetLastChunkOfSlashed splits a string by slash and returns the last chunk.
func GetLastChunkOfSlashed(s string) string {
	split := strings.Split(s, "/")
	return split[len(split)-1]
}
