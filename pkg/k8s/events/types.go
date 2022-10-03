// Package events implements the eventing framework to receive and relay kubernetes events, and a framework to
// publish events to the Kubernetes API server.
package events

import (
	"fmt"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("kube-events")
)

// Kubernetes Fatal Event reasons
// Fatal events are prefixed with 'Fatal' to help the event recording framework to wait for fatal
// events to be recorded prior to aborting.
const (
	// InvalidCLIParameters signifies invalid CLI parameters
	InvalidCLIParameters = "FatalInvalidCLIParameters"

	// InitializationError signifies an error during initialization
	InitializationError = "FatalInitializationError"

	// InvalidCertificateManager signifies that the certificate manager is invalid
	InvalidCertificateManager = "FatalInvalidCertificateManager"

	// CertificateIssuanceFailure signifies that a request to issue a certificate failed
	CertificateIssuanceFailure = "FatalCertificateIssuanceFailure"
)

// Kubernetes Event reasons
const (
	// CertificateRotationFailure signifies that a certificate failed to rotate
	CertificateRotationFailure = "CertificateRotationFailure"
)

// PubSubMessage represents a common messages abstraction to pass through the PubSub interface
type PubSubMessage struct {
	Kind   Kind
	Type   EventType
	OldObj interface{}
	NewObj interface{}
}

// Topic returns the PubSub Topic for the given message.
func (m *PubSubMessage) Topic() string {
	return fmt.Sprintf("%s-%s", m.Kind, m.Type)
}

// EventType represents the type of event we have received from Kubernetes: added, updated, or deleted
type EventType string

const (
	// Added is an EventType representing a new object being added.
	Added EventType = "added"

	// Updated is an EventType representing an object being updated.
	Updated EventType = "updated"

	// Deleted is an EventType represent an object being deleted.
	Deleted EventType = "deleted"
)

// Kind is used to record the kind of announcement
type Kind string

// String returns the string representation of the Kind
func (at Kind) String() string {
	return string(at)
}

// Added returns the string representation of the Kind combined with the added keyword to be used for a PubSub Topic.
func (at Kind) Added() string {
	return fmt.Sprintf("%s-%s", at, Added)
}

// Updated returns the string representation of the Kind combined with the updated keyword to be used for a PubSub Topic.
func (at Kind) Updated() string {
	return fmt.Sprintf("%s-%s", at, Updated)
}

// Deleted returns the string representation of the Kind combined with the deleted keyword to be used for a PubSub Topic.
func (at Kind) Deleted() string {
	return fmt.Sprintf("%s-%s", at, Deleted)
}

const (
	// ProxyUpdate is a special osm event kind that does not correspond to a k8s event, but instead is used as an
	// indicator for osm to trigger a broadcast to update all proxies.
	ProxyUpdate Kind = "proxy"

	// Pod is a Kind for Kubernetes pod events.
	Pod Kind = "pod"

	// Endpoint is the Kind for Kubernetes events.
	Endpoint Kind = "endpoint"

	// Namespace is the Kind for Kubernetes namespace events.
	Namespace Kind = "namespace"

	// Service is the Kind for Kubernetes service events.
	Service Kind = "service"

	// ServiceAccount is the Kind for Kubernetes service account events.
	ServiceAccount Kind = "serviceaccount"

	// TrafficSplit is the Kind for Kubernetes traffic split events.
	TrafficSplit Kind = "trafficsplit"

	// RouteGroup is the Kind for Kubernetes route group events.
	RouteGroup Kind = "routegroup"

	// TCPRoute is the Kind for Kubernetes tcp route events.
	TCPRoute Kind = "tcproute"

	// TrafficTarget is the Kind for Kubernetes traffic target events.
	TrafficTarget Kind = "traffictarget"

	// Ingress is the Kind for Kubernetes ingress events.
	Ingress Kind = "ingress"

	// MeshConfig is the Kind for Kubernetes meshconfig events.
	MeshConfig Kind = "meshconfig"

	// MeshRootCertificate is the Kind for Kubernetes mrc events.
	MeshRootCertificate Kind = "meshrootcertificate"

	// Egress is the Kind for Kubernetes egress events.
	Egress Kind = "egress"

	// IngressBackend is the Kind for Kubernetes ingress backend events.
	IngressBackend Kind = "ingressbackend"

	// RetryPolicy is the Kind for Kubernetes retry policy events.
	RetryPolicy Kind = "retry"

	// UpstreamTrafficSetting is the Kind for Kubernetes UpstreamTrafficSetting events.
	UpstreamTrafficSetting Kind = "upstreamtrafficsetting"

	// Telemetry is the Kind for Kubernetes Telemetry events.
	Telemetry Kind = "telemetry"

	// ExtensionService is the Kind for Kubernetes ExtensionService events.
	ExtensionService Kind = "extensionservice"
)

// GetKind returns the Kind for the given k8s object.
func GetKind(obj interface{}) Kind {
	switch obj.(type) {
	case *corev1.Pod:
		return Pod
	case *corev1.Endpoints:
		return Endpoint
	case *corev1.Namespace:
		return Namespace
	case *corev1.Service:
		return Service
	case *corev1.ServiceAccount:
		return ServiceAccount
	case *networkingv1.Ingress:
		return Ingress
	case *smiSplit.TrafficSplit:
		return TrafficSplit
	case *smiSpecs.HTTPRouteGroup:
		return RouteGroup
	case *smiSpecs.TCPRoute:
		return TCPRoute
	case *smiAccess.TrafficTarget:
		return TrafficTarget
	case *configv1alpha2.MeshConfig:
		return MeshConfig
	case *configv1alpha2.MeshRootCertificate:
		return MeshRootCertificate
	case *policyv1alpha1.Egress:
		return Egress
	case *policyv1alpha1.IngressBackend:
		return IngressBackend
	case *policyv1alpha1.Retry:
		return RetryPolicy
	case *policyv1alpha1.UpstreamTrafficSetting:
		return UpstreamTrafficSetting
	case *policyv1alpha1.Telemetry:
		return Telemetry
	case *configv1alpha2.ExtensionService:
		return ExtensionService
	default:
		log.Error().Msgf("Unknown kind: %v", obj)
		return ""
	}
}
