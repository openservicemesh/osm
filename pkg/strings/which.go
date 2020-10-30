package strings

type Which []string

func (which Which) NotEqual(rightSide string) []string {
	notEqual := []string{}
	for _, leftSide := range which {
		if leftSide != rightSide {
			notEqual = append(notEqual, leftSide)
		}
	}
	return notEqual
}
