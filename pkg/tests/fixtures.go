package tests

import (
	"fmt"
	"net"

	set "github.com/deckarep/golang-set"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/trafficpolicy"
)

const (
	// Namespace is the commonly used namespace.
	Namespace = "default"

	// PodName is the name of the pod commonly used namespace.
	PodName = "pod-name"

	// BookstoreServiceName is the name of the bookstore service.
	BookstoreServiceName = "bookstore"
	// BookbuyerServiceName is the name of the bookbuyer service
	BookbuyerServiceName = "bookbuyer"

	// BookstoreServiceAccountName is the name of the bookstore service account
	BookstoreServiceAccountName = "bookstore-serviceaccount"
	// BookbuyerServiceAccountName is the name of the bookbuyer service account
	BookbuyerServiceAccountName = "bookbuyer-serviceaccount"

	// TrafficTargetName is the name of the traffic target SMI object.
	TrafficTargetName = "bookbuyer-access-bookstore"

	// MatchName is the name of the match object.
	MatchName = "buy-books"

	// Domain is a domain
	Domain = "contoso.com"

	// Weight is the percentage of the traffic to be sent this way in a traffic split scenario.
	Weight = 100

	// RouteGroupName is the name of the route group SMI object.
	RouteGroupName = "bookstore-service-routes"

	// BookstoreBuyPath is the path to the bookstore.
	BookstoreBuyPath = "/buy"

	// SelectorKey is a Pod selector key constant.
	SelectorKey = "app"

	// SelectorValue is a Pod selector value constant.
	SelectorValue = "frontend"

	// EnvoyUID is the unique ID of the Envoy used for unit tests.
	EnvoyUID = "A-B-C-D"
)

var (
	// BookstoreService is the bookstore service.
	BookstoreService = service.NamespacedService{
		Namespace: Namespace,
		Service:   BookstoreServiceName,
	}

	// BookbuyerService is the bookbuyer service.
	BookbuyerService = service.NamespacedService{
		Namespace: Namespace,
		Service:   BookbuyerServiceName,
	}

	// RoutePolicy is a route policy.
	RoutePolicy = trafficpolicy.Route{
		PathRegex: BookstoreBuyPath,
		Methods:   []string{"GET"},
	}

	// Endpoint is an endpoint object.
	Endpoint = endpoint.Endpoint{
		IP:   net.ParseIP("8.8.8.8"),
		Port: endpoint.Port(8888),
	}

	// TrafficPolicy is a traffic policy SMI object.
	TrafficPolicy = trafficpolicy.TrafficTarget{
		Name: TrafficTargetName,
		Destination: trafficpolicy.TrafficResource{
			ServiceAccount: BookstoreServiceAccountName,
			Namespace:      Namespace,
			Services:       set.NewSet(BookstoreService),
		},
		Source: trafficpolicy.TrafficResource{
			ServiceAccount: BookbuyerServiceAccountName,
			Namespace:      Namespace,
			Services:       set.NewSet(BookbuyerService),
		},
		Routes: []trafficpolicy.Route{{
			PathRegex: BookstoreBuyPath,
			Methods:   []string{"GET"},
		}},
	}

	// TrafficTarget is a traffic target SMI object.
	TrafficTarget = target.TrafficTarget{
		TypeMeta: v1.TypeMeta{
			APIVersion: "access.smi-spec.io/v1alpha1",
			Kind:       "TrafficTarget",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      TrafficTargetName,
			Namespace: "default",
		},
		Destination: target.IdentityBindingSubject{
			Kind:      "BookstoreServiceAccount",
			Name:      BookstoreServiceAccountName,
			Namespace: "default",
		},
		Sources: []target.IdentityBindingSubject{{
			Kind:      "BookstoreServiceAccount",
			Name:      BookbuyerServiceAccountName,
			Namespace: "default",
		}},
		Specs: []target.TrafficTargetSpec{{
			Kind:    "HTTPRouteGroup",
			Name:    RouteGroupName,
			Matches: []string{"buy-books"},
		}},
	}

	// RoutePolicyMap is a map of a key to a route policy SMI object.
	RoutePolicyMap = map[string]trafficpolicy.Route{
		fmt.Sprintf("HTTPRouteGroup/%s/%s/%s", Namespace, TrafficTargetName, MatchName): RoutePolicy}

	// NamespacedServiceName is a namespaced service.
	NamespacedServiceName = service.Name(fmt.Sprintf("%s/%s", BookstoreService.Namespace, BookstoreService.Service))

	// BookstoreServiceAccount is a namespaced service account.
	BookstoreServiceAccount = service.NamespacedServiceAccount{
		Namespace:      Namespace,
		ServiceAccount: BookstoreServiceAccountName,
	}

	// BookbuyerServiceAccount is a namespaced bookbuyer account.
	BookbuyerServiceAccount = service.NamespacedServiceAccount{
		Namespace:      Namespace,
		ServiceAccount: BookbuyerServiceAccountName,
	}

	// WeightedService is a service with a weight used for traffic split.
	WeightedService = service.WeightedService{
		ServiceName: service.NamespacedService{
			Namespace: Namespace,
			Service:   BookstoreServiceName,
		},
		Weight: Weight,
		Domain: Domain,
	}

	// HTTPRouteGroup is the HTTP route group SMI object.
	HTTPRouteGroup = spec.HTTPRouteGroup{
		TypeMeta: v1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha1",
			Kind:       "HTTPRouteGroup",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: "default",
			Name:      RouteGroupName,
		},
		Matches: []spec.HTTPMatch{{
			Name:      MatchName,
			PathRegex: BookstoreBuyPath,
			Methods:   []string{"GET"},
		}},
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
			ServiceAccountName: BookbuyerServiceAccountName,
		},
	}
}

// NewServiceFixture creates a new Service
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
				Port:     int32(8080),
			}},
			Selector: selectors,
		},
	}
}
