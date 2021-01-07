package utils

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestK8sSvcToMeshSvc(t *testing.T) {
	assert := tassert.New(t)

	v1Service := tests.NewServiceFixture(tests.BookstoreV1ServiceName, tests.Namespace, nil)
	meshSvc := K8sSvcToMeshSvc(v1Service)
	expectedMeshSvc := service.MeshService{
		Name:      tests.BookstoreV1ServiceName,
		Namespace: tests.Namespace,
	}

	assert.Equal(meshSvc, expectedMeshSvc)
}

func TestGetTrafficTargetName(t *testing.T) {
	assert := tassert.New(t)

	type getTrafficTargetNameTest struct {
		input              string
		expectedTargetName string
	}

	getTrafficTargetNameTests := []getTrafficTargetNameTest{
		{"TrafficTarget", "TrafficTarget:default/bookbuyer->default/bookstore-v1"},
		{"", "default/bookbuyer->default/bookstore-v1"},
	}

	for _, tn := range getTrafficTargetNameTests {
		trafficTargetName := GetTrafficTargetName(tn.input, tests.BookbuyerService, tests.BookstoreV1Service)

		assert.Equal(trafficTargetName, tn.expectedTargetName)
	}
}
