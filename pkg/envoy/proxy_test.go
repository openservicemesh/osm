package envoy

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test proxy methods", func() {
	proxyUUID := uuid.New()
	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.svc-acc.namespace", proxyUUID, KindSidecar))
	certSerialNumber := certificate.SerialNumber("123456")
	podUID := uuid.New().String()
	proxy, err := NewProxy(certCommonName, certSerialNumber, tests.NewMockAddress("1.2.3.4"))

	Context("Proxy is valid", func() {
		Expect(proxy).ToNot((BeNil()))
		Expect(err).ToNot(HaveOccurred())
	})

	Context("test GetLastAppliedVersion()", func() {
		It("returns correct values", func() {
			actual := proxy.GetLastAppliedVersion(TypeCDS)
			Expect(actual).To(Equal(uint64(0)))

			proxy.SetLastAppliedVersion(TypeCDS, uint64(345))

			actual = proxy.GetLastAppliedVersion(TypeCDS)
			Expect(actual).To(Equal(uint64(345)))
		})
	})

	Context("test GetLastSentNonce()", func() {
		It("returns empty if nonce doesn't exist", func() {
			res := proxy.GetLastSentNonce(TypeCDS)
			Expect(res).To(Equal(""))
		})

		It("returns correct values if nonce exists", func() {
			proxy.SetNewNonce(TypeCDS)

			firstNonce := proxy.GetLastSentNonce(TypeCDS)
			Expect(firstNonce).ToNot(Equal(uint64(0)))
			// Platform(Windows): Sleep to accommodate `time.Now()` lower accuracy.
			if runtime.GOOS == constants.OSWindows {
				time.Sleep(1 * time.Millisecond)
			}
			proxy.SetNewNonce(TypeCDS)

			secondNonce := proxy.GetLastSentNonce(TypeCDS)
			Expect(secondNonce).ToNot(Equal(firstNonce))
		})
	})

	Context("test GetLastSentVersion()", func() {
		It("returns correct values", func() {
			actual := proxy.GetLastSentVersion(TypeCDS)
			Expect(actual).To(Equal(uint64(0)))

			newVersion := uint64(132)
			proxy.SetLastSentVersion(TypeCDS, newVersion)

			actual = proxy.GetLastSentVersion(TypeCDS)
			Expect(actual).To(Equal(newVersion))

			proxy.IncrementLastSentVersion(TypeCDS)
			actual = proxy.GetLastSentVersion(TypeCDS)
			Expect(actual).To(Equal(newVersion + 1))
		})
	})

	Context("test GetConnectedAt()", func() {
		It("returns correct values", func() {
			actual := proxy.GetConnectedAt()
			Expect(actual).ToNot(Equal(uint64(0)))
		})
	})

	Context("test GetIP()", func() {
		It("returns correct values", func() {
			actual := proxy.GetIP()
			Expect(actual.Network()).To(Equal("mockNetwork"))
			Expect(actual.String()).To(Equal("1.2.3.4"))
		})
	})

	Context("test HasPodMetadata()", func() {
		It("returns correct values", func() {
			actual := proxy.HasPodMetadata()
			Expect(actual).To(BeFalse())
		})
	})

	Context("test UUID", func() {
		It("returns correct values", func() {
			Expect(proxy.UUID).To(Equal(proxyUUID))
		})
	})

	Context("test StatsHeaders()", func() {
		It("returns correct values", func() {
			actual := proxy.StatsHeaders()
			expected := map[string]string{
				"osm-stats-namespace": "unknown",
				"osm-stats-kind":      "unknown",
				"osm-stats-name":      "unknown",
				"osm-stats-pod":       "unknown",
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("test correctness proxy object creation", func() {
		It("returns correct values", func() {
			Expect(proxy.GetCertificateCommonName()).To(Equal(certCommonName))
			Expect(proxy.GetCertificateSerialNumber()).To(Equal(certSerialNumber))
			Expect(proxy.HasPodMetadata()).To(BeFalse())

			proxy.PodMetadata = &PodMetadata{
				UID: podUID,
			}

			Expect(proxy.HasPodMetadata()).To(BeTrue())
			Expect(proxy.PodMetadata.UID).To(Equal(podUID))
			Expect(strings.Contains(proxy.String(), "[Serial=123456]")).To(BeTrue())
		})
	})
})

func TestStatsHeaders(t *testing.T) {
	const unknown = "unknown"
	tests := []struct {
		name     string
		proxy    Proxy
		expected map[string]string
	}{
		{
			name: "nil metadata",
			proxy: Proxy{
				PodMetadata: nil,
			},
			expected: map[string]string{
				"osm-stats-kind":      unknown,
				"osm-stats-name":      unknown,
				"osm-stats-namespace": unknown,
				"osm-stats-pod":       unknown,
			},
		},
		{
			name: "empty metadata",
			proxy: Proxy{
				PodMetadata: &PodMetadata{},
			},
			expected: map[string]string{
				"osm-stats-kind":      unknown,
				"osm-stats-name":      unknown,
				"osm-stats-namespace": unknown,
				"osm-stats-pod":       unknown,
			},
		},
		{
			name: "full metadata",
			proxy: Proxy{
				PodMetadata: &PodMetadata{
					Name:         "pod",
					Namespace:    "ns",
					WorkloadKind: "kind",
					WorkloadName: "name",
				},
			},
			expected: map[string]string{
				"osm-stats-kind":      "kind",
				"osm-stats-name":      "name",
				"osm-stats-namespace": "ns",
				"osm-stats-pod":       "pod",
			},
		},
		{
			name: "replicaset with expected name format",
			proxy: Proxy{
				PodMetadata: &PodMetadata{
					WorkloadKind: "ReplicaSet",
					WorkloadName: "some-name-randomchars",
				},
			},
			expected: map[string]string{
				"osm-stats-kind":      "Deployment",
				"osm-stats-name":      "some-name",
				"osm-stats-namespace": unknown,
				"osm-stats-pod":       unknown,
			},
		},
		{
			name: "replicaset without expected name format",
			proxy: Proxy{
				PodMetadata: &PodMetadata{
					WorkloadKind: "ReplicaSet",
					WorkloadName: "name",
				},
			},
			expected: map[string]string{
				"osm-stats-kind":      "ReplicaSet",
				"osm-stats-name":      "name",
				"osm-stats-namespace": unknown,
				"osm-stats-pod":       unknown,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.proxy.StatsHeaders()
			assert.Equal(t, test.expected, actual)
		})
	}
}

var _ = Describe("Test XDS certificate tooling", func() {
	mockCtrl := gomock.NewController(ginkgo.GinkgoT())
	kubeClient := fake.NewSimpleClientset()

	Context("Test GetPodFromCertificate()", func() {
		It("works as expected", func() {
			proxyUUID := uuid.New()
			someOtherEnvoyUID := uuid.New().String()
			namespace := uuid.New().String()
			mockKubeController := k8s.NewMockController(mockCtrl)
			podlabels := map[string]string{
				constants.AppLabel:               tests.SelectorValue,
				constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
			}
			someOthePodLabels := map[string]string{
				constants.AppLabel:               tests.SelectorValue,
				constants.EnvoyUniqueIDLabelName: someOtherEnvoyUID,
			}

			// Ensure correct presetup
			pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(0))

			newPod0 := tests.NewPodFixture(namespace, fmt.Sprintf("pod-0-%s", uuid.New()), tests.BookstoreServiceAccountName, someOthePodLabels)
			_, err = kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod0, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newPod1 := tests.NewPodFixture(namespace, fmt.Sprintf("pod-1-%s", uuid.New()), tests.BookstoreServiceAccountName, podlabels)
			_, err = kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod1, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newPod2 := tests.NewPodFixture(namespace, fmt.Sprintf("pod-2-%s", uuid.New()), tests.BookstoreServiceAccountName, someOthePodLabels)
			_, err = kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod2, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Ensure correct setup
			pods, err = kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(3))

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", proxyUUID, KindSidecar, tests.BookstoreServiceAccountName, namespace))

			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&newPod0, &newPod1, &newPod2})
			actualPod, err := GetPodFromCertificate(newCN, mockKubeController)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualPod.Name).To(Equal(newPod1.Name))
			Expect(actualPod).To(Equal(&newPod1))
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails with invalid certificate", func() {
			namespace := uuid.New().String()
			proxyUUID := uuid.New()
			mockKubeController := k8s.NewMockController(mockCtrl)

			// Create a pod with the same certificateCN twice
			for range []int{0, 1} {
				podName := uuid.New().String()
				newPod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
				newPod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()

				_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			// No service account in this CN
			newCN := certificate.CommonName(fmt.Sprintf("%s.%s", proxyUUID, namespace))
			actualPod, err := GetPodFromCertificate(newCN, mockKubeController)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(ErrInvalidCertificateCN))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails with two pods with same cert", func() {
			namespace := uuid.New().String()
			proxyUUID := uuid.New()
			mockKubeController := k8s.NewMockController(mockCtrl)

			// Create a pod with the same certificateCN twice
			var pods []*v1.Pod
			for range []int{0, 1} {
				podName := uuid.New().String()
				tests.PodLabels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()
				newPod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
				pods = append(pods, &newPod)

				_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			mockKubeController.EXPECT().ListPods().Return(pods)
			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", proxyUUID, KindSidecar, tests.BookstoreServiceAccountName, namespace))
			actualPod, err := GetPodFromCertificate(newCN, mockKubeController)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(ErrMoreThanOnePodForCertificate))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails when service account does not match certificate", func() {
			namespace := uuid.New().String()
			proxyUUID := uuid.New()
			randomServiceAccount := uuid.New().String()
			mockKubeController := k8s.NewMockController(mockCtrl)

			podName := uuid.New().String()
			newPod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
			newPod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()

			_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(newPod.Spec.ServiceAccountName).ToNot(Equal(randomServiceAccount))
			Expect(newPod.Spec.ServiceAccountName).To(Equal(tests.BookstoreServiceAccountName))

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", proxyUUID, KindSidecar, randomServiceAccount, namespace))
			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&newPod})
			actualPod, err := GetPodFromCertificate(newCN, mockKubeController)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(ErrServiceAccountDoesNotMatchCertificate))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test GetPodFromCertificate()", func() {
		It("fails when namespace does not match certificate", func() {
			namespace := uuid.New().String()
			proxyUUID := uuid.New()
			someOtherRandomNamespace := uuid.New().String()
			mockKubeController := k8s.NewMockController(mockCtrl)

			podName := uuid.New().String()
			newPod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
			newPod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()

			_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", proxyUUID, KindSidecar, tests.BookstoreServiceAccountName, someOtherRandomNamespace))
			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&newPod})
			actualPod, err := GetPodFromCertificate(newCN, mockKubeController)
			Expect(err).To(HaveOccurred())
			// Since the namespace on the certificate is different than where the pod is...
			Expect(err).To(Equal(ErrDidNotFindPodForCertificate))
			Expect(actualPod).To(BeNil())
		})
	})

	Context("Test getCertificateCommonNameMeta()", func() {
		It("parses CN into certificateCommonNameMeta", func() {
			proxyUUID := uuid.New()
			testNamespace := uuid.New().String()
			serviceAccount := uuid.New().String()

			cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s.%s", proxyUUID, KindSidecar, serviceAccount, testNamespace, identity.ClusterLocalTrustDomain))

			cnMeta, err := getCertificateCommonNameMeta(cn)
			Expect(err).ToNot(HaveOccurred())

			expected := &certificateCommonNameMeta{
				ProxyUUID:       proxyUUID,
				ProxyKind:       KindSidecar,
				ServiceIdentity: identity.ServiceIdentity(fmt.Sprintf("%s.%s.%s", serviceAccount, testNamespace, identity.ClusterLocalTrustDomain)),
			}
			Expect(cnMeta).To(Equal(expected))
		})

		It("parses CN into certificateCommonNameMeta", func() {
			_, err := getCertificateCommonNameMeta("a")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test NewXDSCertCommonName() and getCertificateCommonNameMeta() together", func() {
		It("returns the the CommonName of the form <proxyID>.<kind>.<service-account>.<namespace>", func() {
			proxyUUID := uuid.New()
			serviceAccount := uuid.New().String()
			namespace := uuid.New().String()

			cn := NewXDSCertCommonName(proxyUUID, KindSidecar, serviceAccount, namespace)
			Expect(cn).To(Equal(certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s.%s", proxyUUID, KindSidecar, serviceAccount, namespace, identity.ClusterLocalTrustDomain))))

			actualMeta, err := getCertificateCommonNameMeta(cn)
			expectedMeta := certificateCommonNameMeta{
				ProxyUUID:       proxyUUID,
				ProxyKind:       KindSidecar,
				ServiceIdentity: identity.ServiceIdentity(fmt.Sprintf("%s.%s.%s", serviceAccount, namespace, identity.ClusterLocalTrustDomain)),
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(actualMeta).To(Equal(&expectedMeta))
		})
	})

	Context("Test GetServiceIdentityFromProxyCertificate", func() {
		It("should correctly return the ServiceAccount encoded in the XDS certificate CN", func() {
			cn := certificate.CommonName(fmt.Sprintf("%s.sidecar.sa-name.sa-namespace.cluster.local", uuid.New()))
			proxyIdentity, err := GetServiceIdentityFromProxyCertificate(cn)
			Expect(err).ToNot(HaveOccurred())
			Expect(proxyIdentity).To(Equal(identity.ServiceIdentity("sa-name.sa-namespace.cluster.local")))
		})

		It("should correctly error when the XDS certificate CN is invalid", func() {
			proxyIdentity, err := GetServiceIdentityFromProxyCertificate(certificate.CommonName("invalid"))
			Expect(err).To(HaveOccurred())
			Expect(proxyIdentity).To(Equal(identity.ServiceIdentity("")))
		})
	})
})

func TestPodMetadataString(t *testing.T) {
	testCases := []struct {
		name     string
		proxy    *Proxy
		expected string
	}{
		{
			name: "with valid pod metadata",
			proxy: &Proxy{
				PodMetadata: &PodMetadata{
					UID:            "some-UID",
					Namespace:      "some-ns",
					Name:           "some-pod",
					ServiceAccount: identity.K8sServiceAccount{Name: "some-service-account"},
				},
			},
			expected: "UID=some-UID, Namespace=some-ns, Name=some-pod, ServiceAccount=some-service-account",
		},
		{
			name: "no pod metadata",
			proxy: &Proxy{
				PodMetadata: nil,
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := tc.proxy.PodMetadataString()
			assert.Equal(tc.expected, actual)
		})
	}
}

func TestSubscribedResources(t *testing.T) {
	assert := tassert.New(t)

	p := Proxy{
		subscribedResources: make(map[TypeURI]mapset.Set),
	}

	res := p.GetSubscribedResources("test")
	assert.Zero(res.Cardinality())

	p.SetSubscribedResources(TypeRDS, mapset.NewSetWith("A", "B", "C"))

	res = p.GetSubscribedResources(TypeRDS)
	assert.Equal(res.Cardinality(), 3)
	assert.True(res.Contains("A"))
	assert.True(res.Contains("B"))
	assert.True(res.Contains("C"))
}
