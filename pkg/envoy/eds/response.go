package eds

import (
	"strings"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	namespacedNameDelimiter = "/"
)

// NewResponse creates a new Endpoint Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, _ configurator.Configurator, _ certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	proxyIdentity, err := envoy.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}

	if request == nil {
		return nil, errors.Errorf("Endpoint discovery request for proxy %s cannot be nil", proxyIdentity)
	}

	var rdsResources []types.Resource
	for _, cluster := range request.ResourceNames {
		meshSvc, err := clusterToMeshSvc(cluster)
		if err != nil {
			log.Error().Err(err).Msgf("Error retrieving MeshService from Cluster %s", cluster)
			continue
		}
		endpoints, err := meshCatalog.ListAllowedEndpointsForService(proxyIdentity.ToServiceIdentity(), meshSvc)
		if err != nil {
			log.Error().Err(err).Msgf("Failed listing allowed endpoints for service %s, for proxy identity %s", meshSvc, proxyIdentity)
			continue
		}
		loadAssignment := newClusterLoadAssignment(meshSvc, endpoints)
		rdsResources = append(rdsResources, loadAssignment)
	}

	return rdsResources, nil
}

func clusterToMeshSvc(cluster string) (service.MeshService, error) {
	chunks := strings.Split(cluster, namespacedNameDelimiter)
	if len(chunks) != 2 {
		return service.MeshService{}, errors.Errorf("Invalid cluster name. Expected: <namespace>/<name>, Got: %s", cluster)
	}
	return service.MeshService{Namespace: chunks[0], Name: chunks[1]}, nil
}
