package fake

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/k8s/informers"

	"github.com/openservicemesh/osm/pkg/catalog"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/policy"
	kubeFake "github.com/openservicemesh/osm/pkg/providers/kube/fake"
	"github.com/openservicemesh/osm/pkg/service"
	smiFake "github.com/openservicemesh/osm/pkg/smi/fake"
	"github.com/openservicemesh/osm/pkg/tests"
)

// NewFakeMeshCatalog creates a new struct implementing catalog.MeshCataloger interface used for testing.
func NewFakeMeshCatalog(kubeClient kubernetes.Interface, meshConfigClient configClientset.Interface) *catalog.MeshCatalog {
	mockCtrl := gomock.NewController(ginkgo.GinkgoT())
	mockKubeController := k8s.NewMockController(mockCtrl)
	mockPolicyController := policy.NewMockController(mockCtrl)

	meshSpec := smiFake.NewFakeMeshSpecClient()

	stop := make(<-chan struct{})

	provider := kubeFake.NewFakeProvider()

	endpointProviders := []endpoint.Provider{
		provider,
	}
	serviceProviders := []service.Provider{
		provider,
	}

	osmNamespace := "-test-osm-namespace-"
	osmMeshConfigName := "-test-osm-mesh-config-"
	ic, err := informers.NewInformerCollection("osm", stop, informers.WithKubeClient(kubeClient), informers.WithConfigClient(meshConfigClient, osmMeshConfigName, osmNamespace))
	if err != nil {
		return nil
	}

	cfg := configurator.NewConfigurator(ic, osmNamespace, osmMeshConfigName, nil)

	certManager := tresorFake.NewFake(nil, 1*time.Hour)

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

		podRet := []*corev1.Pod{}
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
	mockKubeController.EXPECT().GetTargetPortForServicePort(
		gomock.Any(), gomock.Any()).Return(uint16(tests.ServicePort), nil).AnyTimes()

	mockPolicyController.EXPECT().ListEgressPoliciesForSourceIdentity(gomock.Any()).Return(nil).AnyTimes()
	mockPolicyController.EXPECT().GetIngressBackendPolicy(gomock.Any()).Return(nil).AnyTimes()
	mockPolicyController.EXPECT().GetUpstreamTrafficSetting(gomock.Any()).Return(nil).AnyTimes()

	return catalog.NewMeshCatalog(mockKubeController, meshSpec, certManager,
		mockPolicyController, stop, cfg, serviceProviders, endpointProviders, messaging.NewBroker(stop))
}
