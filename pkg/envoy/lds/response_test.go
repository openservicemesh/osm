package lds

import (
	"testing"
	"time"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	specs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/catalog"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestNewResponse(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)

	stop := make(chan struct{})

	mockMeshSpec.EXPECT().ListTrafficTargets(gomock.Any()).Return([]*access.TrafficTarget{&tests.TrafficTarget, &tests.BookstoreV2TrafficTarget}).AnyTimes()
	mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*specs.HTTPRouteGroup{&tests.HTTPRouteGroup}).AnyTimes()
	mockMeshSpec.EXPECT().ListTrafficSplits(gomock.Any()).Return([]*split.TrafficSplit{}).AnyTimes()

	pod := tests.NewPodFixture(tests.Namespace, tests.BookbuyerServiceName, tests.BookbuyerServiceAccountName, map[string]string{
		constants.AppLabel:               tests.BookbuyerService.Name,
		constants.EnvoyUniqueIDLabelName: tests.ProxyUUID,
	})
	pod.Annotations = map[string]string{
		constants.PrometheusScrapeAnnotation: "true",
	}
	proxy := envoy.NewProxy(envoy.KindSidecar, uuid.MustParse(tests.ProxyUUID), identity.New(tests.BookbuyerServiceAccountName, tests.Namespace), nil, 1)
	provider := compute.NewMockInterface(mockCtrl)
	provider.EXPECT().ListEgressPoliciesForServiceAccount(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetIngressBackendPolicyForService(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetUpstreamTrafficSettingByService(gomock.Any()).Return(nil).AnyTimes()
	provider.EXPECT().GetUpstreamTrafficSettingByNamespace(gomock.Any()).Return(nil).AnyTimes()

	provider.EXPECT().GetServicesForServiceIdentity(tests.BookstoreServiceIdentity).Return([]service.MeshService{
		tests.BookstoreApexService,
		tests.BookstoreV1Service,
		tests.BookstoreV2Service,
	}).AnyTimes()
	provider.EXPECT().GetServicesForServiceIdentity(tests.BookstoreV2ServiceIdentity).Return([]service.MeshService{
		tests.BookstoreApexService,
		tests.BookstoreV2Service,
	}).AnyTimes()
	provider.EXPECT().GetResolvableEndpointsForService(gomock.Any()).Return([]endpoint.Endpoint{tests.Endpoint}).AnyTimes()
	provider.EXPECT().GetHostnamesForService(gomock.Any(), gomock.Any()).Return([]string{"dummy-hostname"}).AnyTimes()
	provider.EXPECT().IsMetricsEnabled(gomock.Any()).Return(true, nil).AnyTimes()
	provider.EXPECT().GetMeshConfig().Return(configv1alpha2.MeshConfig{
		Spec: configv1alpha2.MeshConfigSpec{
			Traffic: configv1alpha2.TrafficSpec{
				EnablePermissiveTrafficPolicyMode: false,
				EnableEgress:                      true,
			},
			Observability: configv1alpha2.ObservabilitySpec{
				Tracing: configv1alpha2.TracingSpec{
					Enable: false,
				},
			},
			FeatureFlags: configv1alpha2.FeatureFlags{
				EnableEgressPolicy: true,
			},
		},
	}).AnyTimes()
	provider.EXPECT().ListServicesForProxy(proxy).Return([]service.MeshService{tests.BookbuyerService}, nil).AnyTimes()

	meshCatalog := catalog.NewMeshCatalog(
		mockMeshSpec,
		tresorFake.NewFake(time.Hour),
		stop,
		provider,
		messaging.NewBroker(stop),
	)

	cm := tresorFake.NewFake(1 * time.Hour)
	resources, err := NewResponse(meshCatalog, proxy, cm, nil)
	assert.Empty(err)
	assert.NotNil(resources)
	// There are 3 listeners configured based on the configuration:
	// 1. Outbound listener (outbound-listener)
	// 2. inbound listener (inbound-listener)
	// 3. Prometheus listener (inbound-prometheus-listener)
	assert.Len(resources, 3)

	// validating outbound listener
	listener, ok := resources[0].(*xds_listener.Listener)
	assert.True(ok)
	assert.Equal(listener.Name, OutboundListenerName)
	assert.Equal(listener.TrafficDirection, xds_core.TrafficDirection_OUTBOUND)
	assert.Len(listener.ListenerFilters, 3) // Test has egress policy feature enabled, so 3 filters are expected: OriginalDst, TlsInspector, HttpInspector
	assert.Equal(envoy.OriginalDstFilterName, listener.ListenerFilters[0].Name)
	assert.NotNil(listener.FilterChains)
	// There are 3 filter chains configured on the outbound-listener based on the configuration:
	// 1. Filter chain for bookstore-v1
	// 2. Filter chain for bookstore-v2
	// 3. Filter chain for bookstore-apex due to TrafficSplit being configured
	expectedServiceFilterChainNames := []string{"outbound_default/bookstore-v1_8888_http", "outbound_default/bookstore-v2_8888_http", "outbound_default/bookstore-apex_8888_http"}
	var actualServiceFilterChainNames []string
	for _, fc := range listener.FilterChains {
		actualServiceFilterChainNames = append(actualServiceFilterChainNames, fc.Name)
	}
	assert.ElementsMatch(expectedServiceFilterChainNames, actualServiceFilterChainNames)
	assert.Len(listener.FilterChains, 3)
	assert.NotNil(listener.DefaultFilterChain)
	assert.Equal(listener.DefaultFilterChain.Name, outboundEgressFilterChainName)
	assert.Equal(listener.DefaultFilterChain.Filters[0].Name, envoy.TCPProxyFilterName)

	// validating inbound listener
	listener, ok = resources[1].(*xds_listener.Listener)
	assert.True(ok)
	assert.Equal(listener.Name, InboundListenerName)
	assert.Equal(listener.TrafficDirection, xds_core.TrafficDirection_INBOUND)
	assert.Len(listener.ListenerFilters, 2)
	assert.Equal(listener.ListenerFilters[0].Name, envoy.TLSInspectorFilterName)
	assert.Equal(listener.ListenerFilters[1].Name, envoy.OriginalDstFilterName)
	assert.NotNil(listener.FilterChains)
	// There is 1 filter chains configured on the inbound-listner based on the configuration:
	// 1. Filter chanin for bookbuyer
	assert.Len(listener.FilterChains, 1)

	// validating prometheus listener
	listener, ok = resources[2].(*xds_listener.Listener)
	assert.True(ok)
	assert.Equal(listener.Name, prometheusListenerName)
	assert.Equal(listener.TrafficDirection, xds_core.TrafficDirection_INBOUND)
	assert.NotNil(listener.FilterChains)
	assert.Len(listener.FilterChains, 1)
}
