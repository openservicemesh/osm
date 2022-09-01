package verifier

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/types"
)

type fakeConfigGetter struct {
	configFilePath string
}

// Get returns the Config parsed from the config dump file
func (f fakeConfigGetter) Get() (*Config, error) {
	sampleConfig, err := os.ReadFile(f.configFilePath) //#nosec G304: file inclusion via variable
	if err != nil {
		return nil, err
	}
	cfg, err := parseEnvoyConfig(sampleConfig)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("parsed Envoy config %s is empty", f.configFilePath)
	}
	return cfg, nil
}

type fakeHTTPProber struct {
	err error
}

// Probe returns the error specified in the fakeHTTPProber
func (f fakeHTTPProber) Probe(_ types.NamespacedName) error {
	return f.err
}
