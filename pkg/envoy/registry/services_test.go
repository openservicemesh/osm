package registry

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

const (
	service1Name = "svc-name-1"
)

var _ = Describe("Test Proxy-Service mapping", func() {
	mockCtrl := gomock.NewController(ginkgo.GinkgoT())
	kubeClient := testclient.NewSimpleClientset()
	mockKubeController := k8s.NewMockController(mockCtrl)
	proxyRegistry := NewProxyRegistry(&KubeProxyServiceMapper{mockKubeController}, nil)

	Context("Test ListProxyServices()", func() {
		It("works as expected", func() {
			proxyUUID := uuid.New()

			pod := tests.NewPodFixture(tests.Namespace, "pod-name", tests.BookstoreServiceAccountName,
				map[string]string{
					constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
					constants.AppLabel:               tests.SelectorValue})
			Expect(pod.Spec.ServiceAccountName).To(Equal(tests.BookstoreServiceAccountName))
			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&pod}).Times(1)

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{constants.AppLabel: tests.SelectorValue}
			svc1 := tests.NewServiceFixture(svcName, tests.Namespace, selector)

			svcName2 := uuid.New().String()
			svc2 := tests.NewServiceFixture(svcName2, tests.Namespace, selector)
			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{svc1, svc2}).Times(1)

			expectedSvc1 := service.MeshService{
				Namespace: tests.Namespace,
				Name:      svcName,
				Port:      tests.ServicePort,
				Protocol:  "http",
			}

			expectedSvc2 := service.MeshService{
				Namespace: tests.Namespace,
				Name:      svcName2,
				Port:      tests.ServicePort,
				Protocol:  "http",
			}
			expectedList := []service.MeshService{expectedSvc1, expectedSvc2}

			mockKubeController.EXPECT().GetEndpoints(gomock.Any()).Return(nil, nil).Times(2)

			certCommonName := envoy.NewXDSCertCommonName(proxyUUID, envoy.KindSidecar, tests.BookstoreServiceAccountName, tests.Namespace)
			certSerialNumber := certificate.SerialNumber("123456")
			proxy, err := envoy.NewProxy(certCommonName, certSerialNumber, nil)
			Expect(err).ToNot(HaveOccurred())
			meshServices, err := proxyRegistry.ListProxyServices(proxy)
			Expect(err).ToNot(HaveOccurred())

			Expect(meshServices).Should(HaveLen(len(expectedList)))
			Expect(meshServices).Should(ConsistOf(expectedList))
		})
	})

	Context("Test getServiceFromCertificate()", func() {
		It("works as expected", func() {
			// Create the POD
			proxyUUID := uuid.New()
			namespace := uuid.New().String()
			podName := uuid.New().String()
			newPod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, tests.PodLabels)
			newPod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()

			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&newPod}).Times(1)

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{constants.AppLabel: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, namespace, selector)
			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{svc}).Times(1)

			podCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", proxyUUID, envoy.KindSidecar, tests.BookstoreServiceAccountName, namespace))
			certSerialNumber := certificate.SerialNumber("123456")
			newProxy, err := envoy.NewProxy(podCN, certSerialNumber, nil)
			Expect(err).ToNot(HaveOccurred())

			expected := service.MeshService{
				Namespace: namespace,
				Name:      svcName,
				Port:      tests.ServicePort,
				Protocol:  "http",
			}
			expectedList := []service.MeshService{expected}
			mockKubeController.EXPECT().GetEndpoints(gomock.Any()).Return(nil, nil)

			meshServices, err := proxyRegistry.ListProxyServices(newProxy)
			Expect(err).ToNot(HaveOccurred())

			Expect(meshServices).To(Equal(expectedList))
		})
	})

	Context("Test listServicesForPod()", func() {
		It("lists services for pod", func() {
			namespace := uuid.New().String()
			selectors := map[string]string{constants.AppLabel: tests.SelectorValue}
			mockKubeController := k8s.NewMockController(mockCtrl)
			var serviceNames []string
			var services []*v1.Service = []*v1.Service{}

			{
				// Create a service
				service := tests.NewServiceFixture(service1Name, namespace, selectors)
				services = append(services, service)
				_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceNames = append(serviceNames, service.Name)
			}

			{
				// Create a second service
				svc2Name := "svc-name-2"
				service := tests.NewServiceFixture(svc2Name, namespace, selectors)
				services = append(services, service)
				_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceNames = append(serviceNames, service.Name)
			}

			pod := tests.NewPodFixture(namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
			mockKubeController.EXPECT().ListServices().Return(services)
			actualSvcs := listServicesForPod(&pod, mockKubeController)
			Expect(len(actualSvcs)).To(Equal(2))

			actualNames := []string{actualSvcs[0].Name, actualSvcs[1].Name}
			Expect(actualNames).To(Equal(serviceNames))
		})

		It("should correctly not list services for pod that don't match the service's selectors", func() {
			namespace := uuid.New().String()
			selectors := map[string]string{"some-key": "some-value"}
			mockKubeController := k8s.NewMockController(mockCtrl)

			// Create a service
			service := tests.NewServiceFixture(service1Name, namespace, selectors)
			_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{service})
			pod := tests.NewPodFixture(namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
			actualSvcs := listServicesForPod(&pod, mockKubeController)
			Expect(len(actualSvcs)).To(Equal(0))
		})

		It("should correctly not list services for pod that don't match the service's selectors", func() {
			namespace := uuid.New().String()
			mockKubeController := k8s.NewMockController(mockCtrl)

			// The selector below has an additional label which the pod does not have.
			// Even though the first selector label matches the label on the pod, the
			// second selector label invalidates k8s selector matching criteria.
			selectors := map[string]string{
				constants.AppLabel: tests.SelectorValue,
				"some-key":         "some-value",
			}

			// Create a service
			service := tests.NewServiceFixture(service1Name, namespace, selectors)
			_, err := kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{service})
			pod := tests.NewPodFixture(namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
			actualSvcs := listServicesForPod(&pod, mockKubeController)
			Expect(len(actualSvcs)).To(Equal(0))
		})
	})

	Context("Test listServiceNames()", func() {
		It("converts a list of OSM Mesh Services to a list of strings (the Mesh Service names)", func() {

			services := []service.MeshService{
				{
					Namespace: "foo",
					Name:      "A",
				},
				{
					Namespace: "default",
					Name:      "bookstore-apex",
				},
			}

			expected := []string{
				"foo/A",
				"default/bookstore-apex",
			}

			actual := listServiceNames(services)

			Expect(actual).To(Equal(expected))
		})
	})
})

func TestListPodsForService(t *testing.T) {
	tests := []struct {
		name         string
		service      *v1.Service
		existingPods []*v1.Pod
		expected     []v1.Pod
	}{
		{
			name: "no existing pods",
			service: &v1.Service{
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"a": "selector",
					},
				},
			},
			existingPods: nil,
			expected:     nil,
		},
		{
			name: "match only pod",
			service: &v1.Service{
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"a": "selector",
					},
				},
			},
			existingPods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod",
						Labels: map[string]string{
							"some": "labels",
							"that": "match",
							"a":    "selector",
						},
					},
				},
			},
			expected: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod",
						Labels: map[string]string{
							"some": "labels",
							"that": "match",
							"a":    "selector",
						},
					},
				},
			},
		},
		{
			name: "match pod except for namespace",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "svc",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"a": "selector",
					},
				},
			},
			existingPods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod",
						Namespace: "pod",
						Labels: map[string]string{
							"a": "selector",
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "empty service selector",
			service: &v1.Service{
				Spec: v1.ServiceSpec{
					Selector: map[string]string{},
				},
			},
			existingPods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod",
						Labels: map[string]string{
							"a": "selector",
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "match several pods",
			service: &v1.Service{
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"a": "selector",
					},
				},
			},
			existingPods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod1",
						Labels: map[string]string{
							"a": "selector",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod2",
						Labels: map[string]string{
							"a": "selector",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "pod3",
						Labels: map[string]string{},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod4",
						Labels: map[string]string{
							"a": "selector",
						},
					},
				},
			},
			expected: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod1",
						Labels: map[string]string{
							"a": "selector",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod2",
						Labels: map[string]string{
							"a": "selector",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod4",
						Labels: map[string]string{
							"a": "selector",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			kubeController := k8s.NewMockController(ctrl)
			if test.service != nil && len(test.service.Spec.Selector) != 0 {
				kubeController.EXPECT().ListPods().Return(test.existingPods).Times(1)
			}

			actual := listPodsForService(test.service, kubeController)
			tassert.Equal(t, test.expected, actual)
		})
	}
}

func TestGetCertCommonNameForPod(t *testing.T) {
	tests := []struct {
		name      string
		pod       v1.Pod
		shouldErr bool
	}{
		{
			name: "valid CN",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uuid.New().String(),
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			shouldErr: false,
		},
		{
			name: "invalid UID",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uuid.New().String() + "-not-valid",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			shouldErr: true,
		},
		{
			name: "no UID",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels:    map[string]string{},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			cn, err := getCertCommonNameForPod(test.pod)
			if test.shouldErr {
				assert.Empty(cn)
				assert.Error(err)
			} else {
				assert.NotEmpty(cn)
				assert.NoError(err)
			}
		})
	}
}

func TestKubernetesServicesToMeshServices(t *testing.T) {
	testCases := []struct {
		name                 string
		k8sServices          []v1.Service
		expectedMeshServices []service.MeshService
	}{
		{
			name: "k8s services to mesh services",
			k8sServices: []v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name: "p1",
							Port: 80,
						}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns2",
						Name:      "s2",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name: "p2",
							Port: 80,
						}},
					},
				},
			},
			expectedMeshServices: []service.MeshService{
				{
					Namespace: "ns1",
					Name:      "s1",
					Protocol:  "http",
					Port:      80,
				},
				{
					Namespace: "ns2",
					Name:      "s2",
					Protocol:  "http",
					Port:      80,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubeController := k8s.NewMockController(mockCtrl)

			mockKubeController.EXPECT().GetEndpoints(gomock.Any()).Return(&v1.Endpoints{}, nil).Times(len(tc.k8sServices))

			actual := kubernetesServicesToMeshServices(mockKubeController, tc.k8sServices)
			assert.ElementsMatch(tc.expectedMeshServices, actual)
		})
	}
}
