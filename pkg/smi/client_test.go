package smi

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	testTrafficTargetClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned/fake"
	testTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned/fake"
	testTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	osmPolicy "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
	osmPolicyClient "github.com/openservicemesh/osm/experimental/pkg/client/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/featureflags"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
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
	osmPolicyClientSet        *osmPolicyClient.Clientset
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
	osmPolicyClientSet := osmPolicyClient.NewSimpleClientset()
	kubernetesClient, err := k8s.NewKubernetesController(kubeClient, meshName, stop)
	if err != nil {
		GinkgoT().Fatalf("Error initializing kubernetes controller: %s", err.Error())
	}

	fakeClientSet := &fakeKubeClientSet{
		kubeClient:                kubeClient,
		smiTrafficSplitClientSet:  smiTrafficSplitClientSet,
		smiTrafficSpecClientSet:   smiTrafficSpecClientSet,
		smiTrafficTargetClientSet: smiTrafficTargetClientSet,
		osmPolicyClientSet:        osmPolicyClientSet,
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
		osmPolicyClientSet,
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
		tsChannel := events.GetPubSubInstance().Subscribe(announcements.TrafficSplitAdded,
			announcements.TrafficSplitDeleted,
			announcements.TrafficSplitUpdated)
		defer events.GetPubSubInstance().Unsub(tsChannel)

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

		err = fakeClientSet.smiTrafficSplitClientSet.SplitV1alpha2().TrafficSplits(testNamespaceName).Delete(context.TODO(), split.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-tsChannel
	})
})

var _ = Describe("When listing TrafficSplit services", func() {
	var (
		meshSpec      MeshSpec
		fakeClientSet *fakeKubeClientSet
		err           error
	)
	BeforeEach(func() {
		meshSpec, fakeClientSet, err = bootstrapClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return a list of weighted services corresponding to the traffic split backends", func() {
		tsChannel := events.GetPubSubInstance().Subscribe(announcements.TrafficSplitAdded,
			announcements.TrafficSplitDeleted,
			announcements.TrafficSplitUpdated)
		defer events.GetPubSubInstance().Unsub(tsChannel)

		split := &smiSplit.TrafficSplit{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ListTrafficSplitServices",
				Namespace: testNamespaceName,
			},
			Spec: smiSplit.TrafficSplitSpec{
				Service: tests.BookstoreApexServiceName,
				Backends: []smiSplit.TrafficSplitBackend{
					{
						Service: "bookstore-v1",
						Weight:  tests.Weight90,
					},
					{
						Service: "bookstore-v2",
						Weight:  tests.Weight10,
					},
				},
			},
		}

		_, err := fakeClientSet.smiTrafficSplitClientSet.SplitV1alpha2().TrafficSplits(testNamespaceName).Create(context.TODO(), split, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-tsChannel

		weightedServices := meshSpec.ListTrafficSplitServices()
		Expect(len(weightedServices)).To(Equal(len(split.Spec.Backends)))
		for i, backend := range split.Spec.Backends {
			Expect(weightedServices[i].Service).To(Equal(service.MeshService{Namespace: split.Namespace, Name: backend.Service}))
			Expect(weightedServices[i].Weight).To(Equal(backend.Weight))
			Expect(weightedServices[i].RootService).To(Equal(split.Spec.Service))
		}

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
		ttChannel := events.GetPubSubInstance().Subscribe(announcements.TrafficTargetAdded,
			announcements.TrafficTargetDeleted,
			announcements.TrafficTargetUpdated)
		defer events.GetPubSubInstance().Unsub(ttChannel)

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
					Kind:      "Name",
					Name:      tests.BookstoreServiceAccountName,
					Namespace: testNamespaceName,
				},
				Sources: []smiAccess.IdentityBindingSubject{{
					Kind:      "Name",
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
		ttChannel := events.GetPubSubInstance().Subscribe(announcements.TrafficTargetAdded,
			announcements.TrafficTargetDeleted,
			announcements.TrafficTargetUpdated)
		defer events.GetPubSubInstance().Unsub(ttChannel)

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
					Kind:      "Name",
					Name:      tests.BookstoreServiceAccountName,
					Namespace: testNamespaceName,
				},
				Sources: []smiAccess.IdentityBindingSubject{{
					Kind:      "Name",
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
		rgChannel := events.GetPubSubInstance().Subscribe(announcements.RouteGroupAdded,
			announcements.RouteGroupDeleted,
			announcements.RouteGroupUpdated)
		defer events.GetPubSubInstance().Unsub(rgChannel)

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
		trChannel := events.GetPubSubInstance().Subscribe(announcements.TCPRouteAdded,
			announcements.TCPRouteDeleted,
			announcements.TCPRouteUpdated)
		defer events.GetPubSubInstance().Unsub(trChannel)
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

var _ = Describe("When fetching BackpressurePolicy for the given MeshService", func() {
	var (
		meshSpec      MeshSpec
		fakeClientSet *fakeKubeClientSet
		err           error
	)

	It("should returns nil when a Backpressure feature is disabled", func() {
		meshSvc := service.MeshService{
			Namespace: testNamespaceName,
			Name:      "test-GetBackpressurePolicy",
		}
		backpressure := meshSpec.GetBackpressurePolicy(meshSvc)
		Expect(backpressure).To(BeNil())
	})

	// Initialize feature for unit testing
	optional := featureflags.OptionalFeatures{
		Backpressure: true,
	}
	featureflags.Initialize(optional)

	BeforeEach(func() {
		meshSpec, fakeClientSet, err = bootstrapClient()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should returns nil when a Backpressure policy does not exist for the given service", func() {
		meshSvc := service.MeshService{
			Namespace: testNamespaceName,
			Name:      "test-GetBackpressurePolicy",
		}
		backpressure := meshSpec.GetBackpressurePolicy(meshSvc)
		Expect(backpressure).To(BeNil())
	})

	It("should return the Backpressure policy for the given service", func() {
		bpChannel := events.GetPubSubInstance().Subscribe(announcements.BackpressureAdded,
			announcements.BackpressureDeleted,
			announcements.BackpressureUpdated)
		defer events.GetPubSubInstance().Unsub(bpChannel)

		meshSvc := service.MeshService{
			Namespace: testNamespaceName,
			Name:      "test-GetBackpressurePolicy",
		}
		backpressurePolicy := &osmPolicy.Backpressure{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "policy.openservicemesh.io/v1alpha1",
				Kind:       "Backpressure",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespaceName,
				Name:      "test-GetBackpressurePolicy",
				Labels:    map[string]string{"app": meshSvc.Name},
			},
			Spec: osmPolicy.BackpressureSpec{
				MaxConnections: 123,
			},
		}

		_, err := fakeClientSet.osmPolicyClientSet.PolicyV1alpha1().Backpressures(testNamespaceName).Create(context.TODO(), backpressurePolicy, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-bpChannel

		backpressurePolicyInCache := meshSpec.GetBackpressurePolicy(meshSvc)
		Expect(backpressurePolicyInCache).ToNot(BeNil())
		Expect(backpressurePolicyInCache.Name).To(Equal(backpressurePolicy.Name))

		err = fakeClientSet.osmPolicyClientSet.PolicyV1alpha1().Backpressures(testNamespaceName).Delete(context.TODO(), backpressurePolicy.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-bpChannel
	})

	It("should return nil when the app label is missing for the given service", func() {
		bpChannel := events.GetPubSubInstance().Subscribe(announcements.BackpressureAdded,
			announcements.BackpressureDeleted,
			announcements.BackpressureUpdated)
		defer events.GetPubSubInstance().Unsub(bpChannel)

		meshSvc := service.MeshService{
			Namespace: testNamespaceName,
			Name:      "test-GetBackpressurePolicy",
		}
		backpressurePolicy := &osmPolicy.Backpressure{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "policy.openservicemesh.io/v1alpha1",
				Kind:       "Backpressure",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespaceName,
				Name:      "test-GetBackpressurePolicy",
			},
			Spec: osmPolicy.BackpressureSpec{
				MaxConnections: 123,
			},
		}

		_, err := fakeClientSet.osmPolicyClientSet.PolicyV1alpha1().Backpressures(testNamespaceName).Create(context.TODO(), backpressurePolicy, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-bpChannel

		backpressurePolicyInCache := meshSpec.GetBackpressurePolicy(meshSvc)
		Expect(backpressurePolicyInCache).To(BeNil())

		err = fakeClientSet.osmPolicyClientSet.PolicyV1alpha1().Backpressures(testNamespaceName).Delete(context.TODO(), backpressurePolicy.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-bpChannel
	})
})
