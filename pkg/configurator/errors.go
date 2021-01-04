package configurator

import "github.com/pkg/errors"

var (
	errMissingKeyInConfigMap = errors.New("missing key in ConfigMap")
	errNilAdmissionRequest   = errors.New("nil admission request")
)
