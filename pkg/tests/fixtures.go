package tests

import (
	"encoding/pem"
	"errors"
	"fmt"
	"net"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	"github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	backpressure "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
	tresorPem "github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests/certificates"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// ErrDecodingPEMBlock is an error message emitted when a PEM block cannot be decoded
var ErrDecodingPEMBlock = errors.New("failed to decode PEM block containing certificate")

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

	// Weight90 is the value representing a share of the traffic to be sent this way in a traffic split scenario.
	Weight90 = 90

	// Weight10 is the value representing a share of the traffic to be sent this way in a traffic split scenario.
	Weight10 = 10

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

	// ProxyUUID is the unique ID of the Envoy used for unit tests.
	ProxyUUID = "abcdef12-5791-9876-abcd-1234567890ab"

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

	// BookstoreV1Hostnames are the hostnames for bookstore-v1 service
	BookstoreV1Hostnames = []string{
		"bookstore-v1",
		"bookstore-v1.default",
		"bookstore-v1.default.svc",
		"bookstore-v1.default.svc.cluster",
		"bookstore-v1.default.svc.cluster.local",
		"bookstore-v1:8888",
		"bookstore-v1.default:8888",
		"bookstore-v1.default.svc:8888",
		"bookstore-v1.default.svc.cluster:8888",
		"bookstore-v1.default.svc.cluster.local:8888",
	}

	// BookstoreV2Hostnames are the hostnames for the bookstore-v2 service
	BookstoreV2Hostnames = []string{
		"bookstore-v2",
		"bookstore-v2.default",
		"bookstore-v2.default.svc",
		"bookstore-v2.default.svc.cluster",
		"bookstore-v2.default.svc.cluster.local",
		"bookstore-v2:8888",
		"bookstore-v2.default:8888",
		"bookstore-v2.default.svc:8888",
		"bookstore-v2.default.svc.cluster:8888",
		"bookstore-v2.default.svc.cluster.local:8888",
	}

	// BookstoreApexHostnames are the hostnames for the bookstore-apex service
	BookstoreApexHostnames = []string{
		"bookstore-apex",
		"bookstore-apex.default",
		"bookstore-apex.default.svc",
		"bookstore-apex.default.svc.cluster",
		"bookstore-apex.default.svc.cluster.local",
		"bookstore-apex:8888",
		"bookstore-apex.default:8888",
		"bookstore-apex.default.svc:8888",
		"bookstore-apex.default.svc.cluster:8888",
		"bookstore-apex.default.svc.cluster.local:8888",
	}

	// BookstoreBuyHTTPRoute is an HTTP route to buy books
	BookstoreBuyHTTPRoute = trafficpolicy.HTTPRouteMatch{
		PathRegex: BookstoreBuyPath,
		Methods:   []string{"GET"},
		Headers: map[string]string{
			"user-agent": HTTPUserAgent,
		},
	}

	// BookstoreSellHTTPRoute is an HTTP route to sell books
	BookstoreSellHTTPRoute = trafficpolicy.HTTPRouteMatch{
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
		HTTPRouteMatches: []trafficpolicy.HTTPRouteMatch{
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
		HTTPRouteMatches: []trafficpolicy.HTTPRouteMatch{
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
		HTTPRouteMatches: []trafficpolicy.HTTPRouteMatch{
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
					Weight:  Weight90,
				},
				{
					Service: BookstoreV2ServiceName,
					Weight:  Weight10,
				},
			},
		},
	}

	// TrafficTarget is a traffic target SMI object.
	TrafficTarget = access.TrafficTarget{
		TypeMeta: v1.TypeMeta{
			APIVersion: "access.smi-spec.io/v1alpha3",
			Kind:       "TrafficTarget",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      TrafficTargetName,
			Namespace: "default",
		},
		Spec: access.TrafficTargetSpec{
			Destination: access.IdentityBindingSubject{
				Kind:      "Name",
				Name:      BookstoreServiceAccountName,
				Namespace: "default",
			},
			Sources: []access.IdentityBindingSubject{{
				Kind:      "Name",
				Name:      BookbuyerServiceAccountName,
				Namespace: "default",
			}},
			Rules: []access.TrafficTargetRule{{
				Kind:    "HTTPRouteGroup",
				Name:    RouteGroupName,
				Matches: []string{BuyBooksMatchName, SellBooksMatchName},
			}},
		},
	}

	// RoutePolicyMap is a map of a key to a route policy SMI object.
	RoutePolicyMap = map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch{
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
		Weight:      Weight90,
		RootService: BookstoreApexServiceName,
	}

	// BookstoreV2WeightedService is a service with a weight used for traffic split.
	BookstoreV2WeightedService = service.WeightedService{
		Service: service.MeshService{
			Namespace: Namespace,
			Name:      BookstoreV2ServiceName,
		},
		Weight:      Weight10,
		RootService: BookstoreApexServiceName,
	}

	// HTTPRouteGroup is the HTTP route group SMI object.
	HTTPRouteGroup = spec.HTTPRouteGroup{
		TypeMeta: v1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha4",
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
			APIVersion: "specs.smi-spec.io/v1alpha4",
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

	// BookstoreV1DefaultWeightedCluster is a weighted cluster for bookstore-v1
	BookstoreV1DefaultWeightedCluster = service.WeightedCluster{
		ClusterName: "default/bookstore-v1",
		Weight:      100,
	}

	// BookstoreV2DefaultWeightedCluster is a weighted cluster for bookstore-v2
	BookstoreV2DefaultWeightedCluster = service.WeightedCluster{
		ClusterName: "default/bookstore-v2",
		Weight:      100,
	}

	// BookstoreApexDefaultWeightedCluster is a weighted cluster for bookstore-apex
	BookstoreApexDefaultWeightedCluster = service.WeightedCluster{
		ClusterName: "default/bookstore-apex",
		Weight:      100,
	}

	// PodLabels is a map of the default labels on pods
	PodLabels = map[string]string{
		SelectorKey:                      SelectorValue,
		constants.EnvoyUniqueIDLabelName: ProxyUUID,
	}
)

// NewPodFixture creates a new Pod struct for testing.
func NewPodFixture(namespace string, podName string, serviceAccountName string, labels map[string]string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccountName,
		},
	}
}

// NewServiceFixture creates a new Kubernetes service
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

// NewMeshServiceFixture creates a new mesh service
func NewMeshServiceFixture(serviceName, namespace string) service.MeshService {
	return service.MeshService{
		Name:      serviceName,
		Namespace: namespace,
	}
}

// NewSMITrafficTarget creates a new SMI Traffic Target
func NewSMITrafficTarget(sourceName, sourceNamespace, destName, destNamespace string) access.TrafficTarget {
	return access.TrafficTarget{
		TypeMeta: v1.TypeMeta{
			APIVersion: "access.smi-spec.io/v1alpha3",
			Kind:       "TrafficTarget",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      destName,
			Namespace: destNamespace,
		},
		Spec: access.TrafficTargetSpec{
			Destination: access.IdentityBindingSubject{
				Kind:      "ServiceAccount",
				Name:      destName,
				Namespace: destNamespace,
			},
			Sources: []access.IdentityBindingSubject{{
				Kind:      "ServiceAccount",
				Name:      sourceName,
				Namespace: sourceNamespace,
			}},
			Rules: []access.TrafficTargetRule{{
				Kind:    "HTTPRouteGroup",
				Name:    RouteGroupName,
				Matches: []string{BuyBooksMatchName, SellBooksMatchName},
			}},
		},
	}
}

// GetPEMCert returns a TEST certificate used ONLY for testing
func GetPEMCert() (tresorPem.Certificate, error) {
	caBlock, _ := pem.Decode([]byte(certificates.SampleCertificatePEM))
	if caBlock == nil || caBlock.Type != "CERTIFICATE" {
		return nil, ErrDecodingPEMBlock
	}

	return pem.EncodeToMemory(caBlock), nil
}

// GetPEMPrivateKey returns a TEST private key used ONLY for testing
func GetPEMPrivateKey() (tresorPem.PrivateKey, error) {
	caKeyBlock, _ := pem.Decode([]byte(certificates.SamplePrivateKeyPEM))
	if caKeyBlock == nil || caKeyBlock.Type != "PRIVATE KEY" {
		return nil, ErrDecodingPEMBlock
	}

	return pem.EncodeToMemory(caKeyBlock), nil
}
