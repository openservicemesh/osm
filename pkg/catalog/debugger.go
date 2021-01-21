package catalog

import (
	"time"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

// ListExpectedProxies lists the Envoy proxies yet to connect and the time their XDS certificate was issued.
func (mc *MeshCatalog) ListExpectedProxies() map[certificate.CommonName]time.Time {
	proxies := make(map[certificate.CommonName]time.Time)

	mc.expectedProxies.Range(func(cnInterface, expectedProxyInterface interface{}) bool {
		cn := cnInterface.(certificate.CommonName)
		props := expectedProxyInterface.(expectedProxy)

		_, isConnected := mc.connectedProxies.Load(cn)
		_, isDisconnected := mc.disconnectedProxies.Load(cn)

		if !isConnected && !isDisconnected {
			proxies[cn] = props.certificateIssuedAt
		}

		return true // continue the iteration
	})

	return proxies
}

// ListConnectedProxies lists the Envoy proxies already connected and the time they first connected.
func (mc *MeshCatalog) ListConnectedProxies() map[certificate.CommonName]*envoy.Proxy {
	proxies := make(map[certificate.CommonName]*envoy.Proxy)
	mc.connectedProxies.Range(func(cnIface, propsIface interface{}) bool {
		cn := cnIface.(certificate.CommonName)
		props := propsIface.(connectedProxy)
		if _, isDisconnected := mc.disconnectedProxies.Load(cn); !isDisconnected {
			proxies[cn] = props.proxy
		}
		return true // continue the iteration
	})
	return proxies
}

// ListDisconnectedProxies lists the Envoy proxies disconnected and the time last seen.
func (mc *MeshCatalog) ListDisconnectedProxies() map[certificate.CommonName]time.Time {
	proxies := make(map[certificate.CommonName]time.Time)
	mc.disconnectedProxies.Range(func(cnInterface, disconnectedProxyInterface interface{}) bool {
		cn := cnInterface.(certificate.CommonName)
		props := disconnectedProxyInterface.(disconnectedProxy)
		proxies[cn] = props.lastSeen
		return true // continue the iteration
	})
	return proxies
}

// ListSMIPolicies returns all policies OSM is aware of.
func (mc *MeshCatalog) ListSMIPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.K8sServiceAccount, []*spec.HTTPRouteGroup, []*access.TrafficTarget) {
	trafficSplits := mc.meshSpec.ListTrafficSplits()
	splitServices := mc.meshSpec.ListTrafficSplitServices()
	serviceAccounts := mc.meshSpec.ListServiceAccounts()
	trafficSpecs := mc.meshSpec.ListHTTPTrafficSpecs()
	trafficTargets := mc.meshSpec.ListTrafficTargets()

	return trafficSplits, splitServices, serviceAccounts, trafficSpecs, trafficTargets
}

// ListMonitoredNamespaces returns all namespaces that the mesh is monitoring.
func (mc *MeshCatalog) ListMonitoredNamespaces() []string {
	namespaces, err := mc.kubeController.ListMonitoredNamespaces()

	if err != nil {
		return nil
	}

	return namespaces
}
