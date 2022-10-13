package debugger

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/catalog"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/tests"
)

// Tests TestGetSMIPolicies through HTTP handler returns the list of SMI policies extracted from MeshCatalog
// in string format
func TestGetSMIPolicies(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mock := compute.NewMockInterface(mockCtrl)
	stop := make(chan struct{})
	meshCatalog := catalog.NewMeshCatalog(
		mock,
		tresorFake.NewFake(time.Hour),
		stop,
		messaging.NewBroker(stop),
	)

	ds := DebugConfig{
		meshCatalog: meshCatalog,
	}

	mock.EXPECT().ListTrafficSplits().Return(
		[]*split.TrafficSplit{
			{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "bar",
				}},
		})
	mock.EXPECT().IsMonitoredNamespace(gomock.Any()).Return(true).AnyTimes()
	mock.EXPECT().ListHTTPTrafficSpecs().Return(
		[]*spec.HTTPRouteGroup{
			&tests.HTTPRouteGroup,
		}).AnyTimes()
	mock.EXPECT().ListTrafficTargets().Return(
		[]*access.TrafficTarget{
			&tests.TrafficTarget,
		}).AnyTimes()

	smiPoliciesHandler := ds.getSMIPoliciesHandler()
	responseRecorder := httptest.NewRecorder()
	smiPoliciesHandler.ServeHTTP(responseRecorder, nil)
	actualResponseBody := responseRecorder.Body.String()
	expectedResponseBody := `{"traffic_splits":[{"metadata":{"name":"bar","namespace":"foo","creationTimestamp":null},"spec":{}}],"service_accounts":[{"Namespace":"default","Name":"bookbuyer"},{"Namespace":"default","Name":"bookstore"}],"route_groups":[{"kind":"HTTPRouteGroup","apiVersion":"specs.smi-spec.io/v1alpha4","metadata":{"name":"bookstore-service-routes","namespace":"default","creationTimestamp":null},"spec":{"matches":[{"name":"buy-books","methods":["GET"],"pathRegex":"/buy","headers":[{"user-agent":"test-UA"}]},{"name":"sell-books","methods":["GET"],"pathRegex":"/sell","headers":[{"user-agent":"test-UA"}]},{"name":"allow-everything-on-header","headers":[{"user-agent":"test-UA"}]}]}}],"traffic_targets":[{"kind":"TrafficTarget","apiVersion":"access.smi-spec.io/v1alpha3","metadata":{"name":"bookbuyer-access-bookstore","namespace":"default","creationTimestamp":null},"spec":{"destination":{"kind":"ServiceAccount","name":"bookstore","namespace":"default"},"sources":[{"kind":"ServiceAccount","name":"bookbuyer","namespace":"default"}],"rules":[{"kind":"HTTPRouteGroup","name":"bookstore-service-routes","matches":["buy-books","sell-books"]}]}}]}`
	assert.Equal(expectedResponseBody, actualResponseBody, "Actual value did not match expectations:\n%s", actualResponseBody)
}
