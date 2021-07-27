package ads

import (
	"fmt"
	"testing"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestMakeRequestForAllSecrets(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)

	type testCase struct {
		name                     string
		proxyIdentity            identity.ServiceIdentity
		allowedOutboundServices  []service.MeshService
		expectedDiscoveryRequest *xds_discovery.DiscoveryRequest
	}

	proxyServiceIdentity := identity.K8sServiceAccount{Name: "test-sa", Namespace: "ns-1"}.ToServiceIdentity()
	proxySvcAccount := proxyServiceIdentity.ToK8sServiceAccount()
	certSerialNumber := certificate.SerialNumber("123456")
	proxyXDSCertCN := envoy.NewXDSCertCommonName(uuid.New(), envoy.KindSidecar, proxySvcAccount.Name, proxySvcAccount.Namespace)
	testProxy, err := envoy.NewProxy(proxyXDSCertCN, certSerialNumber, nil)
	assert.Nil(err)

	testCases := []testCase{
		{
			name:          "scenario where proxy is both downstream and upstream",
			proxyIdentity: proxyServiceIdentity,
			allowedOutboundServices: []service.MeshService{
				{Name: "service-2", Namespace: "ns-2"},
				{Name: "service-3", Namespace: "ns-3"},
			},
			expectedDiscoveryRequest: &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					// 1. Proxy's own cert to present to peer during mTLS/TLS handshake
					"service-cert:ns-1/test-sa",

					// 2. Outbound validation certs to validate upstreams
					"root-cert-for-mtls-outbound:ns-2/service-2",
					"root-cert-for-mtls-outbound:ns-3/service-3",

					// 3. Inbound validation certs to validate downstreams
					"root-cert-for-mtls-inbound:ns-1/test-sa",
				},
			},
		},
		{
			name:          "scenario where proxy is only a downsteam (no service)",
			proxyIdentity: proxyServiceIdentity,
			allowedOutboundServices: []service.MeshService{
				{Name: "service-2", Namespace: "ns-2"},
				{Name: "service-3", Namespace: "ns-3"},
			},
			expectedDiscoveryRequest: &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					// 1. Proxy's own cert to present to peer during mTLS/TLS handshake
					"service-cert:ns-1/test-sa",

					// 2. Outbound validation certs to validate upstreams
					"root-cert-for-mtls-outbound:ns-2/service-2",
					"root-cert-for-mtls-outbound:ns-3/service-3",

					// 3. Inbound validation certs to validate downstreams
					"root-cert-for-mtls-inbound:ns-1/test-sa",
				},
			},
		},
		{
			name:                    "scenario where proxy does not have allowed upstreams to connect to",
			proxyIdentity:           proxyServiceIdentity,
			allowedOutboundServices: nil,
			expectedDiscoveryRequest: &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					// 1. Proxy's own cert to present to peer during mTLS/TLS handshake
					"service-cert:ns-1/test-sa",

					// 4. Inbound validation certs to validate downstreams
					"root-cert-for-mtls-inbound:ns-1/test-sa",
				},
			},
		},
		{
			name:          "scenario where proxy is both downstream and upstream, with mutiple upstreams on the proxy",
			proxyIdentity: proxyServiceIdentity,
			allowedOutboundServices: []service.MeshService{
				{Name: "service-2", Namespace: "ns-2"},
				{Name: "service-3", Namespace: "ns-3"},
			},
			expectedDiscoveryRequest: &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					// 1. Proxy's own cert to present to peer during mTLS/TLS handshake
					"service-cert:ns-1/test-sa",

					// 2. Outbound validation certs to validate upstreams
					"root-cert-for-mtls-outbound:ns-2/service-2",
					"root-cert-for-mtls-outbound:ns-3/service-3",

					// 4. Inbound validation certs to validate downstreams
					"root-cert-for-mtls-inbound:ns-1/test-sa",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			mockCatalog.EXPECT().ListOutboundServicesForIdentity(tc.proxyIdentity).Return(tc.allowedOutboundServices).Times(1)

			actual := makeRequestForAllSecrets(testProxy, mockCatalog)

			assert.Equal(tc.expectedDiscoveryRequest.TypeUrl, actual.TypeUrl)
			assert.ElementsMatch(tc.expectedDiscoveryRequest.ResourceNames, actual.ResourceNames)
		})
	}
}
