package strings

type All []string

func (all All) Equal(rightSide string) bool {
	for _, leftSide := range all {
		if leftSide != rightSide {
			return false
		}
	}
	return true
}
