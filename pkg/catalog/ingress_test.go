package catalog

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	extensionsV1beta "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
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
		"fake1.com": {"/fake1-path1", "/fake1-path2"},
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

	// Create a pod
	pod := tests.NewPodFixture(tests.Namespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
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

	announcementsChan := make(chan announcements.Announcement)

	mockIngressMonitor.EXPECT().GetAnnouncementsChannel().Return(announcementsChan).AnyTimes()
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
	mockKubeController.EXPECT().GetService(gomock.Any()).DoAndReturn(func(msh service.MeshService) *v1.Service {
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

func getFakeIngresses() []*extensionsV1beta.Ingress {
	return []*extensionsV1beta.Ingress{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ingress-1",
				Namespace: fakeIngressNamespace,
				Annotations: map[string]string{
					constants.OSMKubeResourceMonitorAnnotation: "enabled",
				},
			},
			Spec: extensionsV1beta.IngressSpec{
				Backend: &extensionsV1beta.IngressBackend{
					ServiceName: fakeIngressService,
					ServicePort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: fakeIngressPort,
					},
				},
				Rules: []extensionsV1beta.IngressRule{
					{
						Host: "fake1.com",
						IngressRuleValue: extensionsV1beta.IngressRuleValue{
							HTTP: &extensionsV1beta.HTTPIngressRuleValue{
								Paths: []extensionsV1beta.HTTPIngressPath{
									{
										Path: "/fake1-path1",
										Backend: extensionsV1beta.IngressBackend{
											ServiceName: fakeIngressService,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: fakeIngressPort,
											},
										},
									},
									{
										Path: "/fake1-path2",
										Backend: extensionsV1beta.IngressBackend{
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
			Spec: extensionsV1beta.IngressSpec{
				Rules: []extensionsV1beta.IngressRule{
					{
						Host: "fake2.com",
						IngressRuleValue: extensionsV1beta.IngressRuleValue{
							HTTP: &extensionsV1beta.HTTPIngressRuleValue{
								Paths: []extensionsV1beta.HTTPIngressPath{
									{
										Path: "/fake2-path1",
										Backend: extensionsV1beta.IngressBackend{
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

var _ = Describe("Test ingress route policies", func() {
	Context("Testing GetIngressRoutesPerHost", func() {
		mc := newFakeMeshCatalog()
		It("Gets the route policies per domain from multiple ingress resources corresponding to a service", func() {
			fakeService := service.MeshService{
				Namespace: fakeIngressNamespace,
				Name:      fakeIngressService,
			}

			domainRoutesMap, _ := mc.GetIngressRoutesPerHost(fakeService)

			for domain, routePolicies := range domainRoutesMap {
				// The number of route policies per domain is the product of the number of rules and paths per rule
				Expect(len(routePolicies)).To(Equal(len(fakeIngressPaths[domain])))
				for _, routePolicy := range routePolicies {

					// For each ingress path, all HTTP methods are allowed, which is a regex match all of '*'
					Expect(len(routePolicy.Methods)).To(Equal(1))

					Expect(routePolicy.Methods[0]).To(Equal(constants.RegexMatchAll))

					// routePolicy.Path is the path specified in the ingress resource rule. Since the same service
					// could be a backend for multiple ingress resources, we don't know which ingress resource
					// this path corresponds to just from 'domainRoutesMap'. In order to not make assumptions
					// on the implementation of 'GetIngressRoutesPerHost()', we relax the check here
					// to match on any of the ingress paths corresponding to the domain.
					Expect(pathContains(fakeIngressPaths[domain], routePolicy.PathRegex)).To(BeTrue())
				}
			}
		})

	})
})
