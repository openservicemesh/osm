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

// MeshRootCertificateStatus defines the status of the MeshRootCertificate resource
type MeshRootCertificateStatus struct {
	// State specifies the state of the certificate provider
	// All states are specified in constants.go
	State string `json:"state"`
}

// MeshRootCertificateList defines the list of MeshRootCertificate objects
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MeshRootCertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MeshRootCertificate `json:"items"`
}
