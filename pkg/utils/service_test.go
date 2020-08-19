package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Testing utils helpers", func() {
	Context("Test K8sSvcToMeshSvc", func() {
		It("works as expected", func() {
			v1Service := tests.NewServiceFixture(tests.BookstoreServiceName, tests.Namespace, nil)
			meshSvc := K8sSvcToMeshSvc(v1Service)
			expectedMeshSvc := service.MeshService{
				Name:      tests.BookstoreServiceName,
				Namespace: tests.Namespace,
			}
			Expect(meshSvc).To(Equal(expectedMeshSvc))
		})
	})
})
