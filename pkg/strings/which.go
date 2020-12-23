package strings

// Which is a slice of strings
type Which []string

// NotEqual returns a new slice of strings from an existing slice that do not match a given string
func (which Which) NotEqual(rightSide string) []string {
	notEqual := []string{}
	for _, leftSide := range which {
		if leftSide != rightSide {
			notEqual = append(notEqual, leftSide)
		}
	}
	return notEqual
}
