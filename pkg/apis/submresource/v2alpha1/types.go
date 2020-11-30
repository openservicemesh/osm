package v2alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Refer: https://github.com/kubernetes/enhancements/pull/1646

//+genclient
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:noStatus

// ServiceImport describes a service imported from clusters in a clusterset.
type ServiceImport struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec ServiceImportSpec `json:"spec,omitempty"`
	// +optional
	Status ServiceImportStatus `json:"status,omitempty"`
}

// ServiceImportType designates the type of a ServiceImport
type ServiceImportType string

const (
	// Services are only accessible via their clusterset IP.
	ClusterSetIP ServiceImportType = "ClusterSetIP"
	// Headless services allow backend pods to be addressed directly.
	Headless ServiceImportType = "Headless"
)

// ServiceImportSpec describes an imported service and the information necessary to consume it.
type ServiceImportSpec struct {
	// +listType=atomic
	Ports []ServicePort `json:"ports"`
	// +optional
	IP string `json:"ip,omitempty"`
	// +optional
	Type ServiceImportType `json:"type"`
	// +optional
	SessionAffinity corev1.ServiceAffinity `json:"sessionAffinity"`
	// +optional
	SessionAffinityConfig *corev1.SessionAffinityConfig `json:"sessionAffinityConfig"`
}

// ServicePort represents the port on which the service is exposed
type ServicePort struct {
	// The name of this port within the service. This must be a DNS_LABEL.
	// All ports within a ServiceSpec must have unique names. When considering
	// the endpoints for a Service, this must match the 'name' field in the
	// EndpointPort.
	// Optional if only one ServicePort is defined on this service.
	// +optional
	Name string `json:"name,omitempty"`

	// The IP protocol for this port. Supports "TCP", "UDP", and "SCTP".
	// Default is TCP.
	// +optional
	Protocol corev1.Protocol `json:"protocol,omitempty"`

	// The application protocol for this port.
	// This field follows standard Kubernetes label syntax.
	// Un-prefixed names are reserved for IANA standard service names (as per
	// RFC-6335 and http://www.iana.org/assignments/service-names).
	// Non-standard protocols should use prefixed names such as
	// mycompany.com/my-custom-protocol.
	// Field can be enabled with ServiceAppProtocol feature gate.
	// +optional
	AppProtocol *string `json:"appProtocol,omitempty"`

	// The port that will be exposed by this service.
	Port int32 `json:"port"`
}

// ServiceImportStatus describes derived state of an imported service.
type ServiceImportStatus struct {
	// +optional
	// +patchStrategy=merge
	// +patchMergeKey=cluster
	// +listType=map
	// +listMapKey=cluster
	Clusters []ClusterStatus `json:"clusters"`
}

// ClusterStatus contains service configuration mapped to a specific source cluster
type ClusterStatus struct {
	Cluster string `json:"cluster"`

	// The IP(s) of the service running in the cluster.  In the case of a headless service, it is the list of pod IPs
	// that back the service.
	// +optional
	IPs []string `json:"ips,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceImportList is a list of serviceimport objects.
type ServiceImportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ServiceImport `json:"items"`
}
