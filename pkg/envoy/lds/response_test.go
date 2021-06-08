package lds

import (
	"context"
	"fmt"
	"testing"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/service"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/tests"
)

func getProxy(kubeClient kubernetes.Interface) (*envoy.Proxy, error) {
	podLabels := map[string]string{
		tests.SelectorKey:                tests.BookbuyerService.Name,
		constants.EnvoyUniqueIDLabelName: tests.ProxyUUID,
	}

	newPod1 := tests.NewPodFixture(tests.Namespace, tests.BookbuyerServiceName, tests.BookbuyerServiceAccountName, podLabels)
	newPod1.Annotations = map[string]string{
		constants.PrometheusScrapeAnnotation: "true",
	}
	if _, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &newPod1, metav1.CreateOptions{}); err != nil {
		return nil, err
	}

	selectors := map[string]string{
		tests.SelectorKey: tests.BookbuyerServiceName,
	}
	if _, err := tests.MakeService(kubeClient, tests.BookbuyerServiceName, selectors); err != nil {
		return nil, err
	}

	for _, svcName := range []string{tests.BookstoreApexServiceName, tests.BookstoreV1ServiceName, tests.BookstoreV2ServiceName} {
		selectors := map[string]string{
			tests.SelectorKey: "bookstore",
		}
		if _, err := tests.MakeService(kubeClient, svcName, selectors); err != nil {
			return nil, err
		}
	}

	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", tests.ProxyUUID, envoy.KindSidecar, tests.BookbuyerServiceAccountName, tests.Namespace))
	certSerialNumber := certificate.SerialNumber("123456")
	return envoy.NewProxy(certCommonName, certSerialNumber, nil)
}

func TestNewResponse(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	kubeClient := testclient.NewSimpleClientset()
	configClient := configFake.NewSimpleClientset()
	meshCatalog := catalog.NewFakeMeshCatalog(kubeClient, configClient)

	proxy, err := getProxy(kubeClient)
	assert.Empty(err)
	assert.NotNil(proxy)

	// test scenario that listing proxy services returns an error
	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, fmt.Errorf("dummy error")
	}))
	resources, err := NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
	assert.NotNil(err)
	assert.Nil(resources)

	proxyRegistry = registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return []service.MeshService{tests.BookbuyerService}, nil
	}))

	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsEgressEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
		Enable: false,
	}).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableWASMStats:    false,
		EnableEgressPolicy: true,
	}).AnyTimes()

	resources, err = NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
	assert.Empty(err)
	assert.NotNil(resources)
	// There are 3 listeners configured based on the configuration:
	// 1. Outbound listener (outbound-listener)
	// 2. inbound listener (inbound-listener)
	// 3. Prometheus listener (inbound-prometheus-listener)
	assert.Len(resources, 3)

	// validating outbound listener
	listener, ok := resources[1].(*xds_listener.Listener)
	assert.True(ok)
	assert.Equal(listener.Name, outboundListenerName)
	assert.Equal(listener.TrafficDirection, xds_core.TrafficDirection_OUTBOUND)
	assert.Len(listener.ListenerFilters, 3) // Test has egress policy feature enabled, so 3 filters are expected: OriginalDst, TlsInspector, HttpInspector
	assert.Equal(listener.ListenerFilters[0].Name, wellknown.OriginalDestination)
	assert.NotNil(listener.FilterChains)
	// There are 3 filter chains configured on the outbound-listner based on the configuration:
	// 1. Filter chanin for bookstore-v1
	// 2. Filter chanin for bookstore-v2
	// 3. Egress filter chain
	assert.Len(listener.FilterChains, 3)
	assert.NotNil(listener.DefaultFilterChain)
	assert.Equal(listener.DefaultFilterChain.Name, outboundEgressFilterChainName)
	assert.Equal(listener.DefaultFilterChain.Filters[0].Name, wellknown.TCPProxy)

	// validating inbound listener
	listener, ok = resources[2].(*xds_listener.Listener)
	assert.True(ok)
	assert.Equal(listener.Name, inboundListenerName)
	assert.Equal(listener.TrafficDirection, xds_core.TrafficDirection_INBOUND)
	assert.Len(listener.ListenerFilters, 2)
	assert.Equal(listener.ListenerFilters[0].Name, wellknown.TlsInspector)
	assert.Equal(listener.ListenerFilters[1].Name, wellknown.OriginalDestination)
	assert.NotNil(listener.FilterChains)
	// There is 1 filter chains configured on the inbound-listener based on the configuration:
	// 1. Filter chanin for bookbuyer
	assert.Len(listener.FilterChains, 1)

	// validating prometheus listener
	listener, ok = resources[0].(*xds_listener.Listener)
	assert.True(ok)
	assert.Equal(listener.Name, prometheusListenerName)
	assert.Equal(listener.TrafficDirection, xds_core.TrafficDirection_INBOUND)
	assert.NotNil(listener.FilterChains)
	assert.Len(listener.FilterChains, 1)
}
