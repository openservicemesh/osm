package debugger

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha3"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
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
			expectedResponseBody := `{"traffic_splits":[{"metadata":{"name":"bar","namespace":"foo","creationTimestamp":null},"spec":{}}],"weighted_services":[{"service_name:omitempty":{"Namespace":"default","Name":"bookstore"},"weight:omitempty":100,"root_service:omitempty":"bookstore-apex"}],"service_accounts":[{"Namespace":"default","Name":"bookbuyer"}],"route_groups":[{"kind":"HTTPRouteGroup","apiVersion":"specs.smi-spec.io/v1alpha2","metadata":{"name":"bookstore-service-routes","namespace":"default","creationTimestamp":null},"spec":{"matches":[{"name":"buy-books","methods":["GET"],"pathRegex":"/buy","headers":[{"user-agent":"test-UA"}]},{"name":"sell-books","methods":["GET"],"pathRegex":"/sell","headers":[{"user-agent":"test-UA"}]},{"name":"allow-everything-on-header","headers":[{"user-agent":"test-UA"}]}]}}],"traffic_targets":[{"kind":"TrafficTarget","apiVersion":"access.smi-spec.io/v1alpha2","metadata":{"name":"bookbuyer-access-bookstore","namespace":"default","creationTimestamp":null},"spec":{"destination":{"kind":"Name","name":"bookstore","namespace":"default"},"sources":[{"kind":"Name","name":"bookbuyer","namespace":"default"}],"rules":[{"kind":"HTTPRouteGroup","name":"bookstore-service-routes","matches":["buy-books","sell-books"]}]}}]}`
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
func (f fakeMeshCatalogDebuger) ListSMIPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.K8sServiceAccount, []*spec.HTTPRouteGroup, []*target.TrafficTarget) {
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
		}
}

// ListMonitoredNamespaces implements MeshCatalogDebugger
func (f fakeMeshCatalogDebuger) ListMonitoredNamespaces() []string {
	return []string{tests.Namespace}
}

// NewFakeMeshCatalogDebugger implements and creates a new MeshCatalogDebugger
func NewFakeMeshCatalogDebugger() MeshCatalogDebugger {
	return fakeMeshCatalogDebuger{}
}
