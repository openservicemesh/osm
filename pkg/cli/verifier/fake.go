package verifier

import (
	"os"

	"github.com/pkg/errors"
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
		return nil, errors.Errorf("parsed Envoy config %s is empty", f.configFilePath)
	}
	return cfg, nil
}
