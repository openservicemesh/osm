package cli

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		envVars           map[string]string
		expectedNamespace string
	}{
		{
			name:              "default",
			args:              nil,
			envVars:           nil,
			expectedNamespace: defaultOSMNamespace,
		},
		{
			name:              "flag overrides default",
			args:              []string{"--osm-namespace=osm-ns"},
			envVars:           nil,
			expectedNamespace: "osm-ns",
		},
		{
			name: "env var overrides default",
			args: nil,
			envVars: map[string]string{
				osmNamespaceEnvVar: "osm-env",
			},
			expectedNamespace: "osm-env",
		},
		{
			name: "flag overrides env var",
			args: []string{"--osm-namespace=osm-ns"},
			envVars: map[string]string{
				osmNamespaceEnvVar: "osm-env",
			},
			expectedNamespace: "osm-ns",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			flags := pflag.NewFlagSet("test-new", pflag.ContinueOnError)

			for k, v := range test.envVars {
				os.Setenv(k, v)
			}

			settings := New()
			settings.AddFlags(flags)
			err := flags.Parse(test.args)
			assert.Nil(err)
			assert.Equal(settings.Namespace(), test.expectedNamespace)
		})
	}
}
