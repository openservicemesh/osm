package eds

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Endpoint Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	meshSvcEndpoints := make(map[service.MeshService][]endpoint.Endpoint)
	builder := NewEndpointsBuilder()

	for _, dstSvc := range meshCatalog.ListOutboundServicesForIdentity(proxy.Identity) {
		builder.AddEndpoints(
			dstSvc,
			meshCatalog.ListAllowedUpstreamEndpointsForService(proxy.Identity, dstSvc),
		)

		log.Trace().Msgf("Allowed outbound service endpoints for proxy with identity %s: %v", proxy.Identity, meshSvcEndpoints)
	}

	return builder.Build(), nil
}

// clusterToMeshSvc returns the MeshService associated with the given cluster name
func clusterToMeshSvc(cluster string) (service.MeshService, error) {
	splitFunc := func(r rune) bool {
		return r == '/' || r == '|'
	}

	chunks := strings.FieldsFunc(cluster, splitFunc)
	if len(chunks) != 3 {
		return service.MeshService{}, fmt.Errorf("Invalid cluster name. Expected: <namespace>/<name>|<port>, got: %s", cluster)
	}

	port, err := strconv.ParseUint(chunks[2], 10, 16)
	if err != nil {
		return service.MeshService{}, fmt.Errorf("Invalid cluster port %s, expected int value: %w", chunks[2], err)
	}

	subdomain, name, ok := strings.Cut(chunks[1], ".")
	if !ok {
		name = subdomain
		subdomain = ""
	}

	return service.MeshService{
		Namespace: chunks[0],
		Name:      name,
		Subdomain: subdomain,

		// The port always maps to MeshService.TargetPort and not MeshService.Port because
		// endpoints of a service are derived from it's TargetPort and not Port.
		TargetPort: uint16(port),
	}, nil
}
