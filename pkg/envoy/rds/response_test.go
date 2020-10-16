package rds

import (
	"context"
	"fmt"
	"strings"

	set "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
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
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	testHeaderKey1 = "test-header-1"
	testHeaderKey2 = "test-header-2"
)

var _ = Describe("Construct RoutePolicyWeightedClusters object", func() {
	Context("Testing the creating of a RoutePolicyWeightedClusters object", func() {
		It("Returns RoutePolicyWeightedClusters", func() {

			weightedCluster := service.WeightedCluster{
				ClusterName: service.ClusterName("osm/bookstore-1"),
				Weight:      constants.ClusterWeightAcceptAll,
			}
			routePolicy := trafficpolicy.HTTPRoute{
				PathRegex: "/books-bought",
				Methods:   []string{"GET"},
			}

			routePolicyWeightedClusters := createRoutePolicyWeightedClusters(routePolicy, weightedCluster)
			Expect(routePolicyWeightedClusters).NotTo(Equal(nil))
			Expect(routePolicyWeightedClusters.HTTPRoute.PathRegex).To(Equal("/books-bought"))
			Expect(routePolicyWeightedClusters.HTTPRoute.Methods).To(Equal([]string{"GET"}))
			Expect(routePolicyWeightedClusters.WeightedClusters.Cardinality()).To(Equal(1))
			routePolicyWeightedClustersSlice := routePolicyWeightedClusters.WeightedClusters.ToSlice()
			Expect(string(routePolicyWeightedClustersSlice[0].(service.WeightedCluster).ClusterName)).To(Equal("osm/bookstore-1"))
			Expect(routePolicyWeightedClustersSlice[0].(service.WeightedCluster).Weight).To(Equal(100))
		})
	})
})

var _ = Describe("AggregateRoutesByDomain", func() {
	domainRoutesMap := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)
	weightedClustersMap := set.NewSet()
	Context("Building a map of routes by domain", func() {
		It("Returns a new aggregated map of domain and routes", func() {

			weightedCluster := service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 100}
			routePolicies := []trafficpolicy.HTTPRoute{
				{PathRegex: "/books-bought", Methods: []string{"GET"}},
				{PathRegex: "/buy-a-book", Methods: []string{"GET"}},
			}
			weightedClustersMap.Add(weightedCluster)

			for _, routePolicy := range routePolicies {
				aggregateRoutesByHost(domainRoutesMap, routePolicy, weightedCluster, "bookstore.mesh")
			}
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(2))

			for _, routePolicy := range routePolicies {
				_, routePolicyExists := domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex]
				Expect(routePolicyExists).To(Equal(true))
			}
			for path := range domainRoutesMap["bookstore.mesh"] {
				Expect(domainRoutesMap["bookstore.mesh"][path].WeightedClusters.Cardinality()).To(Equal(1))
				Expect(domainRoutesMap["bookstore.mesh"][path].WeightedClusters.Equal(weightedClustersMap)).To(Equal(true))
			}
		})
	})

	Context("Adding a route to existing domain in the map", func() {
		It("Returns the map of with a new route on the domain", func() {

			weightedCluster := service.WeightedCluster{
				ClusterName: service.ClusterName("osm/bookstore-1"),
				Weight:      constants.ClusterWeightAcceptAll,
			}
			routePolicy := trafficpolicy.HTTPRoute{
				PathRegex: "/update-books-bought",
				Methods:   []string{"GET"},
				Headers: map[string]string{
					testHeaderKey1: "This is a test header 1",
				},
			}
			weightedClustersMap.Add(weightedCluster)

			aggregateRoutesByHost(domainRoutesMap, routePolicy, weightedCluster, "bookstore.mesh")
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(3))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].HTTPRoute).To(Equal(routePolicy))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].WeightedClusters.Cardinality()).To(Equal(1))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].HTTPRoute).To(Equal(trafficpolicy.HTTPRoute{PathRegex: "/update-books-bought", Methods: []string{"GET"}, Headers: map[string]string{testHeaderKey1: "This is a test header 1"}}))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].WeightedClusters.Equal(weightedClustersMap)).To(Equal(true))
		})
	})

	Context("Adding a cluster to an existing route to existing domain in the map", func() {
		It("Returns the map of with a new weighted cluster on a route in the domain", func() {

			weightedCluster := service.WeightedCluster{
				ClusterName: service.ClusterName("osm/bookstore-2"),
				Weight:      constants.ClusterWeightAcceptAll,
			}
			routePolicy := trafficpolicy.HTTPRoute{
				PathRegex: "/update-books-bought",
				Methods:   []string{"GET"},
				Headers: map[string]string{
					testHeaderKey2: "This is a test header 2",
				},
			}
			weightedClustersMap.Add(weightedCluster)

			aggregateRoutesByHost(domainRoutesMap, routePolicy, weightedCluster, "bookstore.mesh")
			Expect(domainRoutesMap).NotTo(Equal(nil))
			Expect(len(domainRoutesMap)).To(Equal(1))
			Expect(len(domainRoutesMap["bookstore.mesh"])).To(Equal(3))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].WeightedClusters.Cardinality()).To(Equal(2))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].HTTPRoute).To(Equal(trafficpolicy.HTTPRoute{PathRegex: "/update-books-bought", Methods: []string{"GET", "GET"}, Headers: map[string]string{testHeaderKey1: "This is a test header 1", testHeaderKey2: "This is a test header 2"}}))
			Expect(domainRoutesMap["bookstore.mesh"][routePolicy.PathRegex].WeightedClusters.Equal(weightedClustersMap)).To(Equal(true))
		})
	})
})

var _ = Describe("RDS Response", func() {
	defer GinkgoRecover()

	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		mockIngressMonitor *ingress.MockMonitor
	)
	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	mockIngressMonitor = ingress.NewMockMonitor(mockCtrl)

	endpointProviders := []endpoint.Provider{kube.NewFakeProvider()}
	kubeClient := testclient.NewSimpleClientset()

	stop := make(<-chan struct{})
	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"
	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, cfg)

	testChan := make(chan interface{})

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
		GinkgoT().Fatalf("Error creating new Bookstire Apex service: %s", err.Error())
	}

	mockIngressMonitor.EXPECT().GetIngressResources(gomock.Any()).Return(nil, nil).AnyTimes()
	mockIngressMonitor.EXPECT().GetAnnouncementsChannel().Return(testChan).AnyTimes()

	// Monitored namespaces is made a set to make sure we don't repeat namespaces on mock
	listExpectedNs := tests.GetUnique([]string{
		tests.BookstoreV1Service.Namespace,
		tests.BookbuyerService.Namespace,
		tests.BookstoreApexService.Namespace,
	})

	mockKubeController.EXPECT().ListServices().DoAndReturn(func() []*corev1.Service {
		// Play pretend this is in the controller cache
		var services []*corev1.Service

		for _, ns := range listExpectedNs {
			svcList, _ := kubeClient.CoreV1().Services(ns).List(context.TODO(), metav1.ListOptions{})
			for serviceIdx := range svcList.Items {
				services = append(services, &svcList.Items[serviceIdx])
			}
		}

		return services
	}).AnyTimes()
	mockKubeController.EXPECT().GetService(gomock.Any()).DoAndReturn(func(msh service.MeshService) *v1.Service {
		// Play pretend this is in the controller cache
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
	mockKubeController.EXPECT().GetAnnouncementsChannel(k8s.Namespaces).Return(testChan).AnyTimes()
	mockKubeController.EXPECT().GetAnnouncementsChannel(k8s.Services).Return(testChan).AnyTimes()
	mockKubeController.EXPECT().ListMonitoredNamespaces().Return(listExpectedNs, nil).AnyTimes()

	meshCatalog := catalog.NewMeshCatalog(mockKubeController, kubeClient, smi.NewFakeMeshSpecClient(), certManager,
		mockIngressMonitor, make(<-chan struct{}), cfg, endpointProviders...)

	Context("Test GetHostnamesForService", func() {
		contains := func(domains []string, expected string) bool {
			for _, entry := range domains {
				if entry == expected {
					return true
				}
			}
			return false
		}
		It("returns the domain for a service when traffic split is specified for the given service", func() {

			actual, err := meshCatalog.GetHostnamesForService(tests.BookstoreV1Service)
			Expect(err).ToNot(HaveOccurred())

			domainList := strings.Split(actual, ",")
			Expect(len(domainList)).To(Equal(10))
			Expect(contains(domainList, tests.BookstoreApexServiceName)).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s:%d", tests.BookstoreApexServiceName, tests.ServicePort))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s", tests.BookstoreApexServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s:%d", tests.BookstoreApexServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc", tests.BookstoreApexServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc:%d", tests.BookstoreApexServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc.cluster", tests.BookstoreApexServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc.cluster:%d", tests.BookstoreApexServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc.cluster.local", tests.BookstoreApexServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc.cluster.local:%d", tests.BookstoreApexServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
		})

		It("returns a list of domains for a service when traffic split is not specified for the given service", func() {

			actual, err := meshCatalog.GetHostnamesForService(tests.BookbuyerService)
			Expect(err).ToNot(HaveOccurred())

			domainList := strings.Split(actual, ",")
			Expect(len(domainList)).To(Equal(10))

			Expect(contains(domainList, tests.BookbuyerServiceName)).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s:%d", tests.BookbuyerServiceName, tests.ServicePort))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc.cluster", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc.cluster:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc.cluster.local", tests.BookbuyerServiceName, tests.Namespace))).To(BeTrue())
			Expect(contains(domainList, fmt.Sprintf("%s.%s.svc.cluster.local:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort))).To(BeTrue())
		})

		It("No service found when mesh does not have service", func() {
			_, err := meshCatalog.GetHostnamesForService(tests.BookwarehouseService)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("GetWeightedClusterForService", func() {
		It("returns weighted cluster for service from traffic split", func() {

			actual, err := meshCatalog.GetWeightedClusterForService(tests.BookstoreV1Service)
			Expect(err).ToNot(HaveOccurred())

			expected := service.WeightedCluster{
				ClusterName: service.ClusterName(tests.BookstoreV1WeightedService.Service.String()),
				Weight:      tests.BookstoreV1WeightedService.Weight,
			}
			Expect(actual).To(Equal(expected))
		})

		It("returns weighted cluster for service with default weight of 100", func() {

			actual, err := meshCatalog.GetWeightedClusterForService(tests.BookbuyerService)
			Expect(err).ToNot(HaveOccurred())

			expected := service.WeightedCluster{
				ClusterName: service.ClusterName(tests.BookbuyerService.String()),
				Weight:      tests.Weight,
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
