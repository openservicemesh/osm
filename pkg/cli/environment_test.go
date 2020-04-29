package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

type testCase struct {
	name              string
	argsInput         string
	envVarsInput      map[string]string
	expectedNamespace string
}

func TestNew(t *testing.T) {

	tests := []testCase{
		testCase{
			name:              "default",
			expectedNamespace: "osm-system",
		},
		testCase{
			name:              "with namespace flag set",
			argsInput:         "--namespace=osm-ns",
			expectedNamespace: "osm-ns",
		},
		testCase{
			name: "with envvar set",
			envVarsInput: map[string]string{
				"OSM_NAMESPACE": "osm-env",
			},
			expectedNamespace: "osm-env",
		},
		testCase{
			name:      "with flags and envvar set",
			argsInput: "--namespace=osm-flag-ns",
			envVarsInput: map[string]string{
				"OSM_NAMESPACE": "osm-env",
			},
			expectedNamespace: "osm-flag-ns",
		},
	}

	for _, test := range tests {
		defer resetEnv()()

		for k, v := range test.envVarsInput {
			os.Setenv(k, v)
		}

		flags := pflag.NewFlagSet("test-new", pflag.ContinueOnError)

		settings := New()
		settings.AddFlags(flags)
		flags.Parse(strings.Split(test.argsInput, " "))

		if settings.Namespace() != test.expectedNamespace {
			t.Errorf("[test: %s] expected namespace %s, got %s", test.name, test.expectedNamespace, settings.Namespace())
		}
	}
}

func resetEnv() func() {
	origEnv := os.Environ()

	// unset any local env vars
	for e := range New().EnvVars() {
		os.Unsetenv(e)
	}

	return func() {
		for _, pair := range origEnv {
			kv := strings.SplitN(pair, "=", 2)
			os.Setenv(kv[0], kv[1])
		}
	}
}
