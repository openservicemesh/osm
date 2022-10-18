package generator

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy/generator/sds"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Secrets Discovery Response.
func (g *EnvoyConfigGenerator) generateSDS(ctx context.Context, proxy *models.Proxy) ([]types.Resource, error) {
	builder := sds.NewBuilder().SetProxy(proxy).SetTrustDomain(g.certManager.GetTrustDomains())

	// 1. Issue a service certificate for this proxy
	cert, err := g.certManager.IssueCertificate(certificate.ForServiceIdentity(proxy.Identity))
	if err != nil {
		log.Error().Err(err).Str("proxy", proxy.String()).Msgf("Error issuing a certificate for proxy")
		return nil, err
	}
	builder.SetProxyCert(cert)

	// Set service identities for services in requests
	serviceIdentitiesForOutboundServices := make(map[service.MeshService][]identity.ServiceIdentity)

	for _, svc := range g.catalog.ListOutboundServicesForIdentity(proxy.Identity) {
		identities, err := g.catalog.ListServiceIdentitiesForService(svc.Name, svc.Namespace)
		if err != nil {
			return nil, err
		}
		serviceIdentitiesForOutboundServices[svc] = identities
	}

	builder.SetServiceIdentitiesForService(serviceIdentitiesForOutboundServices)

	// Get SDS Secret Resources based on requested certs in the DiscoveryRequest
	var sdsResources = make([]types.Resource, 0, len(serviceIdentitiesForOutboundServices)+2)
	for _, envoyProto := range builder.Build() {
		sdsResources = append(sdsResources, envoyProto)
	}
	return sdsResources, nil
}
