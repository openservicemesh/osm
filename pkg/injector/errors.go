package injector

import "github.com/pkg/errors"

var (
	errNamespaceNotFound   = errors.New("namespace not found")
	errParseWebhookTimeout = errors.New("could not read webhook timeout")
	errNilAdmissionRequest = errors.New("nil admission request")
)
