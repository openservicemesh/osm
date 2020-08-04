package debugger

import (
	"time"

	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha2"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

type fakeMeshCatalogDebuger struct{}

// ListExpectedProxies implements MeshCatalogDebugger
func (f fakeMeshCatalogDebuger) ListExpectedProxies() map[certificate.CommonName]time.Time {
	panic("implement me")
}

// ListConnectedProxies implements MeshCatalogDebugger
func (f fakeMeshCatalogDebuger) ListConnectedProxies() map[certificate.CommonName]*envoy.Proxy {
	panic("implement me")
}

// ListDisconnectedProxies implements MeshCatalogDebugger
func (f fakeMeshCatalogDebuger) ListDisconnectedProxies() map[certificate.CommonName]time.Time {
	panic("implement me")
}

// ListSMIPolicies implements MeshCatalogDebugger
func (f fakeMeshCatalogDebuger) ListSMIPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.K8sServiceAccount, []*spec.HTTPRouteGroup, []*target.TrafficTarget, []*corev1.Service) {
	return []*split.TrafficSplit{{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "bar",
			}},
		},
		[]service.WeightedService{
			tests.WeightedService,
		},
		[]service.K8sServiceAccount{
			tests.BookbuyerServiceAccount,
		},
		[]*spec.HTTPRouteGroup{
			&tests.HTTPRouteGroup,
		},
		[]*target.TrafficTarget{
			&tests.TrafficTarget,
		},
		[]*corev1.Service{{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "bar",
			},
		}}
}

// NewFakeMeshCatalogDebugger implements and creates a new MeshCatalogDebugger
func NewFakeMeshCatalogDebugger() MeshCatalogDebugger {
	return fakeMeshCatalogDebuger{}
}
