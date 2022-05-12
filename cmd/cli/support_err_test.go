package main

import (
	"bytes"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

// TestErrInfoRun tests errInfoCmd.run()

func TestErrInfoRun(t *testing.T) {
	testCases := []struct {
		name      string
		errCode   string
		expectErr bool
	}{
		{
			name:      "valid error code as input",
			errCode:   "E1000",
			expectErr: false,
		},
		{
			name:      "invalid error code format as input",
			errCode:   "Foo",
			expectErr: true,
		},
		{
			name:      "valid error code format but unrecognized code as input",
			errCode:   "E10000",
			expectErr: true,
		},
		{
			name:      "list all error codes",
			errCode:   "",
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			var out bytes.Buffer
			cmd := &errInfoCmd{out: &out}
			err := cmd.run(tc.errCode)

			assert.Equal(tc.expectErr, err != nil)
		})
	}
}
