package utils

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestPrettyJson(t *testing.T) {
	assert := tassert.New(t)

	type prettyJSONtest struct {
		input              []byte
		prefix             string
		expectedPrettyJSON []byte
		expectedError      string
	}

	prettyJSONtests := []prettyJSONtest{{
		[]byte("{\"name\":\"baba yaga\"}"),
		"--prefix--",
		[]byte(`{
--prefix--    "name": "baba yaga"
--prefix--}`),
		""},
		{[]byte("should error"),
			"",
			nil,
			"Could not Unmarshal a byte array"},
	}

	for _, pjt := range prettyJSONtests {
		prettyJSON, err := PrettyJSON(pjt.input, pjt.prefix)
		if err != nil {
			assert.Contains(err.Error(), pjt.expectedError)
			assert.Nil(prettyJSON)
		} else {
			assert.Equal(pjt.expectedPrettyJSON, prettyJSON)
			assert.Empty(pjt.expectedError)
		}
	}
}
