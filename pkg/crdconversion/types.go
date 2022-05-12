// Package crdconversion implements OSM's CRD conversion facility. The crd-converter webhook
// server intercepts crd get/list/create/update requests to apply the required conversion logic.
package crdconversion

import (
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("crd-conversion")

// crdConversionWebhook is the type used to represent the webhook for the crd converter
type crdConversionWebhook struct {
	config Config
	cert   *certificate.Certificate
}

// Config is the type used to represent the config options for the crd-conversion webhook
type Config struct {
	// ListenPort defines the port on which the crd-conversion webhook listens
	ListenPort int
}
