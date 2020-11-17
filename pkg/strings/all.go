package strings

// All is slice of strings
type All []string

// Equal checks if a string exists in a slice of strings
func (all All) Equal(rightSide string) bool {
	for _, leftSide := range all {
		if leftSide != rightSide {
			return false
		}
	}
	return true
}
