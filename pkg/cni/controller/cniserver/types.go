// Package cniserver implements OSM CNI Control Server.
package cniserver

import "github.com/openservicemesh/osm/pkg/logger"

var (
	log = logger.New("interceptor-ctrl-server")
)

// Server CNI Server.
type Server interface {
	Start() error
	Stop()
}
