// Package crdconversion implements OSM's CRD conversion facility. The crd-converter webhook
// server intercepts crd get/list/create/update requests to apply the required conversion logic.
package crdconversion

import (
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("crd-conversion")
