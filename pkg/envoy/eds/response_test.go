package eds

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test EDS response", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	kubeClient := testclient.NewSimpleClientset()
	catalog := catalog.NewFakeMeshCatalog(kubeClient)

	Context("Test eds.NewResponse", func() {
		It("Correctly returns an response for endpoints when the certificate and service are valid", func() {
			// Initialize the proxy service
			proxyServiceName := tests.BookbuyerServiceName
			proxyServiceAccountName := tests.BookbuyerServiceAccountName
			proxyUUID := uuid.New()

			// The format of the CN matters
			xdsCertificate := certificate.CommonName(fmt.Sprintf("%s.%s.%s.foo.bar", proxyUUID, proxyServiceAccountName, tests.Namespace))
			certSerialNumber := certificate.SerialNumber("123456")
			proxy := envoy.NewProxy(xdsCertificate, certSerialNumber, nil)

			{
				// Create a pod to match the CN
				podName := fmt.Sprintf("pod-0-%s", uuid.New())
				pod := tests.NewPodFixture(tests.Namespace, podName, proxyServiceAccountName, tests.PodLabels)

				pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String() // This is what links the Pod and the Certificate
				_, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			{
				// Create a service for the pod created above
				selectors := map[string]string{
					// These need to match the POD created above
					tests.SelectorKey: tests.SelectorValue,
				}
				// The serviceName must match the SMI
				service := tests.NewServiceFixture(proxyServiceName, tests.Namespace, selectors)
				_, err := kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			_, err := NewResponse(catalog, proxy, nil, mockConfigurator, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Correctly returns an error response for endpoints when the proxy isn't associated with a MeshService", func() {
			// Initialize the proxy service
			proxyServiceAccountName := "non-existent-service-account"
			proxyUUID := uuid.New()

			// The format of the CN matters
			xdsCertificate := certificate.CommonName(fmt.Sprintf("%s.%s.%s.foo.bar", proxyUUID, proxyServiceAccountName, tests.Namespace))
			certSerialNumber := certificate.SerialNumber("123456")
			proxy := envoy.NewProxy(xdsCertificate, certSerialNumber, nil)

			// Don't create a pod/service for this proxy, this should result in an error when the
			// service is being looked up based on the proxy's certificate

			_, err := NewResponse(catalog, proxy, nil, mockConfigurator, nil)
			Expect(err).To(HaveOccurred())
		})
	})
})
