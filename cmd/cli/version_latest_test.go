package main

import (
	"bytes"
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestOutputLatestReleaseVersion(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		err     error
		output  string
	}{
		{
			name:    "invalid current version",
			current: "v1.0.0.0",
			latest:  "v1.0.0",
			err:     fmt.Errorf("illegal version string \"v1.0.0.0\""),
			output:  "",
		},
		{
			name:    "invalid latest version",
			current: "v1.0.0",
			latest:  "1.0.0.0",
			err:     fmt.Errorf("illegal version string \"1.0.0.0\""),
			output:  "",
		},
		{
			name:    "current is less than latest version",
			current: "v1.21.4",
			latest:  "v1.21.5",
			err:     nil,
			output:  "\nOSM v1.21.5 is now available. Please see https://github.com/openservicemesh/osm/releases/latest.\nWARNING: upgrading could introduce breaking changes. Please review the release notes.\n\n",
		},
		{
			name:    "current is latest version",
			current: "v1.23.1",
			latest:  "v1.23.1",
			err:     nil,
			output:  "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)

			buf := bytes.NewBuffer(nil)
			err := outputLatestReleaseVersion(buf, test.latest, test.current)

			assert.Equal(test.err, err)
			assert.Equal(test.output, buf.String())
		})
	}
}
