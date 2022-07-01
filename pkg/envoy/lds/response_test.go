package lds

import (
	"context"
	"fmt"
	"testing"
	"time"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"

	"github.com/openservicemesh/osm/pkg/auth"
	catalogFake "github.com/openservicemesh/osm/pkg/catalog/fake"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func getProxy(kubeClient kubernetes.Interface) (*envoy.Proxy, *v1.Pod, error) {
	podLabels := map[string]string{
		constants.AppLabel:               tests.BookbuyerService.Name,
		constants.EnvoyUniqueIDLabelName: tests.ProxyUUID,
	}

	newPod1 := tests.NewPodFixture(tests.Namespace, tests.BookbuyerServiceName, tests.BookbuyerServiceAccountName, podLabels)
	newPod1.Annotations = map[string]string{
		constants.PrometheusScrapeAnnotation: "true",
	}
	if _, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), newPod1, metav1.CreateOptions{}); err != nil {
		return nil, nil, err
	}

	selectors := map[string]string{
		constants.AppLabel: tests.BookbuyerServiceName,
	}
	if _, err := tests.MakeService(kubeClient, tests.BookbuyerServiceName, selectors); err != nil {
		return nil, nil, err
	}

	for _, svcName := range []string{tests.BookstoreApexServiceName, tests.BookstoreV1ServiceName, tests.BookstoreV2ServiceName} {
		selectors := map[string]string{
			constants.AppLabel: "bookstore",
		}
		if _, err := tests.MakeService(kubeClient, svcName, selectors); err != nil {
			return nil, nil, err
		}
	}

	return envoy.NewProxy(envoy.KindSidecar, uuid.MustParse(tests.ProxyUUID), identity.New(tests.BookbuyerServiceAccountName, tests.Namespace), nil), newPod1, nil
}

func TestNewResponse(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	kubeClient := testclient.NewSimpleClientset()
	configClient := configFake.NewSimpleClientset()
	meshCatalog := catalogFake.NewFakeMeshCatalog(kubeClient, configClient)

	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("some-endpoint").AnyTimes()
	mockConfigurator.EXPECT().IsEgressEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
		Enable: false,
	}).AnyTimes()
	mockConfigurator.EXPECT().GetMeshConfig().AnyTimes()

	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha2.FeatureFlags{
		EnableWASMStats:    false,
		EnableEgressPolicy: true,
	}).AnyTimes()

	proxy, pod, err := getProxy(kubeClient)
	assert.Empty(err)
	assert.NotNil(proxy)
	assert.NotNil(pod)

	mockController := meshCatalog.GetKubeController().(*k8s.MockController)
	mockController.EXPECT().GetPodForProxy(proxy).Return(pod, nil)

	// test scenario that listing proxy services returns an error
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, fmt.Errorf("dummy error")
	}), nil)

	cm := tresorFake.NewFake(nil, 1*time.Hour)
	resources, err := NewResponse(meshCatalog, proxy, nil, mockConfigurator, cm, proxyRegistry)
	assert.NotNil(err)
	assert.Nil(resources)

	proxyRegistry = registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return []service.MeshService{tests.BookbuyerService}, nil
	}), nil)

	resources, err = NewResponse(meshCatalog, proxy, nil, mockConfigurator, cm, proxyRegistry)
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
