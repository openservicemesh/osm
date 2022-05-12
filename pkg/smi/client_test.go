package smi

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	testTrafficTargetClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned/fake"
	testTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned/fake"
	testTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/constants"
	httpServerConstants "github.com/openservicemesh/osm/pkg/httpserver/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
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

func bootstrapClient(stop chan struct{}) (*client, *fakeKubeClientSet, error) {
	osmNamespace := "osm-system"
	meshName := "osm"
	kubeClient := testclient.NewSimpleClientset()
	smiTrafficSplitClientSet := testTrafficSplitClient.NewSimpleClientset()
	smiTrafficSpecClientSet := testTrafficSpecClient.NewSimpleClientset()
	smiTrafficTargetClientSet := testTrafficTargetClient.NewSimpleClientset()
	msgBroker := messaging.NewBroker(stop)
	kubernetesClient, err := k8s.NewKubernetesController(kubeClient, nil, meshName, stop, msgBroker)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	meshSpec, err := newSMIClient(
		smiTrafficSplitClientSet,
		smiTrafficSpecClientSet,
		smiTrafficTargetClientSet,
		osmNamespace,
		kubernetesClient,
		kubernetesClientName,
		stop,
		msgBroker,
	)

	return meshSpec, fakeClientSet, err
}

func TestListTrafficSplits(t *testing.T) {
	a := assert.New(t)
	stop := make(chan struct{})
	defer close(stop)

	c, _, err := bootstrapClient(stop)
	a.Nil(err)

	obj := &smiSplit.TrafficSplit{
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
	err = c.caches.TrafficSplit.Add(obj)
	a.Nil(err)

	// Verify
	actual := c.ListTrafficSplits()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])

	// Verify filter for apex service
	filteredApexAvailable := c.ListTrafficSplits(WithTrafficSplitApexService(service.MeshService{Name: tests.BookstoreApexServiceName, Namespace: testNamespaceName}))
	a.Len(filteredApexAvailable, 1)
	a.Equal(obj, filteredApexAvailable[0])
	filteredApexUnavailable := c.ListTrafficSplits(WithTrafficSplitApexService(tests.BookstoreV1Service))
	a.Len(filteredApexUnavailable, 0)

	// Verify filter for backend service
	filteredBackendAvailable := c.ListTrafficSplits(WithTrafficSplitBackendService(service.MeshService{Name: tests.BookstoreV1ServiceName, Namespace: testNamespaceName}))
	a.Len(filteredBackendAvailable, 1)
	a.Equal(obj, filteredBackendAvailable[0])
	filteredBackendNameMismatch := c.ListTrafficSplits(WithTrafficSplitBackendService(service.MeshService{Namespace: testNamespaceName, Name: "invalid"}))
	a.Len(filteredBackendNameMismatch, 0)
	filteredBackendNamespaceMismatch := c.ListTrafficSplits(WithTrafficSplitBackendService(service.MeshService{Namespace: "invalid", Name: tests.BookstoreV1ServiceName}))
	a.Len(filteredBackendNamespaceMismatch, 0)
}

func TestListTrafficTargets(t *testing.T) {
	a := assert.New(t)
	stop := make(chan struct{})
	defer close(stop)

	c, _, err := bootstrapClient(stop)
	a.Nil(err)

	obj := &smiAccess.TrafficTarget{
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
	err = c.caches.TrafficTarget.Add(obj)
	a.Nil(err)

	// Verify
	actual := c.ListTrafficTargets()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])

	// Verify destination based filtering
	filteredAvailable := c.ListTrafficTargets(WithTrafficTargetDestination(identity.K8sServiceAccount{Namespace: testNamespaceName, Name: tests.BookstoreServiceAccountName}))
	a.Len(filteredAvailable, 1)
	filteredUnavailable := c.ListTrafficTargets(WithTrafficTargetDestination(identity.K8sServiceAccount{Namespace: testNamespaceName, Name: "unavailable"}))
	a.Len(filteredUnavailable, 0)
}

func TestListHTTPTrafficSpecs(t *testing.T) {
	a := assert.New(t)
	stop := make(chan struct{})
	defer close(stop)

	c, _, err := bootstrapClient(stop)
	a.Nil(err)

	obj := &smiSpecs.HTTPRouteGroup{
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
	err = c.caches.HTTPRouteGroup.Add(obj)
	a.Nil(err)

	// Verify
	actual := c.ListHTTPTrafficSpecs()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])
}

func TestGetHTTPRouteGroup(t *testing.T) {
	a := assert.New(t)
	stop := make(chan struct{})
	defer close(stop)

	c, _, err := bootstrapClient(stop)
	a.Nil(err)

	obj := &smiSpecs.HTTPRouteGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha4",
			Kind:       "HTTPRouteGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespaceName,
			Name:      "foo",
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
	err = c.caches.HTTPRouteGroup.Add(obj)
	a.Nil(err)

	// Verify
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	actual := c.GetHTTPRouteGroup(key)
	a.Equal(obj, actual)

	invalid := c.GetHTTPRouteGroup("invalid")
	a.Nil(invalid)
}

func TestListTCPTrafficSpecs(t *testing.T) {
	a := assert.New(t)
	stop := make(chan struct{})
	defer close(stop)

	c, _, err := bootstrapClient(stop)
	a.Nil(err)

	obj := &smiSpecs.TCPRoute{
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
	err = c.caches.TCPRoute.Add(obj)
	a.Nil(err)

	// Verify
	actual := c.ListTCPTrafficSpecs()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])
}

func TestGetTCPRoute(t *testing.T) {
	a := assert.New(t)
	stop := make(chan struct{})
	defer close(stop)

	c, _, err := bootstrapClient(stop)
	a.Nil(err)

	obj := &smiSpecs.TCPRoute{
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
	err = c.caches.TCPRoute.Add(obj)
	a.Nil(err)

	// Verify
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	actual := c.GetTCPRoute(key)
	a.Equal(obj, actual)

	invalid := c.GetTCPRoute("invalid")
	a.Nil(invalid)
}

func TestGetSmiClientVersionHTTPHandler(t *testing.T) {
	a := assert.New(t)

	url := "http://localhost"
	testHTTPServerPort := 8888
	smiVerionPath := httpServerConstants.SmiVersionPath
	recordCall := func(ts *httptest.Server, path string) *http.Response {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		ts.Config.Handler.ServeHTTP(w, req)

		return w.Result()
	}
	handlers := map[string]http.Handler{
		smiVerionPath: GetSmiClientVersionHTTPHandler(),
	}

	router := http.NewServeMux()
	for path, handler := range handlers {
		router.Handle(path, handler)
	}

	testServer := &httptest.Server{
		Config: &http.Server{
			Addr:    fmt.Sprintf(":%d", testHTTPServerPort),
			Handler: router,
		},
	}

	// Verify
	resp := recordCall(testServer, fmt.Sprintf("%s%s", url, smiVerionPath))
	a.Equal(http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	a.Nil(err)
	a.Equal(`{"HTTPRouteGroup":"specs.smi-spec.io/v1alpha4","TCPRoute":"specs.smi-spec.io/v1alpha4","TrafficSplit":"split.smi-spec.io/v1alpha2","TrafficTarget":"access.smi-spec.io/v1alpha3"}`, string(bodyBytes))
}

func TestHasValidRules(t *testing.T) {
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
			a := assert.New(t)
			result := hasValidRules(tc.rules)
			a.Equal(tc.expectedResult, result)
		})
	}
}

func TestIsValidTrafficTarget(t *testing.T) {
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
			a := assert.New(t)
			result := isValidTrafficTarget(tc.trafficTarget)
			a.Equal(tc.expectedResult, result)
		})
	}
}
