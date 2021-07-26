package scenarios

import (
	"fmt"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/multicluster"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func toInt(val uint32) *wrappers.UInt32Value {
	return &wrappers.UInt32Value{
		Value: val,
	}
}

func weightedCluster(serviceName string, weight uint32) *xds_route.WeightedCluster_ClusterWeight {
	return &xds_route.WeightedCluster_ClusterWeight{
		Name:   fmt.Sprintf("default/%s", serviceName),
		Weight: toInt(weight),
	}
}

func getProxy(kubeClient kubernetes.Interface) (*envoy.Proxy, error) {
	bookbuyerPodLabels := map[string]string{
		tests.SelectorKey:                tests.BookbuyerService.Name,
		constants.EnvoyUniqueIDLabelName: tests.ProxyUUID,
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, tests.BookbuyerServiceName, tests.BookbuyerServiceAccountName, bookbuyerPodLabels); err != nil {
		return nil, err
	}

	bookstorePodLabels := map[string]string{
		tests.SelectorKey:                "bookstore",
		constants.EnvoyUniqueIDLabelName: uuid.New().String(),
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, "bookstore", tests.BookstoreServiceAccountName, bookstorePodLabels); err != nil {
		return nil, err
	}

	selectors := map[string]string{
		tests.SelectorKey: tests.BookbuyerServiceName,
	}
	if _, err := tests.MakeService(kubeClient, tests.BookbuyerServiceName, selectors); err != nil {
		return nil, err
	}

	for _, svcName := range []string{tests.BookstoreApexServiceName, tests.BookstoreV1ServiceName, tests.BookstoreV2ServiceName} {
		selectors := map[string]string{
			tests.SelectorKey: "bookstore",
		}
		if _, err := tests.MakeService(kubeClient, svcName, selectors); err != nil {
			return nil, err
		}
	}

	osmServiceAccount := "osm"
	osmNamespace := "osm-system"
	certCommonName := multicluster.GetMulticlusterGatewaySubjectCommonName(osmServiceAccount, osmNamespace)
	certSerialNumber := certificate.SerialNumber("123456")
	return envoy.NewProxy(certCommonName, certSerialNumber, nil)
}

func setupMulticlusterGatewayTest(mockCtrl *gomock.Controller) (catalog.MeshCataloger, *envoy.Proxy, *registry.ProxyRegistry, configurator.Configurator, error) {
	// ---[  Setup the test context  ]---------
	kubeClient := testclient.NewSimpleClientset()
	configClient := configFake.NewSimpleClientset()
	meshCatalog := catalog.NewFakeMeshCatalog(kubeClient, configClient)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	proxy, err := getProxy(kubeClient)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}))

	// ---[  Get the config from rds.NewResponse()  ]-------
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("").AnyTimes()

	// mockConfigurator.EXPECT()..Return().AnyTimes()
	mockConfigurator.EXPECT().IsEgressEnabled().Return(false).AnyTimes()

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableWASMStats:        false,
		EnableEgressPolicy:     false,
		EnableMulticlusterMode: true, // ENABLE MULTICLUSTER
		EnableOSMGateway:       true, // ENABLE MULTICLUSTER GATEWAY
	}).AnyTimes()

	return meshCatalog, proxy, proxyRegistry, mockConfigurator, err
}
