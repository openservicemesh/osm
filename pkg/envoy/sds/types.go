package sds

import (
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/service"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/logger"
)

var (
	log = logger.New("envoy/sds")
)

type task struct {
	makeEnvoyProto func(cert certificate.Certificater, taskName string, serviceForProxy service.NamespacedService, catalog catalog.MeshCataloger) (*auth.Secret, error)
	resourceName   string
}
