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

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
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
	proxyRegistry := NewProxyRegistry(&KubeProxyServiceMapper{mockKubeController})

	Context("Test ListProxyServices()", func() {
		It("works as expected", func() {
			pod := tests.NewPodFixture(tests.Namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
			Expect(pod.Spec.ServiceAccountName).To(Equal(tests.BookstoreServiceAccountName))
			mockKubeController.EXPECT().ListPods().Return([]*v1.Pod{&pod}).Times(1)

			// Create the SERVICE
			svcName := uuid.New().String()
			selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, tests.Namespace, selector)

			svcName2 := uuid.New().String()
			svc2 := tests.NewServiceFixture(svcName2, tests.Namespace, selector)
			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{svc, svc2}).Times(1)

			certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", tests.ProxyUUID, envoy.KindSidecar, tests.BookstoreServiceAccountName, tests.Namespace))
			certSerialNumber := certificate.SerialNumber("123456")
			proxy, err := envoy.NewProxy(certCommonName, certSerialNumber, nil)
			Expect(err).ToNot(HaveOccurred())
			meshServices, err := proxyRegistry.ListProxyServices(proxy)
			Expect(err).ToNot(HaveOccurred())
			expectedSvc := service.MeshService{
				Namespace: tests.Namespace,
				Name:      svcName,
			}

			expectedSvc2 := service.MeshService{
				Namespace: tests.Namespace,
				Name:      svcName2,
			}
			expectedList := []service.MeshService{expectedSvc, expectedSvc2}

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
			selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
			svc := tests.NewServiceFixture(svcName, namespace, selector)
			mockKubeController.EXPECT().ListServices().Return([]*v1.Service{svc}).Times(1)

			podCN := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", proxyUUID, envoy.KindSidecar, tests.BookstoreServiceAccountName, namespace))
			certSerialNumber := certificate.SerialNumber("123456")
			newProxy, err := envoy.NewProxy(podCN, certSerialNumber, nil)
			Expect(err).ToNot(HaveOccurred())
			meshServices, err := proxyRegistry.ListProxyServices(newProxy)
			Expect(err).ToNot(HaveOccurred())

			expected := service.MeshService{
				Namespace: namespace,
				Name:      svcName,
			}
			expectedList := []service.MeshService{expected}

			Expect(meshServices).To(Equal(expectedList))
		})
	})

	Context("Test listServicesForPod()", func() {
		It("lists services for pod", func() {
			namespace := uuid.New().String()
			selectors := map[string]string{tests.SelectorKey: tests.SelectorValue}
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
				tests.SelectorKey: tests.SelectorValue,
				"some-key":        "some-value",
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

	Context("Test kubernetesServicesToMeshServices()", func() {
		It("converts a list of Kubernetes Services to a list of OSM Mesh Services", func() {

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

			expected := []service.MeshService{
				{
					Namespace: "foo",
					Name:      "A",
				},
				{
					Namespace: "default",
					Name:      "bookstore-apex",
				},
			}

			actual := kubernetesServicesToMeshServices(services)

			Expect(actual).To(Equal(expected))
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

func TestAsyncKubeProxyServiceMapperListServicesForProxy(t *testing.T) {
	cn1 := envoy.NewCertCommonName(uuid.New(), envoy.KindSidecar, "svcacc1", "ns1")
	cn2 := envoy.NewCertCommonName(uuid.New(), envoy.KindSidecar, "svcacc2", "ns2")
	svc1 := service.MeshService{Namespace: "ns1", Name: "svc1"}
	mapper := &AsyncKubeProxyServiceMapper{
		servicesForCN: map[certificate.CommonName][]service.MeshService{
			cn1: nil,
			cn2: {svc1},
		},
	}

	assert := tassert.New(t)

	proxy, err := envoy.NewProxy(cn1, "", nil)
	assert.NoError(err)
	svcs, err := mapper.ListProxyServices(proxy)
	assert.NoError(err)
	assert.Nil(svcs)

	proxy, err = envoy.NewProxy(cn2, "", nil)
	assert.NoError(err)
	svcs, err = mapper.ListProxyServices(proxy)
	assert.NoError(err)
	assert.Equal([]service.MeshService{svc1}, svcs)
}

func TestAsyncKubeProxyServiceMapperRun(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	kubeController := k8s.NewMockController(mockCtrl)

	stop := make(chan struct{})
	defer close(stop)

	k := NewAsyncKubeProxyServiceMapper(kubeController)

	assert.Empty(k.servicesForCN)

	proxyUID := uuid.New()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Labels: map[string]string{
				"app":                            "my-app",
				constants.EnvoyUniqueIDLabelName: proxyUID.String(),
			},
		},
		Spec: v1.PodSpec{
			ServiceAccountName: "my-service-acc",
		},
	}
	cn := envoy.NewCertCommonName(proxyUID, envoy.KindSidecar, pod.Spec.ServiceAccountName, pod.Namespace)
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "my-namespace",
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "my-app",
			},
		},
	}
	broadcast := events.GetPubSubInstance().Subscribe(announcements.ScheduleProxyBroadcast)

	kubeController.EXPECT().ListPods().Return([]*v1.Pod{pod}).Times(1)
	kubeController.EXPECT().ListServices().Return([]*v1.Service{svc}).Times(1)

	k.Run(stop)

	svcs := k.servicesForCN[cn]
	assert.NotEmpty(svcs)

	events.GetPubSubInstance().Publish(events.PubSubMessage{
		AnnouncementType: announcements.ServiceDeleted,
		OldObj:           svc,
	})
	<-broadcast

	assert.Empty(k.servicesForCN[cn])

	events.GetPubSubInstance().Publish(events.PubSubMessage{
		AnnouncementType: announcements.PodDeleted,
		OldObj:           pod,
	})
	<-broadcast

	assert.Empty(k.servicesForCN)

	kubeController.EXPECT().ListServices().Return(nil).Times(1)
	events.GetPubSubInstance().Publish(events.PubSubMessage{
		AnnouncementType: announcements.PodAdded,
		NewObj:           pod,
	})
	<-broadcast

	svcs, ok := k.servicesForCN[cn]
	assert.True(ok, "expected CN %s to exist in cache, but it doesn't", cn)
	assert.Empty(svcs)

	kubeController.EXPECT().ListPods().Return([]*v1.Pod{pod}).Times(1)
	events.GetPubSubInstance().Publish(events.PubSubMessage{
		AnnouncementType: announcements.ServiceAdded,
		NewObj:           svc,
	})
	<-broadcast

	svcs = k.servicesForCN[cn]
	assert.NotEmpty(svcs)
}

func TestKubeHandlePodUpdate(t *testing.T) {
	uid1 := uuid.New()
	uid2 := uuid.New()

	tests := []struct {
		name                  string
		existingCNsToServices map[certificate.CommonName][]service.MeshService
		existingServicesToCNs map[service.MeshService]map[certificate.CommonName]struct{}
		existingSvcs          []*v1.Service
		pod                   *v1.Pod
		expectedCNsToServices map[certificate.CommonName][]service.MeshService
		expectedServicesToCNs map[service.MeshService]map[certificate.CommonName]struct{}
	}{
		{
			name:                  "nil pod",
			pod:                   nil,
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "first new pod matching no services",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uid1.String(),
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "first new pod matching one service",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uid1.String(),
						"app":                            "my-app",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			existingSvcs: []*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc",
						Namespace: "ns",
					},
					Spec: v1.ServiceSpec{
						Selector: map[string]string{
							"app": "my-app",
						},
					},
				},
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
		{
			name: "first new pod matching two services",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uid1.String(),
						"app":                            "my-app",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			existingSvcs: []*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc1",
						Namespace: "ns",
					},
					Spec: v1.ServiceSpec{
						Selector: map[string]string{
							"app": "my-app",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc2",
						Namespace: "ns",
					},
					Spec: v1.ServiceSpec{
						Selector: map[string]string{
							"app": "my-app",
						},
					},
				},
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc1",
						Namespace: "ns",
					},
					{
						Name:      "svc2",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc1", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "svc2", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
		{
			name: "second pod matching no services",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uid1.String(),
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "other-svc1",
						Namespace: "ns",
					},
				},
			},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "other-svc1", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): nil,
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "other-svc1",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "other-svc1", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
		{
			name: "pod without envoy uid label",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels:    map[string]string{},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "second pod matching same service",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uid2.String(),
						"app":                            "my-app",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			existingSvcs: []*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "svc",
						Namespace: "ns",
					},
					Spec: v1.ServiceSpec{
						Selector: map[string]string{
							"app": "my-app",
						},
					},
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			kubeController := k8s.NewMockController(mockCtrl)
			kubeController.EXPECT().ListServices().Return(test.existingSvcs)

			k := &AsyncKubeProxyServiceMapper{
				kubeController: kubeController,
				servicesForCN:  test.existingCNsToServices,
				cnsForService:  test.existingServicesToCNs,
			}
			if k.servicesForCN == nil {
				k.servicesForCN = map[certificate.CommonName][]service.MeshService{}
			}
			if k.cnsForService == nil {
				k.cnsForService = map[service.MeshService]map[certificate.CommonName]struct{}{}
			}

			k.handlePodUpdate(test.pod)

			tassert.Equal(t, test.expectedCNsToServices, k.servicesForCN)
			tassert.Equal(t, test.expectedServicesToCNs, k.cnsForService)
		})
	}
}

func TestKubeHandlePodDelete(t *testing.T) {
	uid1 := uuid.New()
	uid2 := uuid.New()

	tests := []struct {
		name                  string
		existingCNsToServices map[certificate.CommonName][]service.MeshService
		existingServicesToCNs map[service.MeshService]map[certificate.CommonName]struct{}
		pod                   *v1.Pod
		expectedCNsToServices map[certificate.CommonName][]service.MeshService
		expectedServicesToCNs map[service.MeshService]map[certificate.CommonName]struct{}
	}{
		{
			name:                  "empty existing cache",
			pod:                   nil,
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "delete nil pod",
			pod:  nil,
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "delete pod without proxy uuid",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels:    map[string]string{},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "delete pod not matching any existing",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uid1.String(),
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "delete only pod",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uid1.String(),
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "delete one of several pods",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uid1.String(),
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): nil,
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "delete one of several pods matching a service",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uid1.String(),
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "svcacc",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{Namespace: "ns", Name: "svc"},
				},
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{Namespace: "ns", Name: "svc"},
				},
			},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Namespace: "ns", Name: "svc"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{Namespace: "ns", Name: "svc"},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Namespace: "ns", Name: "svc"}: {
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k := &AsyncKubeProxyServiceMapper{
				servicesForCN: test.existingCNsToServices,
				cnsForService: test.existingServicesToCNs,
			}
			if k.servicesForCN == nil {
				k.servicesForCN = map[certificate.CommonName][]service.MeshService{}
			}
			if k.cnsForService == nil {
				k.cnsForService = map[service.MeshService]map[certificate.CommonName]struct{}{}
			}

			k.handlePodDelete(test.pod)

			tassert.Equal(t, test.expectedCNsToServices, k.servicesForCN)
			tassert.Equal(t, test.expectedServicesToCNs, k.cnsForService)
		})
	}
}

func TestKubeHandleServiceUpdate(t *testing.T) {
	uid1 := uuid.New()
	uid2 := uuid.New()

	tests := []struct {
		name                  string
		existingCNsToServices map[certificate.CommonName][]service.MeshService
		existingServicesToCNs map[service.MeshService]map[certificate.CommonName]struct{}
		existingPods          []*v1.Pod
		service               *v1.Service
		expectedCNsToServices map[certificate.CommonName][]service.MeshService
		expectedServicesToCNs map[service.MeshService]map[certificate.CommonName]struct{}
	}{
		{
			name:    "add nil service",
			service: nil,
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "add service backed by no pods",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: "ns",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "not-svc",
						Namespace: "ns",
					},
				},
			},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "not-svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "not-svc",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "not-svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "svc", Namespace: "ns"}: {},
			},
		},
		{
			name: "add service backed by one pod to empty cache",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: "ns",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app": "my-app",
					},
				},
			},
			existingPods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns",
						Labels: map[string]string{
							constants.EnvoyUniqueIDLabelName: uid1.String(),
							"app":                            "my-app",
						},
					},
					Spec: v1.PodSpec{
						ServiceAccountName: "svcacc",
					},
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
		{
			name: "add service backed by one pod with invalid UID",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: "ns",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app": "my-app",
					},
				},
			},
			existingPods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns",
						Labels: map[string]string{
							constants.EnvoyUniqueIDLabelName: uid1.String() + "-invalid",
							"app":                            "my-app",
						},
					},
					Spec: v1.PodSpec{
						ServiceAccountName: "svcacc",
					},
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "other-svc",
						Namespace: "ns",
					},
				},
			},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "other-svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "other-svc",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "other-svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "svc", Namespace: "ns"}: {},
			},
		},
		{
			name: "add service backed by multiple pods to empty cache",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: "ns",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app": "my-app",
					},
				},
			},
			existingPods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns",
						Labels: map[string]string{
							constants.EnvoyUniqueIDLabelName: uid1.String(),
							"app":                            "my-app",
						},
					},
					Spec: v1.PodSpec{
						ServiceAccountName: "svcacc",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns",
						Labels: map[string]string{
							constants.EnvoyUniqueIDLabelName: uid2.String(),
							"app":                            "my-app",
						},
					},
					Spec: v1.PodSpec{
						ServiceAccountName: "svcacc",
					},
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
		{
			name: "update service backed by multiple pods already cached",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: "ns",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app": "my-app",
					},
				},
			},
			existingPods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns",
						Labels: map[string]string{
							constants.EnvoyUniqueIDLabelName: uid1.String(),
							"app":                            "my-app",
						},
					},
					Spec: v1.PodSpec{
						ServiceAccountName: "svcacc",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns",
						Labels: map[string]string{
							constants.EnvoyUniqueIDLabelName: uid2.String(),
							"app":                            "my-app",
						},
					},
					Spec: v1.PodSpec{
						ServiceAccountName: "svcacc",
					},
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "not-svc",
						Namespace: "ns",
					},
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
					{
						Name:      "svc",
						Namespace: "not-ns",
					},
				},
			},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "not-svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "svc", Namespace: "not-ns"}: {
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
			expectedCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "not-svc",
						Namespace: "ns",
					},
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
					{
						Name:      "svc",
						Namespace: "not-ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "not-svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "svc", Namespace: "not-ns"}: {
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			kubeController := k8s.NewMockController(mockCtrl)
			kubeController.EXPECT().ListPods().Return(test.existingPods)

			k := &AsyncKubeProxyServiceMapper{
				kubeController: kubeController,
				servicesForCN:  test.existingCNsToServices,
				cnsForService:  test.existingServicesToCNs,
			}
			if k.servicesForCN == nil {
				k.servicesForCN = map[certificate.CommonName][]service.MeshService{}
			}
			if k.cnsForService == nil {
				k.cnsForService = map[service.MeshService]map[certificate.CommonName]struct{}{}
			}

			k.handleServiceUpdate(test.service)

			tassert.Equal(t, test.expectedCNsToServices, k.servicesForCN)
			tassert.Equal(t, test.expectedServicesToCNs, k.cnsForService)
		})
	}
}

func TestKubeHandleServiceDelete(t *testing.T) {
	uid1 := uuid.New()
	uid2 := uuid.New()

	tests := []struct {
		name                  string
		existingCNsToServices map[certificate.CommonName][]service.MeshService
		existingServicesToCNs map[service.MeshService]map[certificate.CommonName]struct{}
		service               *v1.Service
		expectedCnsToServices map[certificate.CommonName][]service.MeshService
		expectedServicesToCNs map[service.MeshService]map[certificate.CommonName]struct{}
	}{
		{
			name:                  "empty existing cache",
			service:               nil,
			expectedCnsToServices: map[certificate.CommonName][]service.MeshService{},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name:    "delete nil service",
			service: nil,
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
			expectedCnsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
		{
			name: "delete service backed by no proxies",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "svc",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "not-svc",
						Namespace: "ns",
					},
					{
						Name:      "svc",
						Namespace: "not-ns",
					},
				},
			},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "not-svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "svc", Namespace: "not-ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
			expectedCnsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "not-svc",
						Namespace: "ns",
					},
					{
						Name:      "svc",
						Namespace: "not-ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "not-svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "svc", Namespace: "not-ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
		{
			name: "delete only service from one proxy",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "svc",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
			expectedCnsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): nil,
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{},
		},
		{
			name: "delete service from all proxies",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "svc",
				},
			},
			existingCNsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "ns",
					},
					{
						Name:      "svc",
						Namespace: "not-ns",
					},
				},
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "not-svc",
						Namespace: "ns",
					},
					{
						Name:      "svc",
						Namespace: "ns",
					},
				},
			},
			existingServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "not-svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "svc", Namespace: "not-ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
			expectedCnsToServices: map[certificate.CommonName][]service.MeshService{
				envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "svc",
						Namespace: "not-ns",
					},
				},
				envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {
					{
						Name:      "not-svc",
						Namespace: "ns",
					},
				},
			},
			expectedServicesToCNs: map[service.MeshService]map[certificate.CommonName]struct{}{
				{Name: "not-svc", Namespace: "ns"}: {
					envoy.NewCertCommonName(uid2, envoy.KindSidecar, "svcacc", "ns"): {},
				},
				{Name: "svc", Namespace: "not-ns"}: {
					envoy.NewCertCommonName(uid1, envoy.KindSidecar, "svcacc", "ns"): {},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k := &AsyncKubeProxyServiceMapper{
				servicesForCN: test.existingCNsToServices,
				cnsForService: test.existingServicesToCNs,
			}
			if k.servicesForCN == nil {
				k.servicesForCN = map[certificate.CommonName][]service.MeshService{}
			}
			if k.cnsForService == nil {
				k.cnsForService = map[service.MeshService]map[certificate.CommonName]struct{}{}
			}

			k.handleServiceDelete(test.service)

			tassert.Equal(t, test.expectedCnsToServices, k.servicesForCN)
			tassert.Equal(t, test.expectedServicesToCNs, k.cnsForService)
		})
	}
}

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
			kubeController.EXPECT().ListPods().Return(test.existingPods).Times(1)

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

func TestAsyncKubeProxyServiceMapperRace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	kubeController := k8s.NewMockController(mockCtrl)

	k := NewAsyncKubeProxyServiceMapper(kubeController)

	stop := make(chan struct{})

	kubeController.EXPECT().ListPods().Return(nil).Times(1)
	k.Run(stop)

	proxyUUID := uuid.New()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
				"app":                            "my-app",
			},
		},
	}
	svc := &v1.Service{
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "my-app",
			},
		},
	}
	cn := envoy.NewCertCommonName(proxyUUID, envoy.KindSidecar, pod.Spec.ServiceAccountName, pod.Namespace)
	proxy, err := envoy.NewProxy(cn, "", nil)
	tassert.NoError(t, err)

	numWriteLoopIters := 100
	totalExpectedBroadcasts := numWriteLoopIters * 4 // four Publish() calls per loop
	doWrites := func() {
		for i := 0; i < numWriteLoopIters; i++ {
			kubeController.EXPECT().ListServices().Return(nil).Times(1)
			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: announcements.PodAdded,
				NewObj:           pod,
			})

			kubeController.EXPECT().ListPods().Return([]*v1.Pod{pod}).Times(1)
			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: announcements.ServiceAdded,
				NewObj:           svc,
			})

			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: announcements.ServiceDeleted,
				OldObj:           svc,
			})
			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: announcements.PodDeleted,
				OldObj:           pod,
			})
		}
	}

	doReads := func(stop <-chan struct{}) {
		for {
			select {
			case <-stop:
				return
			default:
				_, _ = k.ListProxyServices(proxy)
			}
		}
	}

	// wait ensures all the published events have been handled by the mapper.
	wait := func() {
		broadcast := events.GetPubSubInstance().Subscribe(announcements.ScheduleProxyBroadcast)
		defer events.GetPubSubInstance().Unsub(broadcast)
		for i := 0; i < totalExpectedBroadcasts; i++ {
			<-broadcast
		}
		close(stop)
	}

	go wait()
	go doWrites()
	go doReads(stop)
	go doReads(stop)

	<-stop
}
