package catalog

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	specs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/policy"
	kubeFake "github.com/openservicemesh/osm/pkg/providers/kube/fake"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
)

type testParams struct {
	permissiveMode bool
}

func newFakeMeshCatalogForRoutes(t *testing.T, testParams testParams) *MeshCatalog {
	mockCtrl := gomock.NewController(t)
	kubeClient := testclient.NewSimpleClientset()

	mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockKubeController := k8s.NewMockController(mockCtrl)
	mockPolicyController := policy.NewMockController(mockCtrl)
	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()
	mockConfigurator.EXPECT().GetOSMNamespace().Return("osm-system").AnyTimes()

	provider := kubeFake.NewFakeProvider()
	endpointProviders := []endpoint.Provider{
		provider,
	}
	serviceProviders := []service.Provider{
		provider,
	}

	stop := make(chan struct{})

	certManager := tresorFake.NewFake(nil)

	// Create a bookstoreV1 pod
	bookstoreV1Pod := tests.NewPodFixture(tests.BookstoreV1Service.Namespace, tests.BookstoreV1Service.Name, tests.BookstoreServiceAccountName, tests.PodLabels)
	if _, err := kubeClient.CoreV1().Pods(tests.BookstoreV1Service.Namespace).Create(context.TODO(), &bookstoreV1Pod, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Error creating new pod: %s", err.Error())
	}

	// Create a bookstoreV2 pod
	bookstoreV2Pod := tests.NewPodFixture(tests.BookstoreV2Service.Namespace, tests.BookstoreV2Service.Name, tests.BookstoreV2ServiceAccountName, tests.PodLabels)
	if _, err := kubeClient.CoreV1().Pods(tests.BookstoreV2Service.Namespace).Create(context.TODO(), &bookstoreV2Pod, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Error creating new pod: %s", err.Error())
	}

	// Create a bookbuyer pod
	bookbuyerPod := tests.NewPodFixture(tests.BookbuyerService.Namespace, tests.BookbuyerService.Name, tests.BookbuyerServiceAccountName, tests.PodLabels)
	if _, err := kubeClient.CoreV1().Pods(tests.BookbuyerService.Namespace).Create(context.TODO(), &bookbuyerPod, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Error creating new pod: %s", err.Error())
	}

	// Create Bookstore-v1 Service
	svc := tests.NewServiceFixture(tests.BookstoreV1Service.Name, tests.BookstoreV1Service.Namespace, map[string]string{constants.AppLabel: "bookstore", "version": "v1"})
	if _, err := kubeClient.CoreV1().Services(tests.BookstoreV1Service.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Error creating new Bookstore v1 service: %s", err.Error())
	}

	// Create Bookstore-v2 Service
	svc = tests.NewServiceFixture(tests.BookstoreV2Service.Name, tests.BookstoreV2Service.Namespace, map[string]string{constants.AppLabel: "bookstore", "version": "v2"})
	if _, err := kubeClient.CoreV1().Services(tests.BookstoreV2Service.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Error creating new Bookstore v2 service: %s", err.Error())
	}

	// Create Bookstore-apex Service
	svc = tests.NewServiceFixture(tests.BookstoreApexService.Name, tests.BookstoreApexService.Namespace, map[string]string{constants.AppLabel: "bookstore"})
	if _, err := kubeClient.CoreV1().Services(tests.BookstoreApexService.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Error creating new Bookstore Apex service: %s", err.Error())
	}

	// Create Bookbuyer Service
	svc = tests.NewServiceFixture(tests.BookbuyerService.Name, tests.BookbuyerService.Namespace, nil)
	if _, err := kubeClient.CoreV1().Services(tests.BookbuyerService.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Error creating new Bookbuyer service: %s", err.Error())
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
	mockKubeController.EXPECT().ListMonitoredNamespaces().Return(listExpectedNs, nil).AnyTimes()

	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(testParams.permissiveMode).AnyTimes()
	mockConfigurator.EXPECT().GetConfigResyncInterval().Return(time.Duration(0)).AnyTimes()

	mockMeshSpec.EXPECT().ListTrafficTargets().Return([]*access.TrafficTarget{&tests.TrafficTarget, &tests.BookstoreV2TrafficTarget}).AnyTimes()
	mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*specs.HTTPRouteGroup{&tests.HTTPRouteGroup}).AnyTimes()
	mockMeshSpec.EXPECT().ListTrafficSplits().Return([]*split.TrafficSplit{}).AnyTimes()

	return NewMeshCatalog(mockKubeController, mockMeshSpec, certManager,
		mockPolicyController, stop, mockConfigurator, serviceProviders, endpointProviders, messaging.NewBroker(stop))
}
