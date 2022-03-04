package vault

import (
	"fmt"

	"github.com/pkg/errors"
)

func (options *Options) Address() string {
	return fmt.Sprintf("%s://%s:%d", options.Protocol, options.Host, options.Port)
}

// Validate validates the options for Hashi Vault certificate provider
func (options *Options) Validate() error {
	if options.Host == "" {
		return errors.New("VaultHost not specified in Hashi Vault options")
	}

	if options.Token == "" {
		return errors.New("VaultToken not specified in Hashi Vault options")
	}

	if options.Role == "" {
		return errors.New("VaultRole not specified in Hashi Vault options")
	}

	if _, ok := map[string]interface{}{"http": nil, "https": nil}[options.Protocol]; !ok {
		return errors.Errorf("VaultProtocol in Hashi Vault options must be one of [http, https], got %s", options.Protocol)
	}

	return nil
}
