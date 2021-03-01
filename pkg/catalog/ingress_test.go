package catalog

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingV1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/endpoint/providers/kube"
	"github.com/openservicemesh/osm/pkg/ingress"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
)

var (
	fakeIngressService         = "fake-service"
	fakeIngressNamespace       = "ingress-ns"
	fakeIngressPort      int32 = 80

	// fakeIngressPaths is a mapping of the fake ingress resource domains to its paths
	fakeIngressPaths = map[string][]string{
		"fake1.com": {"/fake1-path1", "/fake1-path2", "/fake1-path3"},
		"fake2.com": {"/fake2-path1"},
		"*":         {".*"},
	}
)

func newFakeMeshCatalog() *MeshCatalog {
	defer GinkgoRecover()

	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		mockIngressMonitor *ingress.MockMonitor
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	mockIngressMonitor = ingress.NewMockMonitor(mockCtrl)

	meshSpec := smi.NewFakeMeshSpecClient()

	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"

	stop := make(chan struct{})
	endpointProviders := []endpoint.Provider{
		kube.NewFakeProvider(),
	}
	kubeClient := testclient.NewSimpleClientset()

	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

	certManager := tresor.NewFakeCertManager(cfg)

	// Create a Bookstore-v1 pod
	pod := tests.NewPodFixture(tests.Namespace, tests.BookstoreV1Service.Name, tests.BookstoreServiceAccountName, tests.PodLabels)
	if _, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("Error creating new fake Mesh Catalog: %s", err.Error())
	}

	// Create a Bookstore-v2 pod
	pod = tests.NewPodFixture(tests.Namespace, tests.BookstoreV2Service.Name, tests.BookstoreV2ServiceAccountName, tests.PodLabels)
	if _, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("Error creating new fake Mesh Catalog: %s", err.Error())
	}

	// Create Bookstore-v1 Service
	selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
	svc := tests.NewServiceFixture(tests.BookstoreV1Service.Name, tests.BookstoreV1Service.Namespace, selector)
	if _, err := kubeClient.CoreV1().Services(tests.BookstoreV1Service.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("Error creating new Bookstore service: %s", err.Error())
	}

	// Create Bookstore-v2 Service
	svc = tests.NewServiceFixture(tests.BookstoreV2Service.Name, tests.BookstoreV2Service.Namespace, selector)
	if _, err := kubeClient.CoreV1().Services(tests.BookstoreV2Service.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("Error creating new Bookstore service: %s", err.Error())
	}

	// Create Bookbuyer Service
	svc = tests.NewServiceFixture(tests.BookbuyerService.Name, tests.BookbuyerService.Namespace, nil)
	if _, err := kubeClient.CoreV1().Services(tests.BookbuyerService.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("Error creating new Bookbuyer service: %s", err.Error())
	}

	// Create Bookstore apex Service
	svc = tests.NewServiceFixture(tests.BookstoreApexService.Name, tests.BookstoreApexService.Namespace, nil)
	if _, err := kubeClient.CoreV1().Services(tests.BookstoreApexService.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("Error creating new Bookstore Apex service", err.Error())
	}

	mockIngressMonitor.EXPECT().GetIngressResources(gomock.Any()).Return(getFakeIngresses(), nil).AnyTimes()

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

	return NewMeshCatalog(mockKubeController, kubeClient, meshSpec, certManager,
		mockIngressMonitor, stop, cfg, endpointProviders...)
}

func getFakeIngresses() []*networkingV1beta1.Ingress {
	return []*networkingV1beta1.Ingress{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ingress-1",
				Namespace: fakeIngressNamespace,
				Annotations: map[string]string{
					constants.OSMKubeResourceMonitorAnnotation: "enabled",
				},
			},
			Spec: networkingV1beta1.IngressSpec{
				Backend: &networkingV1beta1.IngressBackend{
					ServiceName: fakeIngressService,
					ServicePort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: fakeIngressPort,
					},
				},
				Rules: []networkingV1beta1.IngressRule{
					{
						Host: "fake1.com",
						IngressRuleValue: networkingV1beta1.IngressRuleValue{
							HTTP: &networkingV1beta1.HTTPIngressRuleValue{
								Paths: []networkingV1beta1.HTTPIngressPath{
									{
										Path: "/fake1-path1",
										Backend: networkingV1beta1.IngressBackend{
											ServiceName: fakeIngressService,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: fakeIngressPort,
											},
										},
									},
									{
										Path: "/fake1-path2",
										Backend: networkingV1beta1.IngressBackend{
											ServiceName: fakeIngressService,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: fakeIngressPort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},

		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ingress-2",
				Namespace: fakeIngressNamespace,
				Annotations: map[string]string{
					constants.OSMKubeResourceMonitorAnnotation: "enabled",
				},
			},
			Spec: networkingV1beta1.IngressSpec{
				Backend: &networkingV1beta1.IngressBackend{
					ServiceName: fakeIngressService,
					ServicePort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: fakeIngressPort,
					},
				},
				Rules: []networkingV1beta1.IngressRule{
					{
						Host: "fake2.com",
						IngressRuleValue: networkingV1beta1.IngressRuleValue{
							HTTP: &networkingV1beta1.HTTPIngressRuleValue{
								Paths: []networkingV1beta1.HTTPIngressPath{
									{
										Path: "/fake2-path1",
										Backend: networkingV1beta1.IngressBackend{
											ServiceName: fakeIngressService,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: fakeIngressPort,
											},
										},
									},
								},
							},
						},
					},
					{
						Host: "fake1.com",
						IngressRuleValue: networkingV1beta1.IngressRuleValue{
							HTTP: &networkingV1beta1.HTTPIngressRuleValue{
								Paths: []networkingV1beta1.HTTPIngressPath{
									{
										Path: "/fake1-path3",
										Backend: networkingV1beta1.IngressBackend{
											ServiceName: fakeIngressService,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: fakeIngressPort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func pathContains(allowed []string, path string) bool {
	for _, p := range allowed {
		if path == p {
			return true
		}
	}
	return false
}

func TestGetIngressPoliciesForService(t *testing.T) {
	assert := tassert.New(t)
	mc := newFakeMeshCatalog()
	fakeService := service.MeshService{
		Namespace: fakeIngressNamespace,
		Name:      fakeIngressService,
	}

	inboundIngressPolicies, err := mc.GetIngressPoliciesForService(fakeService)
	assert.Empty(err)

	// The number of ingress inbound policies is the number of unique hosts across the ingress resources :
	// 1. Hostnames: {"*"}
	// 2. Hostnames: {"fake1.com"}
	// 2. Hostnames: {"fake2.com"}
	assert.Equal(len(inboundIngressPolicies), len(fakeIngressPaths))

	for i := 0; i < len(inboundIngressPolicies); i++ {
		for _, rule := range inboundIngressPolicies[i].Rules {
			// For each ingress path, all HTTP methods are allowed, which is a regex match all of '*'
			assert.Len(rule.Route.HTTPRouteMatch.Methods, 1)
			assert.Equal(rule.Route.HTTPRouteMatch.Methods[0], constants.WildcardHTTPMethod)

			//  rule.Route constains the path specified in the ingress resource rule. Since the same service
			// could be a backend for multiple ingress resources, we don't know which ingress resource
			// this path corresponds to. In order to not make assumptions
			// on the implementation of 'GetIngressPoliciesForServices()', we relax the check here
			// to match on any of the ingress paths corresponding to the host.
			assert.True(pathContains(fakeIngressPaths[inboundIngressPolicies[i].Hostnames[0]], rule.Route.HTTPRouteMatch.PathRegex))

			// Allowed service accounts should be wildcarded with an empty service account for ingress rules
			assert.NotNil(rule.AllowedServiceAccounts)
			assert.Equal(1, rule.AllowedServiceAccounts.Cardinality()) // single wildcard service account
			allowedSvcAccounts := rule.AllowedServiceAccounts.Pop().(service.K8sServiceAccount)
			assert.True((allowedSvcAccounts.IsEmpty()))
		}
	}
}

func TestBuildIngressPolicyName(t *testing.T) {
	assert := tassert.New(t)
	testCases := []struct {
		name         string
		namespace    string
		host         string
		expectedName string
	}{
		{
			name:         "bookbuyer",
			namespace:    "default",
			host:         "*",
			expectedName: "bookbuyer.default|*",
		},
		{
			name:         "bookbuyer",
			namespace:    "bookbuyer-ns",
			host:         "foobar.com",
			expectedName: "bookbuyer.bookbuyer-ns|foobar.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := buildIngressPolicyName(tc.name, tc.namespace, tc.host)
			assert.Equal(tc.expectedName, actual)
		})
	}
}
