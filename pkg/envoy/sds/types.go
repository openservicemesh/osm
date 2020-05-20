package sds

import (
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/logger"
)

var (
	log = logger.New("envoy/sds")
)

type task struct {
	structMaker  func(certificate.Certificater, string) (*auth.Secret, error)
	resourceName string
}
