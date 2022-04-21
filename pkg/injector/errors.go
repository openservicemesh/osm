package injector

import "github.com/pkg/errors"

var (
	errNamespaceNotFound   = errors.New("namespace not found")
	errNilAdmissionRequest = errors.New("nil admission request")
)
