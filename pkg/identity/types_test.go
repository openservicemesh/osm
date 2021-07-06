package identity

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test pkg/service functions", func() {
	defer GinkgoRecover()

	Context("Test K8sServiceAccount struct methods", func() {
		namespace := uuid.New().String()
		serviceAccountName := uuid.New().String()
		sa := K8sServiceAccount{
			Namespace: namespace,
			Name:      serviceAccountName,
		}

		It("implements stringer interface correctly", func() {
			Expect(sa.String()).To(Equal(fmt.Sprintf("%s/%s", namespace, serviceAccountName)))
		})

		It("implements IsEmpty correctly", func() {
			Expect(sa.IsEmpty()).To(BeFalse())
			Expect(K8sServiceAccount{}.IsEmpty()).To(BeTrue())
		})

		It("implements K8sServiceAccount{}.ToServiceIdentity() correctly", func() {
			actual := K8sServiceAccount{
				Namespace: "ns",
				Name:      "name",
			}.ToServiceIdentity()
			expected := ServiceIdentity{"name", "ns", "cluster.local"}
			Expect(actual).To(Equal(expected))
		})

		It("implements ServiceIdentity{}.ToK8sServiceAccount() correctly", func() {
			actual := ServiceIdentity{"name", "ns", "cluster.local"}.ToK8sServiceAccount()
			expected := K8sServiceAccount{
				Namespace: "ns",
				Name:      "name",
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
