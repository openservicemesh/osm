// Package tests implements utility routines used for unit testing.
package tests

import (
	"encoding/pem"
	"errors"
	"net"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	"github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	tresorPem "github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests/certificates"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// ErrDecodingPEMBlock is an error message emitted when a PEM block cannot be decoded
var ErrDecodingPEMBlock = errors.New("failed to decode PEM block containing certificate")

const (
	// MeshName is the name of the OSM mesh
	MeshName = "osm"

	// OsmNamespace is the namespace of OSM control plane
	OsmNamespace = "osm-system"

	// OsmMeshConfigName is the name of OSM MeshConfig resource
	OsmMeshConfigName = "osm-mesh-config"

	// Namespace is the commonly used namespace.
	Namespace = "default"

	// BookstoreServiceName is the name of the bookstore service.
	BookstoreServiceName = "bookstore"

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
	// BookstoreV2ServiceAccountName is the name of the bookstore-v2 service account
	BookstoreV2ServiceAccountName = "bookstore-v2"

	// TrafficTargetName is the name of the traffic target SMI object.
	TrafficTargetName = "bookbuyer-access-bookstore"

	// BookstoreV2TrafficTargetName is the name of the traffic target SMI object.
	BookstoreV2TrafficTargetName = "bookbuyer-access-bookstore-v2"

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

	//HTTPHostHeader is the host name in the HTTP header
	HTTPHostHeader = "osm.mesh"
)

var (
	// BookstoreV1Service is the bookstore service.
	BookstoreV1Service = service.MeshService{
		Namespace:  Namespace,
		Name:       BookstoreV1ServiceName,
		Port:       ServicePort,
		TargetPort: ServicePort,
		Protocol:   constants.ProtocolHTTP,
	}

	// BookstoreV2Service is the bookstore service.
	BookstoreV2Service = service.MeshService{
		Namespace:  Namespace,
		Name:       BookstoreV2ServiceName,
		Port:       ServicePort,
		TargetPort: ServicePort,
		Protocol:   constants.ProtocolHTTP,
	}

	// BookbuyerService is the bookbuyer service.
	BookbuyerService = service.MeshService{
		Namespace:  Namespace,
		Name:       BookbuyerServiceName,
		Port:       ServicePort,
		TargetPort: ServicePort,
		Protocol:   constants.ProtocolHTTP,
	}

	// BookstoreApexService is the bookstore-apex service
	BookstoreApexService = service.MeshService{
		Namespace:  Namespace,
		Name:       BookstoreApexServiceName,
		Port:       ServicePort,
		TargetPort: ServicePort,
		Protocol:   constants.ProtocolHTTP,
	}

	// BookwarehouseService is the bookwarehouse service.
	BookwarehouseService = service.MeshService{
		Namespace: Namespace,
		Name:      BookwarehouseServiceName,
		Port:      ServicePort,
	}

	// BookstoreHostnames are the hostnames for bookstore service
	BookstoreHostnames = []string{
		"bookstore",
		"bookstore.default",
		"bookstore.default.svc",
		"bookstore.default.svc.cluster",
		"bookstore.default.svc.cluster.local",
		"bookstore:8888",
		"bookstore.default:8888",
		"bookstore.default.svc:8888",
		"bookstore.default.svc.cluster:8888",
		"bookstore.default.svc.cluster.local:8888",
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

	// BookstoreV1NamespacedHostnames are the hostnames for the bookstore-apex service
	BookstoreV1NamespacedHostnames = []string{
		"bookstore-v1.default",
		"bookstore-v1.default.svc",
		"bookstore-v1.default.svc.cluster",
		"bookstore-v1.default.svc.cluster.local",
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

	// BookstoreApexNamespacedHostnames are the namespaced hostnames for the bookstore-apex service
	BookstoreApexNamespacedHostnames = []string{
		"bookstore-apex.default",
		"bookstore-apex.default.svc",
		"bookstore-apex.default.svc.cluster",
		"bookstore-apex.default.svc.cluster.local",
		"bookstore-apex.default:8888",
		"bookstore-apex.default.svc:8888",
		"bookstore-apex.default.svc.cluster:8888",
		"bookstore-apex.default.svc.cluster.local:8888",
	}

	// BookbuyerHostnames are the hostnames for the bookbuyer service
	BookbuyerHostnames = []string{
		"bookbuyer",
		"bookbuyer.default",
		"bookbuyer.default.svc",
		"bookbuyer.default.svc.cluster",
		"bookbuyer.default.svc.cluster.local",
		"bookbuyer:8888",
		"bookbuyer.default:8888",
		"bookbuyer.default.svc:8888",
		"bookbuyer.default.svc.cluster:8888",
		"bookbuyer.default.svc.cluster.local:8888",
	}

	// BookbuyerTestHostnames are the namespaced hostnames for the bookbuyer service
	BookbuyerTestHostnames = []string{
		"bookbuyer",
		"bookbuyer.test",
		"bookbuyer.test.svc",
		"bookbuyer.test.svc.cluster",
		"bookbuyer.test.svc.cluster.local",
		"bookbuyer:8888",
		"bookbuyer.test:8888",
		"bookbuyer.test.svc:8888",
		"bookbuyer.test.svc.cluster:8888",
		"bookbuyer.test.svc.cluster.local:8888",
	}

	// ApexSplitBazNamespacedHostNames are the namespaced hostnames for the apex split service
	ApexSplitBazNamespacedHostNames = []string{
		"apex-split-1.baz",
		"apex-split-1.baz.svc",
		"apex-split-1.baz.svc.cluster",
		"apex-split-1.baz.svc.cluster.local",
		"apex-split-1.baz:8888",
		"apex-split-1.baz.svc:8888",
		"apex-split-1.baz.svc.cluster:8888",
		"apex-split-1.baz.svc.cluster.local:8888",
	}

	// ApexSplitBarNamespacedHostNames are the namespaced hostnames for the apex split service
	ApexSplitBarNamespacedHostNames = []string{
		"apex-split-1.bar",
		"apex-split-1.bar.svc",
		"apex-split-1.bar.svc.cluster",
		"apex-split-1.bar.svc.cluster.local",
		"apex-split-1.bar:8888",
		"apex-split-1.bar.svc:8888",
		"apex-split-1.bar.svc.cluster:8888",
		"apex-split-1.bar.svc.cluster.local:8888",
	}

	// BookstoreBuyHTTPRoute is an HTTP route to buy books
	BookstoreBuyHTTPRoute = trafficpolicy.HTTPRouteMatch{
		Path:          BookstoreBuyPath,
		PathMatchType: trafficpolicy.PathMatchRegex,
		Methods:       []string{"GET"},
		Headers: map[string]string{
			"user-agent": HTTPUserAgent,
		},
	}

	// BookstoreBuyHTTPRouteWithHost is an HTTP route to buy books
	BookstoreBuyHTTPRouteWithHost = trafficpolicy.HTTPRouteMatch{
		Path:          BookstoreBuyPath,
		PathMatchType: trafficpolicy.PathMatchRegex,
		Methods:       []string{"GET"},
		Headers: map[string]string{
			"user-agent": HTTPUserAgent,
			"host":       HTTPHostHeader,
		},
	}

	// BookstoreSellHTTPRoute is an HTTP route to sell books
	BookstoreSellHTTPRoute = trafficpolicy.HTTPRouteMatch{
		Path:          BookstoreSellPath,
		PathMatchType: trafficpolicy.PathMatchRegex,
		Methods:       []string{"GET"},
		Headers: map[string]string{
			"user-agent": HTTPUserAgent,
		},
	}

	// Endpoint is an endpoint object.
	Endpoint = endpoint.Endpoint{
		IP:   net.ParseIP(ServiceIP),
		Port: endpoint.Port(ServicePort),
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
				Kind:      "ServiceAccount",
				Name:      BookstoreServiceAccountName,
				Namespace: "default",
			},
			Sources: []access.IdentityBindingSubject{{
				Kind:      "ServiceAccount",
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

	// BookstoreV2TrafficTarget is a traffic target SMI object for bookstore-v2.
	BookstoreV2TrafficTarget = access.TrafficTarget{
		TypeMeta: v1.TypeMeta{
			APIVersion: "access.smi-spec.io/v1alpha3",
			Kind:       "TrafficTarget",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      BookstoreV2TrafficTargetName,
			Namespace: "default",
		},
		Spec: access.TrafficTargetSpec{
			Destination: access.IdentityBindingSubject{
				Kind:      "ServiceAccount",
				Name:      BookstoreV2ServiceAccountName,
				Namespace: "default",
			},
			Sources: []access.IdentityBindingSubject{{
				Kind:      "ServiceAccount",
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

	// BookstoreServiceAccount is a namespaced service account.
	BookstoreServiceAccount = identity.K8sServiceAccount{
		Namespace: Namespace,
		Name:      BookstoreServiceAccountName,
	}

	// BookstoreServiceIdentity is the ServiceIdentity for the Bookstore service.
	BookstoreServiceIdentity = BookstoreServiceAccount.ToServiceIdentity()

	// BookstoreV2ServiceAccount is a namespaced service account.
	BookstoreV2ServiceAccount = identity.K8sServiceAccount{
		Namespace: Namespace,
		Name:      BookstoreV2ServiceAccountName,
	}

	// BookstoreV2ServiceIdentity is the ServiceIdentity for the Bokstore v2 service.
	BookstoreV2ServiceIdentity = BookstoreV2ServiceAccount.ToServiceIdentity()

	// BookbuyerServiceAccount is a namespaced bookbuyer account.
	BookbuyerServiceAccount = identity.K8sServiceAccount{
		Namespace: Namespace,
		Name:      BookbuyerServiceAccountName,
	}

	// BookbuyerServiceIdentity is the ServiceIdentity for the Bookbuyer service.
	BookbuyerServiceIdentity = BookbuyerServiceAccount.ToServiceIdentity()

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

	// HTTPRouteGroupWithHost is the HTTP route group SMI object having a host header.
	HTTPRouteGroupWithHost = spec.HTTPRouteGroup{
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
						"host":       HTTPHostHeader,
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

	// BookbuyerDefaultWeightedCluster is a weighted cluster for bookbuyer
	BookbuyerDefaultWeightedCluster = service.WeightedCluster{
		ClusterName: "default/bookbuyer",
		Weight:      100,
	}

	// PodLabels is a map of the default labels on pods
	PodLabels = map[string]string{
		constants.AppLabel:               SelectorValue,
		constants.EnvoyUniqueIDLabelName: ProxyUUID,
	}

	// WildCardRouteMatch is HTTPRouteMatch with wildcard path and method
	WildCardRouteMatch = trafficpolicy.HTTPRouteMatch{
		Path:          constants.RegexMatchAll,
		PathMatchType: trafficpolicy.PathMatchRegex,
		Methods:       []string{constants.WildcardHTTPMethod},
	}

	// ExpectedHostnames is a map of host names for target service
	ExpectedHostnames = map[string][]string{
		BookstoreServiceName:          BookstoreHostnames,
		BookstoreV1ServiceName:        BookstoreV1Hostnames,
		"bookstore-v1-namespaced":     BookstoreV1NamespacedHostnames,
		BookstoreV2ServiceName:        BookstoreV2Hostnames,
		BookstoreApexServiceName:      BookstoreApexHostnames,
		"bookstore-apex-namespaced":   BookstoreApexNamespacedHostnames,
		BookbuyerServiceName:          BookbuyerHostnames,
		"bookbuyer-test":              BookbuyerTestHostnames,
		"apex-split-1-namespaced":     ApexSplitBarNamespacedHostNames,
		"apex-split-1-baz-namespaced": ApexSplitBazNamespacedHostNames,
	}
)

// NewPodFixture creates a new Pod struct for testing.
func NewPodFixture(namespace string, podName string, serviceAccountName string, labels map[string]string) *corev1.Pod {
	return NewOsSpecificPodFixture(namespace, podName, serviceAccountName, labels, constants.OSLinux)
}

// NewOsSpecificPodFixture creates a new Pod struct for testing.
func NewOsSpecificPodFixture(namespace string, podName string, serviceAccountName string, labels map[string]string, podOS string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccountName,
			NodeSelector: map[string]string{
				"kubernetes.io/os": podOS,
			},
		},
	}
}

// HeadlessSvc converts a service into a headless service
func HeadlessSvc(svc *corev1.Service) *corev1.Service {
	svc.Spec.ClusterIP = corev1.ClusterIPNone

	return svc
}

// NewServiceFixture creates a new Kubernetes service
func NewServiceFixture(serviceName, namespace string, selectors map[string]string) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.1",
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

	return svc
}

// NewServiceAccountFixture creates a new Kubernetes service account
func NewServiceAccountFixture(svcAccountName, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      svcAccountName,
			Namespace: namespace,
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
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func NewSMITrafficTarget(source identity.ServiceIdentity, destination identity.ServiceIdentity) access.TrafficTarget {
	sourceSA := source.ToK8sServiceAccount()
	destinationSA := destination.ToK8sServiceAccount()
	sourceName := sourceSA.Name
	sourceNamespace := sourceSA.Namespace
	destName := destinationSA.Name
	destNamespace := destinationSA.Namespace

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
