package catalog

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("Test XDS certificate tooling", func() {

	kubeClient := testclient.NewSimpleClientset()

	mc := NewFakeMeshCatalog(kubeClient)
	cn := certificate.CommonName(fmt.Sprintf("%s.%s", tests.EnvoyUID, tests.Namespace))

	Context("Test GetServiceFromEnvoyCertificate()", func() {
		It("works as expected", func() {
			pod := tests.NewPodTestFixture(tests.Namespace, "pod-name")
			_, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, tests.Namespace, selector)
			_, err = kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			service, err := mc.GetServiceFromEnvoyCertificate(cn)
			Expect(err).ToNot(HaveOccurred())
			expected := endpoint.NamespacedService{
				Namespace: tests.Namespace,
				Service:   svcName,
			}
			Expect(service).To(Equal(&expected))
		})

		It("returns an error with an invalid CN", func() {
			service, err := mc.GetServiceFromEnvoyCertificate("blah")
			Expect(err).To(HaveOccurred())
			Expect(service).To(BeNil())
		})
	})

	Context("Test getServiceFromCertificate()", func() {
		It("works as expected", func() {

			// Create the POD
			envoyUID := uuid.New().String()
			namespace := uuid.New().String()
			podName := uuid.New().String()
			newPod := tests.NewPodTestFixture(namespace, podName)
			newPod.Labels[constants.EnvoyUniqueIDLabelName] = envoyUID
			newPod.Labels[tests.SelectorKey] = tests.SelectorValue

			_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, namespace, selector)
			_, err = kubeClient.CoreV1().Services(namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			podCN := certificate.CommonName(fmt.Sprintf("%s.%s", envoyUID, namespace))
			service, err := getServiceFromCertificate(podCN, kubeClient)
			Expect(err).ToNot(HaveOccurred())

			expected := endpoint.NamespacedService{
				Namespace: namespace,
				Service:   svcName,
			}
			Expect(service).To(Equal(&expected))
		})
	})

	Context("Test getPodFromCertificate()", func() {
		It("works as expected", func() {
			envoyUID := uuid.New().String()
			namespace := uuid.New().String()
			podName := uuid.New().String()
			newPod := tests.NewPodTestFixture(namespace, podName)
			newPod.Labels[constants.EnvoyUniqueIDLabelName] = envoyUID

			_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s", envoyUID, namespace))
			actualPod, err := getPodFromCertificate(newCN, kubeClient)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualPod.Name).To(Equal(newPod.Name))
			Expect(actualPod).To(Equal(&newPod))
		})
	})

	Context("Test getPodFromCertificate()", func() {
		It("fails with two pods with same cert", func() {
			namespace := uuid.New().String()
			envoyUID := uuid.New().String()

			// Create a pod with the same certificateCN twice
			for range []int{0, 1} {
				podName := uuid.New().String()
				newPod := tests.NewPodTestFixture(namespace, podName)
				newPod.Labels[constants.EnvoyUniqueIDLabelName] = envoyUID

				_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s", envoyUID, namespace))
			actualPod, err := getPodFromCertificate(newCN, kubeClient)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errMoreThanOnePodForCertificate))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test mapStringStringToSet()", func() {
		It("lists services for pod", func() {
			labels := map[string]string{constants.EnvoyUniqueIDLabelName: tests.EnvoyUID}
			stringSet := mapStringStringToSet(labels)
			Expect(stringSet.Cardinality()).To(Equal(1))
			Expect(stringSet.ToSlice()[0]).To(Equal("osm-envoy-uid:A-B-C-D"))
		})
	})

	Context("Test listServicesForPod()", func() {
		It("lists services for pod", func() {
			namespace := uuid.New().String()
			labels := map[string]string{constants.EnvoyUniqueIDLabelName: tests.EnvoyUID}

			var serviceNames []string
			const serviceName = "svc-name-1"
			{
				// Create a second service
				service := tests.NewServiceFixture(serviceName, namespace, labels)
				_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceNames = append(serviceNames, service.Name)
			}

			{
				// Create a second service
				service := tests.NewServiceFixture(uuid.New().String(), namespace, labels)
				_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceNames = append(serviceNames, service.Name)
			}

			pod := tests.NewPodTestFixture(namespace, "pod-name")
			actualSvcs, err := listServicesForPod(&pod, kubeClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(actualSvcs)).To(Equal(2))

			actualNames := []string{actualSvcs[0].Name, actualSvcs[1].Name}
			Expect(actualNames).To(Equal(serviceNames))
		})
	})

	Context("Test getCertificateCommonNameMeta()", func() {
		It("parses CN into certificateCommonNameMeta", func() {
			proxyID := uuid.New().String()
			testNamespace := uuid.New().String()

			cn := certificate.CommonName(fmt.Sprintf("%s.%s", proxyID, testNamespace))

			cnMeta, err := getCertificateCommonNameMeta(cn)
			Expect(err).ToNot(HaveOccurred())

			expected := &certificateCommonNameMeta{
				ProxyID:   proxyID,
				Namespace: testNamespace,
			}
			Expect(cnMeta).To(Equal(expected))
		})

		It("parses CN into certificateCommonNameMeta", func() {
			_, err := getCertificateCommonNameMeta("a")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test NewCertCommonNameWithProxyID() and getCertificateCommonNameMeta() together", func() {
		It("returns the the CommonName of the form <proxyID>.<namespace>", func() {

			proxyID := uuid.New().String()
			namespace := uuid.New().String()

			cn := NewCertCommonNameWithProxyID(proxyID, namespace)
			Expect(cn).To(Equal(certificate.CommonName(fmt.Sprintf("%s.%s", proxyID, namespace))))

			actualMeta, err := getCertificateCommonNameMeta(cn)
			expectedMeta := certificateCommonNameMeta{
				ProxyID:   proxyID,
				Namespace: namespace,
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(actualMeta).To(Equal(&expectedMeta))
		})
	})
})
