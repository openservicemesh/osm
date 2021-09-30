package smi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	testTrafficTargetClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned/fake"
	testTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned/fake"
	testTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned/fake"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	httpserverconstants "github.com/openservicemesh/osm/pkg/httpserver/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

const (
	testNamespaceName = "test"
)

type fakeKubeClientSet struct {
	kubeClient                *testclient.Clientset
	smiTrafficSplitClientSet  *testTrafficSplitClient.Clientset
	smiTrafficSpecClientSet   *testTrafficSpecClient.Clientset
	smiTrafficTargetClientSet *testTrafficTargetClient.Clientset
}

func bootstrapClient() (MeshSpec, *fakeKubeClientSet, error) {
	defer GinkgoRecover()

	osmNamespace := "osm-system"
	meshName := "osm"
	stop := make(chan struct{})
	kubeClient := testclient.NewSimpleClientset()
	smiTrafficSplitClientSet := testTrafficSplitClient.NewSimpleClientset()
	smiTrafficSpecClientSet := testTrafficSpecClient.NewSimpleClientset()
	smiTrafficTargetClientSet := testTrafficTargetClient.NewSimpleClientset()
	kubernetesClient, err := k8s.NewKubernetesController(kubeClient, nil, meshName, stop)
	if err != nil {
		GinkgoT().Fatalf("Error initializing kubernetes controller: %s", err.Error())
	}

	fakeClientSet := &fakeKubeClientSet{
		kubeClient:                kubeClient,
		smiTrafficSplitClientSet:  smiTrafficSplitClientSet,
		smiTrafficSpecClientSet:   smiTrafficSpecClientSet,
		smiTrafficTargetClientSet: smiTrafficTargetClientSet,
	}

	// Create a test namespace that is monitored
	testNamespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespaceName,
			Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: meshName}, // Label selectors don't work with fake clients, only here to signify its importance
		},
	}
	if _, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), &testNamespace, metav1.CreateOptions{}); err != nil {
		GinkgoT().Fatalf("Error creating Namespace %v: %s", testNamespace, err.Error())
	}

	meshSpec, err := newSMIClient(
		kubeClient,
		smiTrafficSplitClientSet,
		smiTrafficSpecClientSet,
		smiTrafficTargetClientSet,
		osmNamespace,
		kubernetesClient,
		kubernetesClientName,
		stop,
	)

	return meshSpec, fakeClientSet, err
}

var _ = Describe("When listing TrafficSplit", func() {
	var (
		meshSpec      MeshSpec
		fakeClientSet *fakeKubeClientSet
		err           error
	)
	BeforeEach(func() {
		meshSpec, fakeClientSet, err = bootstrapClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return a list of traffic split resources", func() {
		tsChannel := events.Subscribe(announcements.TrafficSplitAdded,
			announcements.TrafficSplitDeleted,
			announcements.TrafficSplitUpdated)
		defer events.Unsub(tsChannel)

		split := &smiSplit.TrafficSplit{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ListTrafficSplits",
				Namespace: testNamespaceName,
			},
			Spec: smiSplit.TrafficSplitSpec{
				Service: tests.BookstoreApexServiceName,
				Backends: []smiSplit.TrafficSplitBackend{
					{
						Service: tests.BookstoreV1ServiceName,
						Weight:  tests.Weight90,
					},
					{
						Service: tests.BookstoreV2ServiceName,
						Weight:  tests.Weight10,
					},
				},
			},
		}

		_, err := fakeClientSet.smiTrafficSplitClientSet.SplitV1alpha2().TrafficSplits(testNamespaceName).Create(context.TODO(), split, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-tsChannel

		splits := meshSpec.ListTrafficSplits()
		Expect(len(splits)).To(Equal(1))
		Expect(split).To(Equal(splits[0]))

		// --
		// Verify filter for apex service
		filteredApexAvailable := meshSpec.ListTrafficSplits(WithTrafficSplitApexService(service.MeshService{Name: tests.BookstoreApexServiceName, Namespace: testNamespaceName}))
		Expect(len(filteredApexAvailable)).To(Equal(1))
		Expect(split).To(Equal(filteredApexAvailable[0]))
		filteredApexUnavailable := meshSpec.ListTrafficSplits(WithTrafficSplitApexService(tests.BookstoreV1Service))
		Expect(len(filteredApexUnavailable)).To(Equal(0))
		// Verify filter for backend service
		filteredBackendAvailable := meshSpec.ListTrafficSplits(WithTrafficSplitBackendService(service.MeshService{Name: tests.BookstoreV1ServiceName, Namespace: testNamespaceName}))
		Expect(len(filteredBackendAvailable)).To(Equal(1))
		Expect(split).To(Equal(filteredBackendAvailable[0]))
		filteredBackendNameMismatch := meshSpec.ListTrafficSplits(WithTrafficSplitBackendService(service.MeshService{Namespace: testNamespaceName, Name: "invalid"}))
		Expect(len(filteredBackendNameMismatch)).To(Equal(0))
		filteredBackendNamespaceMismatch := meshSpec.ListTrafficSplits(WithTrafficSplitBackendService(service.MeshService{Namespace: "invalid", Name: tests.BookstoreV1ServiceName}))
		Expect(len(filteredBackendNamespaceMismatch)).To(Equal(0))

		err = fakeClientSet.smiTrafficSplitClientSet.SplitV1alpha2().TrafficSplits(testNamespaceName).Delete(context.TODO(), split.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-tsChannel
	})
})

var _ = Describe("When listing ServiceAccounts", func() {
	var (
		meshSpec      MeshSpec
		fakeClientSet *fakeKubeClientSet
		err           error
	)
	BeforeEach(func() {
		meshSpec, fakeClientSet, err = bootstrapClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return a list of service accounts specified in TrafficTarget resources", func() {
		ttChannel := events.Subscribe(announcements.TrafficTargetAdded,
			announcements.TrafficTargetDeleted,
			announcements.TrafficTargetUpdated)
		defer events.Unsub(ttChannel)

		trafficTarget := &smiAccess.TrafficTarget{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "access.smi-spec.io/v1alpha3",
				Kind:       "TrafficTarget",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ListServiceAccounts",
				Namespace: testNamespaceName,
			},
			Spec: smiAccess.TrafficTargetSpec{
				Destination: smiAccess.IdentityBindingSubject{
					Kind:      "ServiceAccount",
					Name:      tests.BookstoreServiceAccountName,
					Namespace: testNamespaceName,
				},
				Sources: []smiAccess.IdentityBindingSubject{{
					Kind:      "ServiceAccount",
					Name:      tests.BookbuyerServiceAccountName,
					Namespace: testNamespaceName,
				}},
				Rules: []smiAccess.TrafficTargetRule{{
					Kind:    "HTTPRouteGroup",
					Name:    tests.RouteGroupName,
					Matches: []string{tests.BuyBooksMatchName},
				}},
			},
		}

		_, err := fakeClientSet.smiTrafficTargetClientSet.AccessV1alpha3().TrafficTargets(testNamespaceName).Create(context.TODO(), trafficTarget, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-ttChannel

		svcAccounts := meshSpec.ListServiceAccounts()

		numExpectedSvcAccounts := len(trafficTarget.Spec.Sources) + 1 // 1 for the destination ServiceAccount
		Expect(len(svcAccounts)).To(Equal(numExpectedSvcAccounts))

		err = fakeClientSet.smiTrafficTargetClientSet.AccessV1alpha3().TrafficTargets(testNamespaceName).Delete(context.TODO(), trafficTarget.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-ttChannel
	})
})

var _ = Describe("When listing TrafficTargets", func() {
	var (
		meshSpec      MeshSpec
		fakeClientSet *fakeKubeClientSet
		err           error
	)
	BeforeEach(func() {
		meshSpec, fakeClientSet, err = bootstrapClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("Returns a list of TrafficTarget resources", func() {
		ttChannel := events.Subscribe(announcements.TrafficTargetAdded,
			announcements.TrafficTargetDeleted,
			announcements.TrafficTargetUpdated)
		defer events.Unsub(ttChannel)

		trafficTarget := &smiAccess.TrafficTarget{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "access.smi-spec.io/v1alpha3",
				Kind:       "TrafficTarget",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ListTrafficTargets",
				Namespace: testNamespaceName,
			},
			Spec: smiAccess.TrafficTargetSpec{
				Destination: smiAccess.IdentityBindingSubject{
					Kind:      "ServiceAccount",
					Name:      tests.BookstoreServiceAccountName,
					Namespace: testNamespaceName,
				},
				Sources: []smiAccess.IdentityBindingSubject{{
					Kind:      "ServiceAccount",
					Name:      tests.BookbuyerServiceAccountName,
					Namespace: testNamespaceName,
				}},
				Rules: []smiAccess.TrafficTargetRule{{
					Kind:    "HTTPRouteGroup",
					Name:    tests.RouteGroupName,
					Matches: []string{tests.BuyBooksMatchName},
				}},
			},
		}

		_, err := fakeClientSet.smiTrafficTargetClientSet.AccessV1alpha3().TrafficTargets(testNamespaceName).Create(context.TODO(), trafficTarget, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-ttChannel

		targets := meshSpec.ListTrafficTargets()
		Expect(len(targets)).To(Equal(1))

		// Verify destination based filtering
		filteredAvailable := meshSpec.ListTrafficTargets(WithTrafficTargetDestination(identity.K8sServiceAccount{Namespace: testNamespaceName, Name: tests.BookstoreServiceAccountName}))
		Expect(len(filteredAvailable)).To(Equal(1))
		filteredUnavailable := meshSpec.ListTrafficTargets(WithTrafficTargetDestination(identity.K8sServiceAccount{Namespace: testNamespaceName, Name: "unavailable"}))
		Expect(len(filteredUnavailable)).To(Equal(0))

		err = fakeClientSet.smiTrafficTargetClientSet.AccessV1alpha3().TrafficTargets(testNamespaceName).Delete(context.TODO(), trafficTarget.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-ttChannel
	})
})

var _ = Describe("When listing ListHTTPTrafficSpecs", func() {
	var (
		meshSpec      MeshSpec
		fakeClientSet *fakeKubeClientSet
		err           error
	)
	BeforeEach(func() {
		meshSpec, fakeClientSet, err = bootstrapClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("Returns an empty list when no HTTPRouteGroup are found", func() {
		services := meshSpec.ListHTTPTrafficSpecs()
		Expect(len(services)).To(Equal(0))
	})

	It("should return a list of ListHTTPTrafficSpecs resources", func() {
		rgChannel := events.Subscribe(announcements.RouteGroupAdded,
			announcements.RouteGroupDeleted,
			announcements.RouteGroupUpdated)
		defer events.Unsub(rgChannel)

		routeSpec := &smiSpecs.HTTPRouteGroup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "specs.smi-spec.io/v1alpha4",
				Kind:       "HTTPRouteGroup",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespaceName,
				Name:      "test-ListHTTPTrafficSpecs",
			},
			Spec: smiSpecs.HTTPRouteGroupSpec{
				Matches: []smiSpecs.HTTPMatch{
					{
						Name:      tests.BuyBooksMatchName,
						PathRegex: tests.BookstoreBuyPath,
						Methods:   []string{"GET"},
						Headers: map[string]string{
							"user-agent": tests.HTTPUserAgent,
						},
					},
					{
						Name:      tests.SellBooksMatchName,
						PathRegex: tests.BookstoreSellPath,
						Methods:   []string{"GET"},
					},
					{
						Name: tests.WildcardWithHeadersMatchName,
						Headers: map[string]string{
							"user-agent": tests.HTTPUserAgent,
						},
					},
				},
			},
		}

		_, err := fakeClientSet.smiTrafficSpecClientSet.SpecsV1alpha4().HTTPRouteGroups(testNamespaceName).Create(context.TODO(), routeSpec, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-rgChannel

		httpRoutes := meshSpec.ListHTTPTrafficSpecs()
		Expect(len(httpRoutes)).To(Equal(1))
		Expect(httpRoutes[0].Name).To(Equal(routeSpec.Name))

		err = fakeClientSet.smiTrafficSpecClientSet.SpecsV1alpha4().HTTPRouteGroups(testNamespaceName).Delete(context.TODO(), routeSpec.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-rgChannel
	})
})

var _ = Describe("When listing TCP routes", func() {
	var (
		meshSpec      MeshSpec
		fakeClientSet *fakeKubeClientSet
		err           error
	)
	BeforeEach(func() {
		meshSpec, fakeClientSet, err = bootstrapClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("Returns an empty list when no TCPRoute resources are found", func() {
		services := meshSpec.ListTCPTrafficSpecs()
		Expect(len(services)).To(Equal(0))
	})

	It("should return a list of TCPRoute resources", func() {
		trChannel := events.Subscribe(announcements.TCPRouteAdded,
			announcements.TCPRouteDeleted,
			announcements.TCPRouteUpdated)
		defer events.Unsub(trChannel)
		routeSpec := &smiSpecs.TCPRoute{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "specs.smi-spec.io/v1alpha4",
				Kind:       "TCPRoute",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespaceName,
				Name:      "tcp-route",
			},
			Spec: smiSpecs.TCPRouteSpec{},
		}

		_, err := fakeClientSet.smiTrafficSpecClientSet.SpecsV1alpha4().TCPRoutes(testNamespaceName).Create(context.TODO(), routeSpec, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-trChannel

		tcpRoutes := meshSpec.ListTCPTrafficSpecs()
		Expect(len(tcpRoutes)).To(Equal(1))
		Expect(tcpRoutes[0].Name).To(Equal(routeSpec.Name))

		err = fakeClientSet.smiTrafficSpecClientSet.SpecsV1alpha4().TCPRoutes(testNamespaceName).Delete(context.TODO(), routeSpec.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-trChannel
	})
})

var _ = Describe("When getting a TCP route by its namespaced name", func() {
	var (
		meshSpec      MeshSpec
		fakeClientSet *fakeKubeClientSet
		err           error
	)
	BeforeEach(func() {
		meshSpec, fakeClientSet, err = bootstrapClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return nil when a TCP route is not found", func() {
		tcpRoute := meshSpec.GetTCPRoute("ns/route")
		Expect(tcpRoute).To(BeNil())
	})

	It("should return a non nil TCPRoute when found", func() {
		trChannel := events.Subscribe(announcements.TCPRouteAdded,
			announcements.TCPRouteDeleted,
			announcements.TCPRouteUpdated)
		defer events.Unsub(trChannel)
		routeSpec := &smiSpecs.TCPRoute{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "specs.smi-spec.io/v1alpha4",
				Kind:       "TCPRoute",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespaceName,
				Name:      "tcp-route",
			},
			Spec: smiSpecs.TCPRouteSpec{},
		}

		_, err := fakeClientSet.smiTrafficSpecClientSet.SpecsV1alpha4().TCPRoutes(testNamespaceName).Create(context.TODO(), routeSpec, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-trChannel

		tcpRoute := meshSpec.GetTCPRoute(fmt.Sprintf("%s/%s", routeSpec.Namespace, routeSpec.Name))
		Expect(tcpRoute).ToNot(BeNil())
		Expect(tcpRoute).To(Equal(routeSpec))
	})
})

var _ = Describe("When getting an HTTP route by its namespaced name", func() {
	var (
		meshSpec      MeshSpec
		fakeClientSet *fakeKubeClientSet
		err           error
	)
	BeforeEach(func() {
		meshSpec, fakeClientSet, err = bootstrapClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return nil when a HTTP route is not found", func() {
		route := meshSpec.GetHTTPRouteGroup("ns/route")
		Expect(route).To(BeNil())
	})

	It("should return a non nil HTTPRouteGroup when found", func() {
		trChannel := events.Subscribe(announcements.RouteGroupAdded,
			announcements.RouteGroupDeleted,
			announcements.RouteGroupUpdated)
		defer events.Unsub(trChannel)
		routeSpec := &smiSpecs.HTTPRouteGroup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "specs.smi-spec.io/v1alpha4",
				Kind:       "HTTPRouteGroup",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespaceName,
				Name:      "test-GetHTTPRouteGroup",
			},
			Spec: smiSpecs.HTTPRouteGroupSpec{
				Matches: []smiSpecs.HTTPMatch{
					{
						Name:      tests.SellBooksMatchName,
						PathRegex: tests.BookstoreSellPath,
						Methods:   []string{"GET"},
					},
				},
			},
		}

		_, err := fakeClientSet.smiTrafficSpecClientSet.SpecsV1alpha4().HTTPRouteGroups(testNamespaceName).Create(context.TODO(), routeSpec, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-trChannel

		route := meshSpec.GetHTTPRouteGroup(fmt.Sprintf("%s/%s", routeSpec.Namespace, routeSpec.Name))
		Expect(route).ToNot(BeNil())
		Expect(route).To(Equal(routeSpec))
	})
})

var _ = Describe("Test httpserver with probes", func() {
	var (
		testServer         *httptest.Server
		url                = "http://localhost"
		testHTTPServerPort = 8888
		smiVerionPath      = httpserverconstants.SmiVersionPath
		recordCall         = func(ts *httptest.Server, path string) *http.Response {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			ts.Config.Handler.ServeHTTP(w, req)

			return w.Result()
		}
	)

	BeforeEach(func() {
		handlers := map[string]http.Handler{
			smiVerionPath: GetSmiClientVersionHTTPHandler(),
		}

		router := http.NewServeMux()
		for path, handler := range handlers {
			router.Handle(path, handler)
		}

		testServer = &httptest.Server{
			Config: &http.Server{
				Addr:    fmt.Sprintf(":%d", testHTTPServerPort),
				Handler: router,
			},
		}
	})

	It("should result in a successful smi version probe", func() {
		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, smiVerionPath))
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("should result in probe response body with smi version info", func() {
		expectedSmiVersionInfo := map[string]string{
			"TrafficTarget":  smiAccess.SchemeGroupVersion.String(),
			"HTTPRouteGroup": smiSpecs.SchemeGroupVersion.String(),
			"TCPRoute":       smiSpecs.SchemeGroupVersion.String(),
			"TrafficSplit":   smiSplit.SchemeGroupVersion.String(),
		}
		var actualSmiVersionInfo map[string]string

		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, smiVerionPath))

		if err := json.NewDecoder(resp.Body).Decode(&actualSmiVersionInfo); err != nil {
			Fail("Error json decoding smi version info from response body")
		}

		for k, expectedValue := range expectedSmiVersionInfo {
			Expect(expectedValue).To(Equal(actualSmiVersionInfo[k]))
		}
	})
})

func TestHasValidRules(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name           string
		expectedResult bool
		rules          []smiAccess.TrafficTargetRule
	}{
		{
			name:           "has no rules",
			expectedResult: false,
			rules:          []smiAccess.TrafficTargetRule{},
		},
		{
			name:           "has rule with invalid kind",
			expectedResult: false,
			rules: []smiAccess.TrafficTargetRule{
				{
					Name:    "test",
					Kind:    "Invalid",
					Matches: []string{},
				},
			},
		},
		{
			name:           "has rule with valid HTTPRouteGroup kind",
			expectedResult: true,
			rules: []smiAccess.TrafficTargetRule{
				{
					Name:    "test",
					Kind:    HTTPRouteGroupKind,
					Matches: []string{},
				},
			},
		},
		{
			name:           "has rule with valid TCPRouteGroup kind",
			expectedResult: true,
			rules: []smiAccess.TrafficTargetRule{
				{
					Name:    "test",
					Kind:    TCPRouteKind,
					Matches: []string{},
				},
			},
		},
		{
			name:           "has multiple rules with valid and invalid kind",
			expectedResult: false,
			rules: []smiAccess.TrafficTargetRule{
				{
					Name:    "test1",
					Kind:    TCPRouteKind,
					Matches: []string{},
				},
				{
					Name:    "test2",
					Kind:    HTTPRouteGroupKind,
					Matches: []string{},
				},
				{
					Name:    "test2",
					Kind:    "invalid",
					Matches: []string{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasValidRules(tc.rules)
			assert.Equal(tc.expectedResult, result)
		})
	}
}

func TestIsValidTrafficTarget(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name           string
		expectedResult bool
		trafficTarget  *smiAccess.TrafficTarget
	}{
		{
			name:           "traffic target namespace does not match destination namespace",
			expectedResult: false,
			trafficTarget: &smiAccess.TrafficTarget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "access.smi-spec.io/v1alpha3",
					Kind:       "TrafficTarget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: "traffic-target-namespace",
				},
				Spec: smiAccess.TrafficTargetSpec{
					Destination: smiAccess.IdentityBindingSubject{
						Kind:      "ServiceAccount",
						Name:      "sa-2",
						Namespace: "destination-namespace",
					},
					Sources: []smiAccess.IdentityBindingSubject{{
						Kind:      "ServiceAccount",
						Name:      "sa-1",
						Namespace: "ns-1",
					}},
					Rules: []smiAccess.TrafficTargetRule{
						{
							Kind: "TCPRoute",
							Name: "route-1",
						},
					},
				},
			},
		},
		{
			name:           "traffic target is valid",
			expectedResult: true,
			trafficTarget: &smiAccess.TrafficTarget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "access.smi-spec.io/v1alpha3",
					Kind:       "TrafficTarget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: "namespace",
				},
				Spec: smiAccess.TrafficTargetSpec{
					Destination: smiAccess.IdentityBindingSubject{
						Kind:      "ServiceAccount",
						Name:      "sa-2",
						Namespace: "namespace",
					},
					Sources: []smiAccess.IdentityBindingSubject{{
						Kind:      "ServiceAccount",
						Name:      "sa-1",
						Namespace: "ns-1",
					}},
					Rules: []smiAccess.TrafficTargetRule{
						{
							Kind: "TCPRoute",
							Name: "route-1",
						},
					},
				},
			},
		},
		{
			name:           "traffic target has invalid rules",
			expectedResult: false,
			trafficTarget: &smiAccess.TrafficTarget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "access.smi-spec.io/v1alpha3",
					Kind:       "TrafficTarget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: "traffic-target-namespace",
				},
				Spec: smiAccess.TrafficTargetSpec{
					Destination: smiAccess.IdentityBindingSubject{
						Kind:      "ServiceAccount",
						Name:      "sa-2",
						Namespace: "destination-namespace",
					},
					Sources: []smiAccess.IdentityBindingSubject{{
						Kind:      "ServiceAccount",
						Name:      "sa-1",
						Namespace: "ns-1",
					}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidTrafficTarget(tc.trafficTarget)
			assert.Equal(tc.expectedResult, result)
		})
	}
}
