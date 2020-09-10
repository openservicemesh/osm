package catalog

import (
	"time"

	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha3"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

// ListExpectedProxies lists the Envoy proxies yet to connect and the time their XDS certificate was issued.
func (mc *MeshCatalog) ListExpectedProxies() map[certificate.CommonName]time.Time {
	proxies := make(map[certificate.CommonName]time.Time)
	mc.expectedProxiesLock.Lock()
	mc.connectedProxiesLock.Lock()
	mc.disconnectedProxiesLock.Lock()
	for cn, props := range mc.expectedProxies {
		if _, ok := mc.connectedProxies[cn]; ok {
			continue
		}
		if _, ok := mc.disconnectedProxies[cn]; ok {
			continue
		}
		proxies[cn] = props.certificateIssuedAt
	}
	mc.disconnectedProxiesLock.Unlock()
	mc.connectedProxiesLock.Unlock()
	mc.expectedProxiesLock.Unlock()
	return proxies
}

// ListConnectedProxies lists the Envoy proxies already connected and the time they first connected.
func (mc *MeshCatalog) ListConnectedProxies() map[certificate.CommonName]*envoy.Proxy {
	proxies := make(map[certificate.CommonName]*envoy.Proxy)
	mc.connectedProxiesLock.Lock()
	mc.disconnectedProxiesLock.Lock()
	for cn, props := range mc.connectedProxies {
		if _, ok := mc.disconnectedProxies[cn]; ok {
			continue
		}
		proxies[cn] = props.proxy
	}
	mc.disconnectedProxiesLock.Unlock()
	mc.connectedProxiesLock.Unlock()
	return proxies
}

// ListDisconnectedProxies lists the Envoy proxies disconnected and the time last seen.
func (mc *MeshCatalog) ListDisconnectedProxies() map[certificate.CommonName]time.Time {
	proxies := make(map[certificate.CommonName]time.Time)
	mc.disconnectedProxiesLock.Lock()
	for cn, props := range mc.disconnectedProxies {
		proxies[cn] = props.lastSeen
	}
	mc.disconnectedProxiesLock.Unlock()
	return proxies
}

// ListSMIPolicies returns all policies OSM is aware of.
func (mc *MeshCatalog) ListSMIPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.K8sServiceAccount, []*spec.HTTPRouteGroup, []*target.TrafficTarget) {
	trafficSplits := mc.meshSpec.ListTrafficSplits()
	splitServices := mc.meshSpec.ListTrafficSplitServices()
	serviceAccouns := mc.meshSpec.ListServiceAccounts()
	trafficSpecs := mc.meshSpec.ListHTTPTrafficSpecs()
	trafficTargets := mc.meshSpec.ListTrafficTargets()

	return trafficSplits, splitServices, serviceAccouns, trafficSpecs, trafficTargets
}

// ListMonitoredNamespaces returns all namespaces that the mesh is monitoring.
func (mc *MeshCatalog) ListMonitoredNamespaces() []string {
	namespaces, err := mc.kubeController.ListMonitoredNamespaces()

	if err != nil {
		return nil
	}

	return namespaces
}
