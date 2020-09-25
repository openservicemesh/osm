package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrettyJson(t *testing.T) {
	assert := assert.New(t)

	type prettyJSONtest struct {
		input              []byte
		prefix             string
		expectedPrettyJSON []byte
	}

	prettyJSONtests := []prettyJSONtest{{
		[]byte("{\"name\":\"baba yaga\"}"),
		"--prefix--",
		[]byte(`{
--prefix--    "name": "baba yaga"
--prefix--}`)},
		{[]byte("should error"),
			"",
			nil},
	}

	for _, pjt := range prettyJSONtests {
		prettyJSON, err := PrettyJSON(pjt.input, pjt.prefix)

		if err != nil {
			assert.Nil(prettyJSON)
			assert.NotNil(err)
		} else {
			assert.Equal(prettyJSON, pjt.expectedPrettyJSON)
			assert.Nil(err)
		}
	}
}
