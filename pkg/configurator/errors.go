package configurator

import "github.com/pkg/errors"

var (
	errInvalidKeyInConfigMap = errors.New("invalid key in ConfigMap")
	errMissingKeyInConfigMap = errors.New("missing key in ConfigMap")
)
