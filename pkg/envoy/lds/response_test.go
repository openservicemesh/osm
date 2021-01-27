package lds

import (
	"fmt"
	"testing"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	tassert "github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

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
	if _, err := tests.MakePod(kubeClient, tests.Namespace, tests.BookbuyerServiceName, tests.BookbuyerServiceAccountName, podLabels); err != nil {
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

	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", tests.ProxyUUID, tests.BookbuyerServiceAccountName, tests.Namespace))
	certSerialNumber := certificate.SerialNumber("123456")
	proxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)
	return proxy, nil
}

func TestListenerConfiguration(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	kubeClient := testclient.NewSimpleClientset()
	meshCatalog := catalog.NewFakeMeshCatalog(kubeClient)

	proxy, err := getProxy(kubeClient)
	assert.Empty(err)
	assert.NotNil(meshCatalog)
	assert.NotNil(proxy)

	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsPrometheusScrapingEnabled().Return(true).AnyTimes()
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsEgressEnabled().Return(true).AnyTimes()

	actual, err := NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil)
	assert.Empty(err)
	assert.NotNil(actual)
	// There are 3 listeners configured based on the configuration:
	// 1. Outbound listener (outbound-listener)
	// 2. inbound listener (inbound-listener)
	// 3. Prometheus listener (inbound-prometheus-listener)
	assert.Len(actual.Resources, 3)

	listener := xds_listener.Listener{}

	// validating outbound listener
	err = ptypes.UnmarshalAny(actual.Resources[0], &listener)
	assert.Empty(err)
	assert.Equal(listener.Name, outboundListenerName)
	assert.Equal(listener.TrafficDirection, xds_core.TrafficDirection_OUTBOUND)
	assert.Len(listener.ListenerFilters, 1)
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
	err = ptypes.UnmarshalAny(actual.Resources[1], &listener)
	assert.Empty(err)
	assert.Equal(listener.Name, inboundListenerName)
	assert.Equal(listener.TrafficDirection, xds_core.TrafficDirection_INBOUND)
	assert.Len(listener.ListenerFilters, 2)
	assert.Equal(listener.ListenerFilters[0].Name, wellknown.TlsInspector)
	assert.Equal(listener.ListenerFilters[1].Name, wellknown.OriginalDestination)
	assert.NotNil(listener.FilterChains)
	// There is 1 filter chains configured on the inbound-listner based on the configuration:
	// 1. Filter chanin for bookbuyer
	assert.Len(listener.FilterChains, 1)

	// validating prometheus listener
	err = ptypes.UnmarshalAny(actual.Resources[2], &listener)
	assert.Empty(err)
	assert.Equal(listener.Name, prometheusListenerName)
	assert.Equal(listener.TrafficDirection, xds_core.TrafficDirection_INBOUND)
	assert.NotNil(listener.FilterChains)
	assert.Len(listener.FilterChains, 1)
}
