package debugger

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha2"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/tests"
)

func TestEndpoints(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Suite")
}

var _ = Describe("Test debugger methods", func() {
	Context("Testing getSMIPoliciesHandler()", func() {
		It("returns JSON serialized SMI policies", func() {
			mc := NewFakeMeshCatalogDebugger()
			ds := debugServer{
				meshCatalogDebugger: mc,
			}
			smiPoliciesHandler := ds.getSMIPoliciesHandler()
			responseRecorder := httptest.NewRecorder()
			smiPoliciesHandler.ServeHTTP(responseRecorder, nil)
			actualResponseBody := responseRecorder.Body.String()
			expectedResponseBody := `{"traffic_splits":["bar"],"split_services":["default/bookstore"],"service_accounts":["default/bookbuyer-serviceaccount"],"traffic_specs":["bookstore-service-routes"],"traffic_targets":["bookbuyer-access-bookstore"],"services":["bar"]}`
			Expect(actualResponseBody).To(Equal(expectedResponseBody), fmt.Sprintf("Actual value did not match expectations:\n%s", actualResponseBody))
		})
	})
})

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
func (f fakeMeshCatalogDebuger) ListSMIPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.NamespacedServiceAccount, []*spec.HTTPRouteGroup, []*target.TrafficTarget, []*corev1.Service) {
	return []*split.TrafficSplit{{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "bar",
			}},
		},
		[]service.WeightedService{
			tests.WeightedService,
		},
		[]service.NamespacedServiceAccount{
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
