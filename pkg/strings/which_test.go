package strings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWhichNotEqual(t *testing.T) {
	assertion := assert.New(t)

	listOfAlpha := Which{"a", "a"}
	listOfStuff := Which{"a", "b"}

	assertion.Equal(listOfAlpha.NotEqual("a"), []string{})
	assertion.Equal(listOfAlpha.NotEqual("b"), []string{"a", "a"})
	assertion.Equal(listOfStuff.NotEqual("a"), []string{"b"})
}
