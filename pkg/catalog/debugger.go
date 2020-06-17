package catalog

import (
	"github.com/open-service-mesh/osm/pkg/service"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha2"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
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
func (mc *MeshCatalog) ListConnectedProxies() map[certificate.CommonName]time.Time {
	proxies := make(map[certificate.CommonName]time.Time)
	mc.connectedProxiesLock.Lock()
	mc.disconnectedProxiesLock.Lock()
	for cn, props := range mc.connectedProxies {
		if _, ok := mc.disconnectedProxies[cn]; ok {
			continue
		}
		proxies[cn] = props.connectedAt
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

// ListPolicies returns all policies OSM is aware of.
func (mc *MeshCatalog) ListPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.NamespacedServiceAccount, []*spec.HTTPRouteGroup, []*target.TrafficTarget, []*corev1.Service) {
	trafficSplits := mc.meshSpec.ListTrafficSplits()
	splitServices := mc.meshSpec.ListTrafficSplitServices()
	serviceAccouns := mc.meshSpec.ListServiceAccounts()
	trafficSpecs := mc.meshSpec.ListHTTPTrafficSpecs()
	trafficTargets := mc.meshSpec.ListTrafficTargets()
	services, err := mc.meshSpec.ListServices()
	if err != nil {
		services = nil
	}

	return trafficSplits, splitServices, serviceAccouns, trafficSpecs, trafficTargets, services
}
