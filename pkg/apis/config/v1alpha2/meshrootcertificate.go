package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MeshRootCertificate defines the configuration for certificate issuing
// by the mesh control plane
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshRootCertificate struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the MeshRootCertificate config specification
	// +optional
	Spec MeshRootCertificateSpec `json:"spec,omitempty"`

	// Status of the MeshRootCertificate resource
	// +optional
	Status MeshRootCertificateStatus `json:"status,omitempty"`
}

// MeshRootCertificateSpec defines the mesh root certificate specification
type MeshRootCertificateSpec struct {
	// Provider specifies the mesh certificate provider
	Provider ProviderSpec `json:"provider"`

	// TrustDomain is the trust domain to use as a suffix in Common Names for new certificates.
	TrustDomain string `json:"trustDomain"`

	// Intent of the MeshRootCertificate resource
	Intent MeshRootCertificateIntent `json:"intent"`

	// SpiffeEnabled will add a SPIFFE ID to the certificates, creating a SPIFFE compatible x509 SVID document
	// To use SPIFFE ID for validation and routing, 'enableSPIFFE' must be true in the MeshConfig after the MeshRootCertificate is made 'active'
	SpiffeEnabled bool `json:"spiffeEnabled"`
}

// ProviderSpec defines the certificate provider used by the mesh control plane
type ProviderSpec struct {
	// CertManager specifies the cert-manager provider configuration
	// +optional
	CertManager *CertManagerProviderSpec `json:"certManager,omitempty"`

	// Vault specifies the vault provider configuration
	// +optional
	Vault *VaultProviderSpec `json:"vault,omitempty"`

	// Tresor specifies the Tresor provider configuration
	// +optional
	Tresor *TresorProviderSpec `json:"tresor,omitempty"`
}

// CertManagerProviderSpec defines the configuration of the cert-manager provider
type CertManagerProviderSpec struct {
	// IssuerName specifies the name of the Issuer resource
	IssuerName string `json:"issuerName"`

	// IssuerKind specifies the kind of Issuer
	IssuerKind string `json:"issuerKind"`

	// IssuerGroup specifies the group the Issuer belongs to
	IssuerGroup string `json:"issuerGroup"`
}

// VaultProviderSpec defines the configuration of the Vault provider
type VaultProviderSpec struct {
	// Host specifies the name of the Vault server
	Host string `json:"host"`

	// Port specifies the port of the Vault server
	Port int `json:"port"`

	// Role specifies the name of the role for use by mesh control plane
	Role string `json:"role"`

	// Protocol specifies the protocol for connections to Vault
	Protocol string `json:"protocol"`

	// Token specifies the configuration of the token to be used by mesh control plane
	// to connect to Vault
	Token VaultTokenSpec `json:"token"`
}

// VaultTokenSpec defines the configuration of the Vault token
type VaultTokenSpec struct {
	// SecretKeyRef specifies the secret in which the Vault token is stored
	SecretKeyRef SecretKeyReferenceSpec `json:"secretKeyRef"`
}

// SecretKeyReferenceSpec defines the configuration of the secret reference
type SecretKeyReferenceSpec struct {
	// Name specifies the name of the secret in which the Vault token is stored
	Name string `json:"name"`

	// Key specifies the key whose value is the Vault token
	Key string `json:"key"`

	// Namespace specifies the namespace of the secret in which the Vault token is stored
	Namespace string `json:"namespace"`
}

// TresorProviderSpec defines the configuration of the Tresor provider
type TresorProviderSpec struct {
	// CA specifies Tresor's ca configuration
	CA TresorCASpec `json:"ca"`
}

// TresorCASpec defines the configuration of Tresor's root certificate
type TresorCASpec struct {
	// SecretRef specifies the secret in which the root certificate is stored
	SecretRef corev1.SecretReference `json:"secretRef"`
}

// MeshRootCertificateIntent specifies the intent of the MeshRootCertificate
// can be (Active, Passive).
type MeshRootCertificateIntent string

const (
	// ActiveIntent means the settings and certificate provider in this MRC are used for signing and
	// validating certificates.
	ActiveIntent MeshRootCertificateIntent = "active"

	// PassiveIntent means the settings and certificate provider in this MRC are used for validating
	// certificates.
	PassiveIntent MeshRootCertificateIntent = "passive"

	// InactiveIntent means the settings and certificate provider in this MRC no longer in use.
	InactiveIntent MeshRootCertificateIntent = "inactive"
)

// MeshRootCertificateComponentStatus specifies the status of the certificate component,
// can be (`Issuing`, `Validating`, `Unknown`).
type MeshRootCertificateComponentStatus string

const (
	// Issuing means that the root cert described by this MRC is now issuing certs for this component of OSM.
	Issuing MeshRootCertificateComponentStatus = "issuing"

	// Validating means that the root cert's cert chain, described by this MRC is now part of the CABundle used to
	// validate requests for this component..
	Validating MeshRootCertificateComponentStatus = "validating"

	// Unused means that the root cert described by this MRC is unused.
	Unused MeshRootCertificateComponentStatus = "unused"

	// UnknownComponentStatus means that the use of the root cert described by this MRC is in an unknown state for this
	// component.
	UnknownComponentStatus MeshRootCertificateComponentStatus = "unknown"
)

// MeshRootCertificateConditionStatus specifies the status of the MeshRootCertificate condition,
// one of (`True`, `False`, `Unknown`).
type MeshRootCertificateConditionStatus string

// MeshRootCertificateConditionType specifies the type of the condition,
// one of (`Ready`, `Accepted`, `IssuingRollout`, `ValidatingRollout`, `IssuingRollback`, `ValidatingRollback`).
type MeshRootCertificateConditionType string

// MeshRootCertificateComponentStatuses is the set of statuses for each certificate component in the cluster.
type MeshRootCertificateComponentStatuses struct {
	ValidatingWebhook MeshRootCertificateComponentStatus `json:"validatingWebhook"`
	MutatingWebhook   MeshRootCertificateComponentStatus `json:"mutatingWebhook"`
	XDSControlPlane   MeshRootCertificateComponentStatus `json:"xdsControlPlane"`
	Sidecar           MeshRootCertificateComponentStatus `json:"sidecar"`
	Bootstrap         MeshRootCertificateComponentStatus `json:"bootstrap"`
	Gateway           MeshRootCertificateComponentStatus `json:"gateway"`
}

// MeshRootCertificateCondition defines the condition of the MeshRootCertificate resource.
type MeshRootCertificateCondition struct {
	// Type of the condition,
	// one of (`Ready`, `Accepted`, `IssuingRollout`, `ValidatingRollout`, `IssuingRollback`, `ValidatingRollback`).
	Type MeshRootCertificateConditionType `json:"type"`

	// Status of the condition, one of (`True`, `False`, `Unknown`).
	Status MeshRootCertificateConditionStatus `json:"status"`

	// LastTransitionTime is the timestamp corresponding to the last status
	// change of this condition.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine readable explanation for the condition's last
	// transition (should be in camelCase).
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human readable description of the details of the last
	// transition, complementing reason.
	// +optional
	Message string `json:"message,omitempty"`
}

// MeshRootCertificateStatus defines the status of the MeshRootCertificate resource.
type MeshRootCertificateStatus struct {
	// State specifies the state of the certificate provider.
	// All states are specified in constants.go
	State string `json:"state"`

	// If present, this MRC can transition to the next state in the state machine after this timestamp.
	TransitionAfter *metav1.Time `json:"transitionAfter,omitempty"`

	// Set of statuses for each certificate component in the cluster (e.g. webhooks, bootstrap, etc.)
	// NOTE: There is a caveat that since these components belong to horizontally scalable pods, it is possible that not
	// all of these components will be ready. That is, one controller might mark the ADS server as ready, while all other
	// controllers have yet to rotate their controller cert.
	ComponentStatuses MeshRootCertificateComponentStatuses `json:"componentStatuses"`

	// List of status conditions to indicate the status of a MeshRootCertificate.
	// Known condition types are `Ready` and `InvalidRequest`.
	// +optional
	Conditions []MeshRootCertificateCondition `json:"conditions"`
}

// MeshRootCertificateList defines the list of MeshRootCertificate objects
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshRootCertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MeshRootCertificate `json:"items"`
}
