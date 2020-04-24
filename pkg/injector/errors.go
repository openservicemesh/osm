package injector

import "errors"

var (
	// ErrInvalidWebhookName is the error when the webhook name specified is an empty string.
	ErrInvalidWebhookName = errors.New("invalid webhook name")
)
