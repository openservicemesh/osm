package catalog

import "errors"

var (
	errNotFound        = errors.New("no such service found")
	errUnregisterProxy = errors.New("unregister proxy")
)
