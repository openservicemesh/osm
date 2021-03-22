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
	"github.com/openservicemesh/osm/pkg/service"
)

func TestMakeRequestForAllSecrets(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)

	type testCase struct {
		name                     string
		proxySvcAccount          service.K8sServiceAccount
		proxyServices            []service.MeshService
		allowedOutboundServices  []service.MeshService
		expectedDiscoveryRequest *xds_discovery.DiscoveryRequest
	}

	proxySvcAccount := service.K8sServiceAccount{Name: "test-sa", Namespace: "ns-1"}
	certSerialNumber := certificate.SerialNumber("123456")
	proxyXDSCertCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New().String(), proxySvcAccount.Name, proxySvcAccount.Namespace))
	testProxy := envoy.NewProxy(proxyXDSCertCN, certSerialNumber, nil)

	testCases := []testCase{
		{
			name:            "scenario where proxy is both downstream and upstream",
			proxySvcAccount: proxySvcAccount,
			proxyServices: []service.MeshService{
				{Name: "service-1", Namespace: "ns-1"},
			},
			allowedOutboundServices: []service.MeshService{
				{Name: "service-2", Namespace: "ns-2"},
				{Name: "service-3", Namespace: "ns-3"},
			},
			expectedDiscoveryRequest: &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					// 1. Client service cert
					"service-cert:ns-1/test-sa",

					// 2. Outbound validation certs to validate upstreams
					"root-cert-for-mtls-outbound:ns-2/service-2",
					"root-cert-for-mtls-outbound:ns-3/service-3",

					// 3. Server service cert
					"service-cert:ns-1/service-1",

					// 4. Inbound validation certs to validate downstreams
					"root-cert-for-mtls-inbound:ns-1/service-1",
					"root-cert-https:ns-1/service-1",
				},
			},
		},
		{
			name:            "scenario where proxy is only a downsteam (no service)",
			proxySvcAccount: proxySvcAccount,
			proxyServices:   nil,
			allowedOutboundServices: []service.MeshService{
				{Name: "service-2", Namespace: "ns-2"},
				{Name: "service-3", Namespace: "ns-3"},
			},
			expectedDiscoveryRequest: &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					// 1. Client service cert
					"service-cert:ns-1/test-sa",

					// 2. Outbound validation certs to validate upstreams
					"root-cert-for-mtls-outbound:ns-2/service-2",
					"root-cert-for-mtls-outbound:ns-3/service-3",
				},
			},
		},
		{
			name:            "scenario where proxy does not have allowed upstreams to connect to",
			proxySvcAccount: proxySvcAccount,
			proxyServices: []service.MeshService{
				{Name: "service-1", Namespace: "ns-1"},
			},
			allowedOutboundServices: nil,
			expectedDiscoveryRequest: &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					// 1. Client service cert
					"service-cert:ns-1/test-sa",

					// 3. Server service cert
					"service-cert:ns-1/service-1",

					// 4. Inbound validation certs to validate downstreams
					"root-cert-for-mtls-inbound:ns-1/service-1",
					"root-cert-https:ns-1/service-1",
				},
			},
		},
		{
			name:            "scenario where proxy is both downstream and upstream, with mutiple upstreams on the proxy",
			proxySvcAccount: proxySvcAccount,
			proxyServices: []service.MeshService{
				{Name: "service-1", Namespace: "ns-1"},
				{Name: "service-4", Namespace: "ns-4"},
			},
			allowedOutboundServices: []service.MeshService{
				{Name: "service-2", Namespace: "ns-2"},
				{Name: "service-3", Namespace: "ns-3"},
			},
			expectedDiscoveryRequest: &xds_discovery.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					// 1. Client service cert
					"service-cert:ns-1/test-sa",

					// 2. Outbound validation certs to validate upstreams
					"root-cert-for-mtls-outbound:ns-2/service-2",
					"root-cert-for-mtls-outbound:ns-3/service-3",

					// 3. Server service cert
					"service-cert:ns-1/service-1",
					"service-cert:ns-4/service-4",

					// 4. Inbound validation certs to validate downstreams
					"root-cert-for-mtls-inbound:ns-1/service-1",
					"root-cert-https:ns-1/service-1",
					"root-cert-for-mtls-inbound:ns-4/service-4",
					"root-cert-https:ns-4/service-4",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockCatalog.EXPECT().GetServicesFromEnvoyCertificate(gomock.Any()).Return(tc.proxyServices, nil).Times(1)
			mockCatalog.EXPECT().ListAllowedOutboundServicesForIdentity(tc.proxySvcAccount).Return(tc.allowedOutboundServices).Times(1)

			actual := makeRequestForAllSecrets(testProxy, mockCatalog)

			assert.Equal(tc.expectedDiscoveryRequest.TypeUrl, actual.TypeUrl)
			assert.ElementsMatch(tc.expectedDiscoveryRequest.ResourceNames, actual.ResourceNames)
		})
	}
}
