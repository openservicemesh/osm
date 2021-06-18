package service

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/constants"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test pkg/service functions", func() {
	defer GinkgoRecover()

	Context("Test ClusterName's String method", func() {
		clusterNameStr := uuid.New().String()
		cn := ClusterName(clusterNameStr)

		It("implements stringer correctly", func() {
			Expect(cn.String()).To(Equal(clusterNameStr))
		})
	})

	Context("Test MeshService's String method", func() {
		namespace := uuid.New().String()
		name := uuid.New().String()
		ms := MeshService{
			Namespace:     namespace,
			Name:          name,
			ClusterDomain: constants.LocalDomain,
		}

		It("implements stringer correctly", func() {
			Expect(ms.String()).To(Equal(fmt.Sprintf("%s/%s/%s", namespace, name, constants.ClusterDomain)))
		})
	})

})
