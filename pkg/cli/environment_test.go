package cli

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/pflag"
	tassert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestNew(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "osm_test_*.yaml")
	require.Nil(t, err)
	defer os.Remove(tmpFile.Name()) //nolint: errcheck

	envConfig := EnvConfig{
		Install: EnvConfigInstall{
			Kind:      "self-hosted",
			Namespace: "test",
		},
	}
	data, err := yaml.Marshal(&envConfig)
	require.Nil(t, err)
	err = ioutil.WriteFile(tmpFile.Name(), data, 0600)
	require.Nil(t, err)

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
			name: "config file overrides default",
			args: nil,
			envVars: map[string]string{
				osmConfigEnvVar: tmpFile.Name(),
			},
			expectedNamespace: envConfig.Install.Namespace,
		},
		{
			name: "flag overrides config file",
			args: []string{"--osm-namespace=osm-ns"},
			envVars: map[string]string{
				osmConfigEnvVar: tmpFile.Name(),
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
	configPath := "This doesn't even look like a valid path name"
	env.config.KubeConfig = &configPath

	tassert.Equal(t, env.Namespace(), "osm-system")
}

func TestRESTClientGetter(t *testing.T) {
	env := New()
	tassert.Same(t, env.config, env.RESTClientGetter())
}
