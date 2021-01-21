package debugger

import (
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

// Tests TestGetSMIPolicies through HTTP handler returns the list of SMI policies extracted from MeshCatalog
// in string format
func TestGetSMIPolicies(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mock := NewMockMeshCatalogDebugger(mockCtrl)

	ds := DebugConfig{
		meshCatalogDebugger: mock,
	}

	mock.EXPECT().ListSMIPolicies().Return(
		[]*split.TrafficSplit{
			{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar",
				}},
		},
		[]service.WeightedService{
			tests.BookstoreV1WeightedService,
			tests.BookstoreV2WeightedService,
		},
		[]service.K8sServiceAccount{
			tests.BookbuyerServiceAccount,
		},
		[]*spec.HTTPRouteGroup{
			&tests.HTTPRouteGroup,
		},
		[]*access.TrafficTarget{
			&tests.TrafficTarget,
		},
	)

	smiPoliciesHandler := ds.getSMIPoliciesHandler()
	responseRecorder := httptest.NewRecorder()
	smiPoliciesHandler.ServeHTTP(responseRecorder, nil)
	actualResponseBody := responseRecorder.Body.String()
	expectedResponseBody := `{"traffic_splits":[{"metadata":{"name":"bar","namespace":"foo","creationTimestamp":null},"spec":{}}],"weighted_services":[{"service_name:omitempty":{"Namespace":"default","Name":"bookstore-v1"},"weight:omitempty":90,"root_service:omitempty":"bookstore-apex"},{"service_name:omitempty":{"Namespace":"default","Name":"bookstore-v2"},"weight:omitempty":10,"root_service:omitempty":"bookstore-apex"}],"service_accounts":[{"Namespace":"default","Name":"bookbuyer"}],"route_groups":[{"kind":"HTTPRouteGroup","apiVersion":"specs.smi-spec.io/v1alpha4","metadata":{"name":"bookstore-service-routes","namespace":"default","creationTimestamp":null},"spec":{"matches":[{"name":"buy-books","methods":["GET"],"pathRegex":"/buy","headers":[{"user-agent":"test-UA"}]},{"name":"sell-books","methods":["GET"],"pathRegex":"/sell","headers":[{"user-agent":"test-UA"}]},{"name":"allow-everything-on-header","headers":[{"user-agent":"test-UA"}]}]}}],"traffic_targets":[{"kind":"TrafficTarget","apiVersion":"access.smi-spec.io/v1alpha3","metadata":{"name":"bookbuyer-access-bookstore","namespace":"default","creationTimestamp":null},"spec":{"destination":{"kind":"Name","name":"bookstore","namespace":"default"},"sources":[{"kind":"Name","name":"bookbuyer","namespace":"default"}],"rules":[{"kind":"HTTPRouteGroup","name":"bookstore-service-routes","matches":["buy-books","sell-books"]}]}}]}`
	assert.Equal(actualResponseBody, expectedResponseBody, "Actual value did not match expectations:\n%s", actualResponseBody)
}
