package debugger

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http/httptest"
)

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
			expectedResponseBody := `{"traffic_splits":[{"metadata":{"name":"bar","namespace":"foo","creationTimestamp":null},"spec":{}}],"weighted_services":[{"service_name:omitempty":{"Namespace":"default","Name":"bookstore"},"weight:omitempty":100,"domain:omitempty":"contoso.com"}],"service_accounts":[{"Namespace":"default","Name":"bookbuyer"}],"route_groups":[{"kind":"HTTPRouteGroup","apiVersion":"specs.smi-spec.io/v1alpha2","metadata":{"name":"bookstore-service-routes","namespace":"default","creationTimestamp":null},"matches":[{"name":"buy-books","methods":["GET"],"pathRegex":"/buy","headers":[{"host":"contoso.com"}]},{"name":"sell-books","methods":["GET"],"pathRegex":"/sell"},{"name":"allow-everything-on-header","headers":[{"host":"contoso.com"}]}]}],"traffic_targets":[{"kind":"TrafficTarget","apiVersion":"access.smi-spec.io/v1alpha1","metadata":{"name":"bookbuyer-access-bookstore","namespace":"default","creationTimestamp":null},"destination":{"kind":"Name","name":"bookstore","namespace":"default"},"sources":[{"kind":"Name","name":"bookbuyer","namespace":"default"}],"specs":[{"kind":"HTTPRouteGroup","name":"bookstore-service-routes","matches":["buy-books"]}]}],"services":[{"metadata":{"name":"bar","namespace":"foo","creationTimestamp":null},"spec":{},"status":{"loadBalancer":{}}}]}`
			Expect(actualResponseBody).To(Equal(expectedResponseBody), fmt.Sprintf("Actual value did not match expectations:\n%s", actualResponseBody))
		})
	})
})
