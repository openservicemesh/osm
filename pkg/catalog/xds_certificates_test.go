package catalog

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test XDS certificate tooling", func() {

	kubeClient := testclient.NewSimpleClientset()

	mc := NewFakeMeshCatalog(kubeClient)
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", tests.EnvoyUID, tests.BookstoreServiceAccountName, tests.Namespace))

	Context("Test GetServiceFromEnvoyCertificate()", func() {
		It("works as expected", func() {
			pod := tests.NewPodTestFixtureWithOptions(tests.Namespace, "pod-name", tests.BookstoreServiceAccountName)
			_, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.ServiceAccountName).To(Equal(tests.BookstoreServiceAccountName))

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, tests.Namespace, selector)
			_, err = kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			nsService, err := mc.GetServiceFromEnvoyCertificate(cn)
			Expect(err).ToNot(HaveOccurred())
			expected := service.NamespacedService{
				Namespace: tests.Namespace,
				Service:   svcName,
			}
			Expect(nsService).To(Equal(&expected))
		})

		It("returns an error with an invalid CN", func() {
			service, err := mc.GetServiceFromEnvoyCertificate("getAllowedDirectionalServices")
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

			podCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", envoyUID, tests.BookstoreServiceAccountName, namespace))
			nsService, err := mc.GetServiceFromEnvoyCertificate(podCN)
			Expect(err).ToNot(HaveOccurred())

			expected := service.NamespacedService{
				Namespace: namespace,
				Service:   svcName,
			}
			Expect(nsService).To(Equal(&expected))
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("works as expected", func() {
			envoyUID := uuid.New().String()
			someOtherEnvoyUID := uuid.New().String()
			namespace := uuid.New().String()

			// Ensure correct presetup
			pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(0))

			{
				newPod0 := tests.NewPodTestFixture(namespace, fmt.Sprintf("pod-0-%s", uuid.New()))
				newPod0.Labels[constants.EnvoyUniqueIDLabelName] = someOtherEnvoyUID
				_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod0, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			newPod1 := tests.NewPodTestFixture(namespace, fmt.Sprintf("pod-1-%s", uuid.New()))
			newPod1.Labels[constants.EnvoyUniqueIDLabelName] = envoyUID
			_, err = kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod1, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			{
				newPod2 := tests.NewPodTestFixture(namespace, fmt.Sprintf("pod-2-%s", uuid.New()))
				newPod2.Labels[constants.EnvoyUniqueIDLabelName] = someOtherEnvoyUID
				_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod2, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			// Ensure correct setup
			pods, err = kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(3))

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", envoyUID, tests.BookstoreServiceAccountName, namespace))
			actualPod, err := GetPodFromCertificate(newCN, kubeClient)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualPod.Name).To(Equal(newPod1.Name))
			Expect(actualPod).To(Equal(&newPod1))
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails with invalid certificate", func() {
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

			// No service account in this CN
			newCN := certificate.CommonName(fmt.Sprintf("%s.%s", envoyUID, namespace))
			actualPod, err := GetPodFromCertificate(newCN, kubeClient)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errInvalidCertificateCN))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test GetPodFromCertificate()", func() {
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

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", envoyUID, tests.BookstoreServiceAccountName, namespace))
			actualPod, err := GetPodFromCertificate(newCN, kubeClient)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errMoreThanOnePodForCertificate))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails when service account does not match certificate", func() {
			namespace := uuid.New().String()
			envoyUID := uuid.New().String()
			randomServiceAccount := uuid.New().String()

			podName := uuid.New().String()
			newPod := tests.NewPodTestFixture(namespace, podName)
			newPod.Labels[constants.EnvoyUniqueIDLabelName] = envoyUID

			_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(newPod.Spec.ServiceAccountName).ToNot(Equal(randomServiceAccount))
			Expect(newPod.Spec.ServiceAccountName).To(Equal(tests.BookstoreServiceAccountName))

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", envoyUID, randomServiceAccount, namespace))
			actualPod, err := GetPodFromCertificate(newCN, kubeClient)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errServiceAccountDoesNotMatchCertificate))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails when namespace does not match certificate", func() {
			namespace := uuid.New().String()
			envoyUID := uuid.New().String()
			someOtherRandomNamespace := uuid.New().String()

			podName := uuid.New().String()
			newPod := tests.NewPodTestFixture(namespace, podName)
			newPod.Labels[constants.EnvoyUniqueIDLabelName] = envoyUID

			_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s", envoyUID, tests.BookstoreServiceAccountName, someOtherRandomNamespace))
			actualPod, err := GetPodFromCertificate(newCN, kubeClient)
			Expect(err).To(HaveOccurred())
			// Since the namespace on the certificate is different than where the pod is...
			Expect(err).To(Equal(errDidNotFindPodForCertificate))
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
			serviceAccount := uuid.New().String()

			cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyID, serviceAccount, testNamespace))

			cnMeta, err := getCertificateCommonNameMeta(cn)
			Expect(err).ToNot(HaveOccurred())

			expected := &certificateCommonNameMeta{
				ProxyID:        proxyID,
				ServiceAccount: serviceAccount,
				Namespace:      testNamespace,
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
			serviceAccount := uuid.New().String()
			namespace := uuid.New().String()

			cn := NewCertCommonNameWithProxyID(proxyID, serviceAccount, namespace)
			Expect(cn).To(Equal(certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyID, serviceAccount, namespace))))

			actualMeta, err := getCertificateCommonNameMeta(cn)
			expectedMeta := certificateCommonNameMeta{
				ProxyID:        proxyID,
				ServiceAccount: serviceAccount,
				Namespace:      namespace,
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(actualMeta).To(Equal(&expectedMeta))
		})
	})

	Context("Test filterTrafficSplitServices()", func() {
		It("returns services except these to be traffic split", func() {

			services := []v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo",
						Name:      "A",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: tests.TrafficSplit.Namespace,
						Name:      tests.TrafficSplit.Spec.Service,
					},
				},
			}

			expected := []v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo",
						Name:      "A",
					},
				},
			}

			actual := mc.filterTrafficSplitServices(services)

			Expect(actual).To(Equal(expected))
		})
	})
})
