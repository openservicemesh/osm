package injector

import "github.com/pkg/errors"

var (
	errNamespaceNotFound   = errors.New("namespace not found")
	errParseWebhookTimeout = errors.New("Could not read webhook timeout")
)
