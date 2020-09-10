package catalog

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/endpoint/providers/kube"
	"github.com/openservicemesh/osm/pkg/ingress"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
)

// NewFakeMeshCatalog creates a new struct implementing catalog.MeshCataloger interface used for testing.
func NewFakeMeshCatalog(kubeClient kubernetes.Interface) *MeshCatalog {
	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		mockIngressMonitor *ingress.MockMonitor
	)

	mockCtrl = gomock.NewController(ginkgo.GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	mockIngressMonitor = ingress.NewMockMonitor(mockCtrl)

	meshSpec := smi.NewFakeMeshSpecClient()
	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	stop := make(<-chan struct{})
	endpointProviders := []endpoint.Provider{
		kube.NewFakeProvider(),
	}

	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"
	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

	testChan := make(chan interface{})

	mockIngressMonitor.EXPECT().GetIngressResources(gomock.Any()).Return(nil, nil).AnyTimes()
	mockIngressMonitor.EXPECT().GetAnnouncementsChannel().Return(testChan).AnyTimes()

	// Monitored namespaces is made a set to make sure we don't repeat namespaces on mock
	listExpectedNs := tests.GetUnique([]string{
		tests.BookstoreService.Namespace,
		tests.BookbuyerService.Namespace,
		tests.BookstoreApexService.Namespace,
	})

	// #1683 tracks potential improvements to the following dynamic mocks
	mockKubeController.EXPECT().ListServices().DoAndReturn(func() []*corev1.Service {
		// play pretend this call queries a controller cache
		var services []*corev1.Service

		for _, ns := range listExpectedNs {
			svcList, _ := kubeClient.CoreV1().Services(ns).List(context.Background(), metav1.ListOptions{})
			for _, svcItem := range svcList.Items {
				services = append(services, &svcItem)
			}
		}

		return services
	}).AnyTimes()
	mockKubeController.EXPECT().GetService(gomock.Any()).DoAndReturn(func(msh service.MeshService) *v1.Service {
		// play pretend this call queries a controller cache
		vv, err := kubeClient.CoreV1().Services(msh.Namespace).Get(context.Background(), msh.Name, metav1.GetOptions{})
		if err != nil {
			return nil
		}

		return vv
	}).AnyTimes()
	mockKubeController.EXPECT().GetAnnouncementsChannel(k8s.Namespaces).Return(testChan).AnyTimes()
	mockKubeController.EXPECT().GetAnnouncementsChannel(k8s.Services).Return(testChan).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookstoreService.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookwarehouseService.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().GetAnnouncementsChannel(k8s.Namespaces).Return(testChan).AnyTimes()

	return NewMeshCatalog(mockKubeController, kubeClient, meshSpec, certManager,
		mockIngressMonitor, stop, cfg, endpointProviders...)
}
