package catalog

import (
	"time"

	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/endpoint/providers/kube"
	"github.com/openservicemesh/osm/pkg/ingress"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
)

// NewFakeMeshCatalog creates a new struct implementing catalog.MeshCataloger interface used for testing.
func NewFakeMeshCatalog(kubeClient kubernetes.Interface) *MeshCatalog {
	var (
		mockCtrl         *gomock.Controller
		mockNsController *namespace.MockController
	)

	mockCtrl = gomock.NewController(ginkgo.GinkgoT())
	mockNsController = namespace.NewMockController(mockCtrl)
	meshSpec := smi.NewFakeMeshSpecClient()
	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	ingressMonitor := ingress.NewFakeIngressMonitor()
	stop := make(<-chan struct{})
	endpointProviders := []endpoint.Provider{
		kube.NewFakeProvider(),
	}

	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"
	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

	testChan := make(chan interface{})

	mockNsController.EXPECT().IsMonitoredNamespace(tests.BookstoreService.Namespace).Return(true).AnyTimes()
	mockNsController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()
	mockNsController.EXPECT().IsMonitoredNamespace(tests.BookwarehouseService.Namespace).Return(true).AnyTimes()
	mockNsController.EXPECT().GetAnnouncementsChannel().Return(testChan).AnyTimes()

	return NewMeshCatalog(mockNsController, kubeClient, meshSpec, certManager,
		ingressMonitor, stop, cfg, endpointProviders...)
}
