package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestK8sSvcToMeshSvc(t *testing.T) {
	assert := assert.New(t)

	v1Service := tests.NewServiceFixture(tests.BookstoreServiceName, tests.Namespace, nil)
	meshSvc := K8sSvcToMeshSvc(v1Service)
	expectedMeshSvc := service.MeshService{
		Name:      tests.BookstoreServiceName,
		Namespace: tests.Namespace,
	}

	assert.Equal(meshSvc, expectedMeshSvc)

}

func TestGetTrafficTargetName(t *testing.T) {
	assert := assert.New(t)

	type getTrafficTargetNameTest struct {
		input              string
		expectedTargetName string
	}

	getTrafficTargetNameTests := []getTrafficTargetNameTest{
		{"TrafficTarget", "TrafficTarget:default/bookbuyer->default/bookstore"},
		{"", "default/bookbuyer->default/bookstore"},
	}

	for _, tn := range getTrafficTargetNameTests {
		trafficTargetName := GetTrafficTargetName(tn.input, tests.BookbuyerService, tests.BookstoreService)

		assert.Equal(trafficTargetName, tn.expectedTargetName)
	}
}
