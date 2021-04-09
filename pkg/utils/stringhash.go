package utils

import "hash/fnv"

// HashFromString calculates an FNV-1 hash from a given string,
// returns it as a uint64 and error, if any
func HashFromString(s string) (uint64, error) {
	h := fnv.New64()
	_, err := h.Write([]byte(s))
	if err != nil {
		return 0, err
	}

	return h.Sum64(), nil
}
