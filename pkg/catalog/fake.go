package catalog

import (
	"context"

	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/providers/kube"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
)

// NewFakeMeshCatalog creates a new struct implementing catalog.MeshCataloger interface used for testing.
func NewFakeMeshCatalog(kubeClient kubernetes.Interface, meshConfigClient versioned.Interface) *MeshCatalog {
	var (
		mockCtrl             *gomock.Controller
		mockKubeController   *k8s.MockController
		mockPolicyController *policy.MockController
	)

	mockCtrl = gomock.NewController(ginkgo.GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	mockPolicyController = policy.NewMockController(mockCtrl)

	meshSpec := smi.NewFakeMeshSpecClient()

	stop := make(<-chan struct{})

	provider := kube.NewFakeProvider()

	endpointProviders := []endpoint.Provider{
		provider,
	}
	serviceProviders := []service.Provider{
		provider,
	}

	osmNamespace := "-test-osm-namespace-"
	osmMeshConfigName := "-test-osm-mesh-config-"
	cfg := configurator.NewConfigurator(meshConfigClient, stop, osmNamespace, osmMeshConfigName, nil)

	certManager := tresor.NewFake(nil)

	// #1683 tracks potential improvements to the following dynamic mocks
	mockKubeController.EXPECT().ListServices().DoAndReturn(func() []*corev1.Service {
		// play pretend this call queries a controller cache
		var services []*corev1.Service

		// This assumes that catalog tests use monitored namespaces at all times
		svcList, _ := kubeClient.CoreV1().Services("").List(context.Background(), metav1.ListOptions{})
		for idx := range svcList.Items {
			services = append(services, &svcList.Items[idx])
		}

		return services
	}).AnyTimes()
	mockKubeController.EXPECT().ListServiceAccounts().DoAndReturn(func() []*corev1.ServiceAccount {
		// play pretend this call queries a controller cache
		var serviceAccounts []*corev1.ServiceAccount

		// This assumes that catalog tests use monitored namespaces at all times
		svcAccountList, _ := kubeClient.CoreV1().ServiceAccounts("").List(context.Background(), metav1.ListOptions{})
		for idx := range svcAccountList.Items {
			serviceAccounts = append(serviceAccounts, &svcAccountList.Items[idx])
		}

		return serviceAccounts
	}).AnyTimes()
	mockKubeController.EXPECT().GetService(gomock.Any()).DoAndReturn(func(msh service.MeshService) *corev1.Service {
		// play pretend this call queries a controller cache
		vv, err := kubeClient.CoreV1().Services(msh.Namespace).Get(context.Background(), msh.Name, metav1.GetOptions{})
		if err != nil {
			return nil
		}

		return vv
	}).AnyTimes()
	mockKubeController.EXPECT().ListPods().DoAndReturn(func() []*corev1.Pod {
		vv, err := kubeClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil
		}

		var podRet []*corev1.Pod = []*corev1.Pod{}
		for idx := range vv.Items {
			podRet = append(podRet, &vv.Items[idx])
		}
		return podRet
	}).AnyTimes()

	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookstoreV1Service.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookstoreV2Service.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookwarehouseService.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().ListServiceIdentitiesForService(tests.BookstoreV1Service).Return([]identity.K8sServiceAccount{tests.BookstoreServiceAccount}, nil).AnyTimes()
	mockKubeController.EXPECT().ListServiceIdentitiesForService(tests.BookstoreV2Service).Return([]identity.K8sServiceAccount{tests.BookstoreV2ServiceAccount}, nil).AnyTimes()
	mockKubeController.EXPECT().ListServiceIdentitiesForService(tests.BookbuyerService).Return([]identity.K8sServiceAccount{tests.BookbuyerServiceAccount}, nil).AnyTimes()

	mockPolicyController.EXPECT().ListEgressPoliciesForSourceIdentity(gomock.Any()).Return(nil).AnyTimes()
	mockPolicyController.EXPECT().GetIngressBackendPolicy(gomock.Any()).Return(nil).AnyTimes()
	mockPolicyController.EXPECT().GetUpstreamTrafficSetting(gomock.Any()).Return(nil).AnyTimes()

	return NewMeshCatalog(mockKubeController, meshSpec, certManager,
		mockPolicyController, stop, cfg, serviceProviders, endpointProviders, messaging.NewBroker(stop))
}

func newFakeMeshCatalog() *MeshCatalog {
	var (
		mockCtrl             *gomock.Controller
		mockKubeController   *k8s.MockController
		mockPolicyController *policy.MockController
	)

	mockCtrl = gomock.NewController(ginkgo.GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	mockPolicyController = policy.NewMockController(mockCtrl)

	meshSpec := smi.NewFakeMeshSpecClient()

	osmNamespace := "-test-osm-namespace-"
	osmMeshConfigName := "-test-osm-mesh-config-"

	stop := make(chan struct{})

	provider := kube.NewFakeProvider()

	endpointProviders := []endpoint.Provider{
		provider,
	}
	serviceProviders := []service.Provider{
		provider,
	}

	kubeClient := fake.NewSimpleClientset()
	configClient := configFake.NewSimpleClientset()

	cfg := configurator.NewConfigurator(configClient, stop, osmNamespace, osmMeshConfigName, nil)

	certManager := tresor.NewFake(nil)

	// Create a Bookstore-v1 pod
	pod := tests.NewPodFixture(tests.Namespace, tests.BookstoreV1Service.Name, tests.BookstoreServiceAccountName, tests.PodLabels)
	if _, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{}); err != nil {
		ginkgo.GinkgoT().Fatalf("Error creating new fake Mesh Catalog: %s", err.Error())
	}

	// Create a Bookstore-v2 pod
	pod = tests.NewPodFixture(tests.Namespace, tests.BookstoreV2Service.Name, tests.BookstoreV2ServiceAccountName, tests.PodLabels)
	if _, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{}); err != nil {
		ginkgo.GinkgoT().Fatalf("Error creating new fake Mesh Catalog: %s", err.Error())
	}

	// Create Bookstore-v1 Service
	selector := map[string]string{constants.AppLabel: tests.SelectorValue}
	svc := tests.NewServiceFixture(tests.BookstoreV1Service.Name, tests.BookstoreV1Service.Namespace, selector)
	if _, err := kubeClient.CoreV1().Services(tests.BookstoreV1Service.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		ginkgo.GinkgoT().Fatalf("Error creating new Bookstore service: %s", err.Error())
	}

	// Create Bookstore-v2 Service
	svc = tests.NewServiceFixture(tests.BookstoreV2Service.Name, tests.BookstoreV2Service.Namespace, selector)
	if _, err := kubeClient.CoreV1().Services(tests.BookstoreV2Service.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		ginkgo.GinkgoT().Fatalf("Error creating new Bookstore service: %s", err.Error())
	}

	// Create Bookbuyer Service
	svc = tests.NewServiceFixture(tests.BookbuyerService.Name, tests.BookbuyerService.Namespace, nil)
	if _, err := kubeClient.CoreV1().Services(tests.BookbuyerService.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		ginkgo.GinkgoT().Fatalf("Error creating new Bookbuyer service: %s", err.Error())
	}

	// Create Bookstore apex Service
	svc = tests.NewServiceFixture(tests.BookstoreApexService.Name, tests.BookstoreApexService.Namespace, nil)
	if _, err := kubeClient.CoreV1().Services(tests.BookstoreApexService.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		ginkgo.GinkgoT().Fatalf("Error creating new Bookstore Apex service", err.Error())
	}

	// Monitored namespaces is made a set to make sure we don't repeat namespaces on mock
	listExpectedNs := tests.GetUnique([]string{
		tests.BookstoreV1Service.Namespace,
		tests.BookbuyerService.Namespace,
		tests.BookstoreApexService.Namespace,
	})

	// #1683 tracks potential improvements to the following dynamic mocks
	mockKubeController.EXPECT().ListServices().DoAndReturn(func() []*corev1.Service {
		// play pretend this call queries a controller cache
		var services []*corev1.Service

		for _, ns := range listExpectedNs {
			// simulate lookup on controller cache
			svcList, _ := kubeClient.CoreV1().Services(ns).List(context.TODO(), metav1.ListOptions{})
			for serviceIdx := range svcList.Items {
				services = append(services, &svcList.Items[serviceIdx])
			}
		}

		return services
	}).AnyTimes()
	mockKubeController.EXPECT().GetService(gomock.Any()).DoAndReturn(func(msh service.MeshService) *corev1.Service {
		// simulate lookup on controller cache
		vv, err := kubeClient.CoreV1().Services(msh.Namespace).Get(context.TODO(), msh.Name, metav1.GetOptions{})

		if err != nil {
			return nil
		}

		return vv
	}).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookstoreV1Service.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookstoreV2Service.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookwarehouseService.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().ListMonitoredNamespaces().Return(listExpectedNs, nil).AnyTimes()

	mockPolicyController.EXPECT().ListEgressPoliciesForSourceIdentity(gomock.Any()).Return(nil).AnyTimes()

	return NewMeshCatalog(mockKubeController, meshSpec, certManager,
		mockPolicyController, stop, cfg, serviceProviders, endpointProviders, messaging.NewBroker(stop))
}
