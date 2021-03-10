package cli

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	tassert "github.com/stretchr/testify/assert"
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
			assert := tassert.New(t)

			flags := pflag.NewFlagSet("test-new", pflag.ContinueOnError)

			for k, v := range test.envVars {
				oldv, found := os.LookupEnv(k)
				defer func(k string, oldv string, found bool) {
					var err error
					if found {
						err = os.Setenv(k, oldv)
					} else {
						err = os.Unsetenv(k)
					}
					assert.Nil(err)
				}(k, oldv, found)
				err := os.Setenv(k, v)
				assert.Nil(err)
			}

			settings := New()
			settings.AddFlags(flags)
			err := flags.Parse(test.args)
			assert.Nil(err)
			assert.Equal(settings.Namespace(), test.expectedNamespace)
		})
	}
}

func TestNamespaceErr(t *testing.T) {
	env := New()

	// Tell kube to load config from a file that doesn't exist. The exact error
	// doesn't matter, this was just the simplest way to force an error to
	// occur. Users of this package are not able to do this, but the resulting
	// behavior is the same as if any other error had occurred.
	kConfigPath := "This doesn't even look like a valid path name"
	env.config.KubeConfig = &kConfigPath

	tassert.Equal(t, env.Namespace(), "default")
}

func TestEnvVars(t *testing.T) {
	env := New()
	tassert.Equal(t, map[string]string{"OSM_NAMESPACE": "osm-system"}, env.EnvVars())
}

func TestRESTClientGetter(t *testing.T) {
	env := New()
	tassert.Same(t, env.config, env.RESTClientGetter())
}
