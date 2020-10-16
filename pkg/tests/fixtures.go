package tests

import (
	"fmt"
	"net"

	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha3"
	"github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	backpressure "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	// Namespace is the commonly used namespace.
	Namespace = "default"

	// PodName is the name of the pod commonly used namespace.
	PodName = "pod-name"

	// BookstoreV1ServiceName is the name of the bookstore-v1 service.
	BookstoreV1ServiceName = "bookstore-v1"

	// BookstoreV2ServiceName is the name of the bookstore-v2 service.
	BookstoreV2ServiceName = "bookstore-v2"

	// BookstoreApexServiceName that have been is the name of the bookstore service, which is then split into other services.
	BookstoreApexServiceName = "bookstore-apex"

	// BookbuyerServiceName is the name of the bookbuyer service
	BookbuyerServiceName = "bookbuyer"

	// BookwarehouseServiceName is the name of the bookwarehouse service
	BookwarehouseServiceName = "bookwarehouse"

	// BookstoreServiceAccountName is the name of the bookstore service account
	BookstoreServiceAccountName = "bookstore"
	// BookbuyerServiceAccountName is the name of the bookbuyer service account
	BookbuyerServiceAccountName = "bookbuyer"

	// TrafficTargetName is the name of the traffic target SMI object.
	TrafficTargetName = "bookbuyer-access-bookstore"

	// BuyBooksMatchName is the name of the match object.
	BuyBooksMatchName = "buy-books"

	// SellBooksMatchName is the name of the match object.
	SellBooksMatchName = "sell-books"

	// WildcardWithHeadersMatchName is the name of the match object.
	WildcardWithHeadersMatchName = "allow-everything-on-header"

	// Weight is the percentage of the traffic to be sent this way in a traffic split scenario.
	Weight = 100

	// RouteGroupName is the name of the route group SMI object.
	RouteGroupName = "bookstore-service-routes"

	// BookstoreBuyPath is the path to the bookstore.
	BookstoreBuyPath = "/buy"

	// BookstoreSellPath is the path to the bookstore.
	BookstoreSellPath = "/sell"

	// SelectorKey is a Pod selector key constant.
	SelectorKey = "app"

	// SelectorValue is a Pod selector value constant.
	SelectorValue = "frontend"

	// EnvoyUID is the unique ID of the Envoy used for unit tests.
	EnvoyUID = "A-B-C-D"

	// ServicePort is the port used by a service
	ServicePort = 8888

	// ServiceIP is the IP used by a service
	ServiceIP = "8.8.8.8"

	// HTTPUserAgent is the User Agent in the HTTP header
	HTTPUserAgent = "test-UA"
)

var (
	// BookstoreV1Service is the bookstore service.
	BookstoreV1Service = service.MeshService{
		Namespace: Namespace,
		Name:      BookstoreV1ServiceName,
	}

	// BookstoreV2Service is the bookstore service.
	BookstoreV2Service = service.MeshService{
		Namespace: Namespace,
		Name:      BookstoreV2ServiceName,
	}

	// BookbuyerService is the bookbuyer service.
	BookbuyerService = service.MeshService{
		Namespace: Namespace,
		Name:      BookbuyerServiceName,
	}

	// BookstoreApexService is the bookstore-apex service
	BookstoreApexService = service.MeshService{
		Namespace: Namespace,
		Name:      BookstoreApexServiceName,
	}

	// BookwarehouseService is the bookwarehouse service.
	BookwarehouseService = service.MeshService{
		Namespace: Namespace,
		Name:      BookwarehouseServiceName,
	}

	// BookstoreBuyHTTPRoute is an HTTP route to buy books
	BookstoreBuyHTTPRoute = trafficpolicy.HTTPRoute{
		PathRegex: BookstoreBuyPath,
		Methods:   []string{"GET"},
		Headers: map[string]string{
			"user-agent": HTTPUserAgent,
		},
	}

	// BookstoreSellHTTPRoute is an HTTP route to sell books
	BookstoreSellHTTPRoute = trafficpolicy.HTTPRoute{
		PathRegex: BookstoreSellPath,
		Methods:   []string{"GET"},
		Headers: map[string]string{
			"user-agent": HTTPUserAgent,
		},
	}

	// Endpoint is an endpoint object.
	Endpoint = endpoint.Endpoint{
		IP:   net.ParseIP(ServiceIP),
		Port: endpoint.Port(ServicePort),
	}

	// BookstoreV1TrafficPolicy is a traffic policy SMI object.
	BookstoreV1TrafficPolicy = trafficpolicy.TrafficTarget{
		Name:        fmt.Sprintf("%s:default/bookbuyer->default/bookstore-v1", TrafficTargetName),
		Destination: BookstoreV1Service,
		Source:      BookbuyerService,
		HTTPRoutes: []trafficpolicy.HTTPRoute{
			{
				PathRegex: BookstoreBuyPath,
				Methods:   []string{"GET"},
				Headers: map[string]string{
					"user-agent": HTTPUserAgent,
				},
			},
			{
				PathRegex: BookstoreSellPath,
				Methods:   []string{"GET"},
				Headers: map[string]string{
					"user-agent": HTTPUserAgent,
				},
			},
		},
	}

	// BookstoreV2TrafficPolicy is a traffic policy SMI object.
	BookstoreV2TrafficPolicy = trafficpolicy.TrafficTarget{
		Name:        fmt.Sprintf("%s:default/bookbuyer->default/bookstore-v2", TrafficTargetName),
		Destination: BookstoreV2Service,
		Source:      BookbuyerService,
		HTTPRoutes: []trafficpolicy.HTTPRoute{
			{
				PathRegex: BookstoreBuyPath,
				Methods:   []string{"GET"},
				Headers: map[string]string{
					"user-agent": HTTPUserAgent,
				},
			},
			{
				PathRegex: BookstoreSellPath,
				Methods:   []string{"GET"},
				Headers: map[string]string{
					"user-agent": HTTPUserAgent,
				},
			},
		},
	}

	// BookstoreApexTrafficPolicy is a traffic policy SMI object.
	BookstoreApexTrafficPolicy = trafficpolicy.TrafficTarget{
		Name:        fmt.Sprintf("%s:default/bookbuyer->default/bookstore-apex", TrafficTargetName),
		Destination: BookstoreApexService,
		Source:      BookbuyerService,
		HTTPRoutes: []trafficpolicy.HTTPRoute{
			{
				PathRegex: BookstoreBuyPath,
				Methods:   []string{"GET"},
				Headers: map[string]string{
					"user-agent": HTTPUserAgent,
				},
			},
			{
				PathRegex: BookstoreSellPath,
				Methods:   []string{"GET"},
				Headers: map[string]string{
					"user-agent": HTTPUserAgent,
				},
			},
		},
	}

	// TrafficSplit is a traffic split SMI object.
	TrafficSplit = v1alpha2.TrafficSplit{
		ObjectMeta: v1.ObjectMeta{
			Namespace: Namespace,
		},
		Spec: v1alpha2.TrafficSplitSpec{
			Service: BookstoreApexServiceName,
			Backends: []v1alpha2.TrafficSplitBackend{
				{
					Service: BookstoreV1ServiceName,
					Weight:  Weight,
				},
				{
					Service: BookstoreV2ServiceName,
					Weight:  Weight,
				},
			},
		},
	}

	// TrafficTarget is a traffic target SMI object.
	TrafficTarget = target.TrafficTarget{
		TypeMeta: v1.TypeMeta{
			APIVersion: "access.smi-spec.io/v1alpha2",
			Kind:       "TrafficTarget",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      TrafficTargetName,
			Namespace: "default",
		},
		Spec: target.TrafficTargetSpec{
			Destination: target.IdentityBindingSubject{
				Kind:      "Name",
				Name:      BookstoreServiceAccountName,
				Namespace: "default",
			},
			Sources: []target.IdentityBindingSubject{{
				Kind:      "Name",
				Name:      BookbuyerServiceAccountName,
				Namespace: "default",
			}},
			Rules: []target.TrafficTargetRule{{
				Kind:    "HTTPRouteGroup",
				Name:    RouteGroupName,
				Matches: []string{BuyBooksMatchName, SellBooksMatchName},
			}},
		},
	}

	// RoutePolicyMap is a map of a key to a route policy SMI object.
	RoutePolicyMap = map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRoute{
		trafficpolicy.TrafficSpecName(fmt.Sprintf("HTTPRouteGroup/%s/%s", Namespace, RouteGroupName)): {
			trafficpolicy.TrafficSpecMatchName(BuyBooksMatchName):  BookstoreBuyHTTPRoute,
			trafficpolicy.TrafficSpecMatchName(SellBooksMatchName): BookstoreSellHTTPRoute,
		},
	}

	// BookstoreServiceAccount is a namespaced service account.
	BookstoreServiceAccount = service.K8sServiceAccount{
		Namespace: Namespace,
		Name:      BookstoreServiceAccountName,
	}

	// BookbuyerServiceAccount is a namespaced bookbuyer account.
	BookbuyerServiceAccount = service.K8sServiceAccount{
		Namespace: Namespace,
		Name:      BookbuyerServiceAccountName,
	}

	// BookstoreV1WeightedService is a service with a weight used for traffic split.
	BookstoreV1WeightedService = service.WeightedService{
		Service: service.MeshService{
			Namespace: Namespace,
			Name:      BookstoreV1ServiceName,
		},
		Weight:      Weight,
		RootService: BookstoreApexServiceName,
	}

	// BookstoreV2WeightedService is a service with a weight used for traffic split.
	BookstoreV2WeightedService = service.WeightedService{
		Service: service.MeshService{
			Namespace: Namespace,
			Name:      BookstoreV2ServiceName,
		},
		Weight:      Weight,
		RootService: BookstoreApexServiceName,
	}

	// HTTPRouteGroup is the HTTP route group SMI object.
	HTTPRouteGroup = spec.HTTPRouteGroup{
		TypeMeta: v1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha2",
			Kind:       "HTTPRouteGroup",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: "default",
			Name:      RouteGroupName,
		},

		Spec: spec.HTTPRouteGroupSpec{
			Matches: []spec.HTTPMatch{
				{
					Name:      BuyBooksMatchName,
					PathRegex: BookstoreBuyPath,
					Methods:   []string{"GET"},
					Headers: map[string]string{
						"user-agent": HTTPUserAgent,
					},
				},
				{
					Name:      SellBooksMatchName,
					PathRegex: BookstoreSellPath,
					Methods:   []string{"GET"},
					Headers: map[string]string{
						"user-agent": HTTPUserAgent,
					},
				},
				{
					Name: WildcardWithHeadersMatchName,
					Headers: map[string]string{
						"user-agent": HTTPUserAgent,
					},
				},
			},
		},
	}

	// TCPRoute is a TCPRoute SMI resource
	TCPRoute = spec.TCPRoute{
		TypeMeta: v1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha2",
			Kind:       "TCPRoute",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: "default",
			Name:      "tcp-route",
		},
		Spec: spec.TCPRouteSpec{},
	}

	// Backpressure is an experimental Backpressure policy.
	// This will be replaced by an SMI Spec when it is ready.
	Backpressure = backpressure.Backpressure{
		Spec: backpressure.BackpressureSpec{
			MaxConnections: 123,
		},
	}
)

// NewPodTestFixture creates a new Pod struct for testing.
func NewPodTestFixture(namespace string, podName string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				SelectorKey:                      SelectorValue,
				constants.EnvoyUniqueIDLabelName: EnvoyUID,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: BookstoreServiceAccountName,
		},
	}
}

// NewPodTestFixtureWithOptions creates a new Pod struct with options for testing.
func NewPodTestFixtureWithOptions(namespace string, podName string, serviceAccountName string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				SelectorKey:                      SelectorValue,
				constants.EnvoyUniqueIDLabelName: EnvoyUID,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccountName,
		},
	}
}

// NewServiceFixture creates a new MeshService
func NewServiceFixture(serviceName, namespace string, selectors map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name: "servicePort",
				TargetPort: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "backendName",
				},
				Protocol: corev1.ProtocolTCP,
				Port:     ServicePort,
			}},
			Selector: selectors,
		},
	}
}
