package errcode

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	assert := tassert.New(t)
	assert.Equal(ErrInvalidCLIArgument.String(), "E1000")
}

func TestGetErrorCodeWithMetric(t *testing.T) {
	assert := tassert.New(t)
	assert.Equal(GetErrCodeWithMetric(ErrInvalidCLIArgument), "E1000")
}

func TestFromStr(t *testing.T) {
	testCases := []struct {
		name            string
		str             string
		expectedErrCode ErrCode
		expectError     bool
	}{
		{
			name:            "valid error code",
			str:             "E1000",
			expectedErrCode: ErrInvalidCLIArgument,
			expectError:     false,
		},
		{
			name:            "invalid err code",
			str:             "invalid",
			expectedErrCode: ErrCode(0),
			expectError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual, err := FromStr(tc.str)

			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedErrCode, actual)
		})
	}
}
