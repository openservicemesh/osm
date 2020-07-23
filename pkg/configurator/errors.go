package configurator

import "errors"

var (
	errInvalidKeyInConfigMap = errors.New("invalid key in ConfigMap")
	errMissingKeyInConfigMap = errors.New("missing key in ConfigMap")
)
