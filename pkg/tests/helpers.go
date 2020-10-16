package tests

// GetUnique gets a slice of strings and returns a slice with the unique strings
func GetUnique(slice []string) []string {
	// Map as a set
	uniqueSet := make(map[string]struct{})
	uniqueList := []string{}

	for _, item := range slice {
		uniqueSet[item] = struct{}{}
	}

	for keyName := range uniqueSet {
		uniqueList = append(uniqueList, keyName)
	}

	return uniqueList
}
