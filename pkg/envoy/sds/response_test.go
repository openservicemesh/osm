package sds

import (
	"fmt"
	"testing"
	"time"

	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/catalog"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// TestNewResponse sets up a fake kube client, then a pod and makes an SDS request,
// and finally verifies the response from sds.NewResponse().
func TestNewResponse(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	certManager := tresorFake.NewFake(1 * time.Hour)

	// We deliberately set the namespace and service accounts to random values
	// to ensure no hard-coded values sneak in.
	proxySvcID := identity.New(uuid.New().String(), uuid.New().String())

	// This is the thing we are going to be requesting (pretending that the Envoy is requesting it)
	testCases := []struct {
		name                        string
		serviceIdentitiesForService map[service.MeshService][]identity.ServiceIdentity
		trustDomain                 string
		expectedCertToSAN           map[string][]string
	}{
		{
			name: "no identities",
			expectedCertToSAN: map[string][]string{
				secrets.NameForIdentity(proxySvcID): nil,
				secrets.NameForMTLSInbound:          nil,
			},
		},
		{
			name: "multiple outbound identities certs",
			serviceIdentitiesForService: map[service.MeshService][]identity.ServiceIdentity{
				{Name: "svc-1", Namespace: "ns-1"}: {
					identity.New("sa-1", "ns-1"),
					identity.New("sa-2", "ns-1"),
				},
				{Name: "svc-A", Namespace: "ns-A"}: {
					identity.New("sa-A", "ns-A"),
				},
			},
			expectedCertToSAN: map[string][]string{
				secrets.NameForUpstreamService("svc-1", "ns-1"): {
					"sa-1.ns-1.cluster.local",
					"sa-2.ns-1.cluster.local",
				},
				secrets.NameForUpstreamService("svc-A", "ns-A"): {
					"sa-A.ns-A.cluster.local",
				},
				secrets.NameForIdentity(proxySvcID): nil,
				secrets.NameForMTLSInbound:          nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			// The Common Name of the xDS Certificate (issued to the Envoy on the Pod by the Injector) will
			// have be prefixed with the ID of the pod. It is the first chunk of a dot-separated string.
			proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), proxySvcID, nil, 1)
			meshCatalog := catalog.NewMockMeshCataloger(mockCtrl)

			var services []service.MeshService
			for svc, identities := range tc.serviceIdentitiesForService {
				services = append(services, svc)
				meshCatalog.EXPECT().ListServiceIdentitiesForService(svc).Return(identities)
			}
			meshCatalog.EXPECT().ListOutboundServicesForIdentity(proxy.Identity).Return(services)

			// ----- Test with an properly configured proxy
			resources, err := NewResponse(meshCatalog, proxy, certManager, nil)
			assert.Equal(err, nil, fmt.Sprintf("Error evaluating sds.NewResponse(): %s", err))
			assert.NotNil(resources)
			var certNames, expectedCertNames []string

			// Collecting cert names for the assert has an easier to read print statement on failure, compared to
			// the assert.Equal statement or assert.Len, which will print either nothing or the entire cert object respectively.
			for name := range tc.expectedCertToSAN {
				expectedCertNames = append(expectedCertNames, name)
			}

			for _, resource := range resources {
				secret, ok := resource.(*xds_auth.Secret)
				assert.True(ok)
				certNames = append(certNames, secret.Name)

				assert.Contains(tc.expectedCertToSAN, secret.Name)
				if len(tc.expectedCertToSAN[secret.Name]) == 0 {
					continue // nothing more to do.
				}
				assert.Len(secret.GetValidationContext().MatchTypedSubjectAltNames, len(tc.expectedCertToSAN[secret.Name]))
				for _, matchers := range secret.GetValidationContext().MatchTypedSubjectAltNames {
					assert.Contains(tc.expectedCertToSAN[secret.Name], matchers.Matcher.GetExact())
				}
			}
			assert.ElementsMatch(expectedCertNames, certNames)
		})
	}
}
