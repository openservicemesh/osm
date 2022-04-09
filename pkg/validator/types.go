package validator

import (
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/pkg/errors"
)

var log = logger.New(ValidatorWebhookSvc)

var ErrIngressBackendDuplicateBackends = errors.New("error: duplicate backends detected")
