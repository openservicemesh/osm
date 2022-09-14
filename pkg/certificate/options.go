package certificate

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/openservicemesh/osm/pkg/identity"
)

// IssueOption is an option that can be passed to IssueCertificate on the CertificateManager
type IssueOption func(*IssueOptions)

// IssueOptions is passed to the Certificate Providers when creating certificates
type IssueOptions struct {
	fullCNProvided   bool
	trustDomain      string
	commonNamePrefix string
	certType         certType
	ValidityDuration time.Duration
	spiffeEnabled    bool
}

func (o IssueOptions) cacheKey() string {
	return o.commonNamePrefix
}

// CommonName constructs the CommonName for the certificate.
// If the FullCommonName option is set it will use configured name.
// Otherwise it uses the name configured and appends the trustdomain
func (o IssueOptions) CommonName() CommonName {
	if o.fullCNProvided {
		return CommonName(o.commonNamePrefix)
	}

	if o.commonNamePrefix == "" {
		return CommonName(o.trustDomain)
	}

	return CommonName(fmt.Sprintf("%s.%s", o.commonNamePrefix, o.trustDomain))
}

// URISAN generates a URL in the Spiffe format spiffe://trustdomain/sa/svc
func (o IssueOptions) URISAN() *url.URL {
	if !o.spiffeEnabled {
		return &url.URL{}
	}

	// if the trust domain is already appended remove it.
	path := strings.TrimSuffix(o.commonNamePrefix, "."+o.trustDomain)
	path = strings.ReplaceAll(path, ".", "/")

	return &url.URL{
		Scheme: "spiffe",
		Path:   path,
		Host:   o.trustDomain,
	}
}

func withCommonNamePrefix(prefix string) IssueOption {
	return func(opts *IssueOptions) {
		opts.commonNamePrefix = prefix
	}
}

func withFullCommonName() IssueOption {
	return func(opts *IssueOptions) {
		opts.fullCNProvided = true
	}
}

func withCertType(certType certType) IssueOption {
	return func(opts *IssueOptions) {
		opts.certType = certType
	}
}

func withSpiffeEnabled() IssueOption {
	return func(opts *IssueOptions) {
		opts.spiffeEnabled = true
	}
}

// ForServiceIdentity creates a service certificate with the given prefix for the common name
// The trust domain will be appended to the Common Name
func ForServiceIdentity(identity identity.ServiceIdentity) IssueOption {
	return func(opts *IssueOptions) {
		opts.commonNamePrefix = identity.String()
		opts.certType = service
	}
}

// ForIngressGateway creates a certificate which is given a full common name
func ForIngressGateway(fullCommonName string) IssueOption {
	return func(opts *IssueOptions) {
		opts.commonNamePrefix = fullCommonName
		opts.fullCNProvided = true
		opts.certType = ingressGateway
	}
}

// ForCommonNamePrefix creates an internal certificate with a prefix for the common name.
// The trust domain will be appended to the Common Name
func ForCommonNamePrefix(prefix string) IssueOption {
	return func(opts *IssueOptions) {
		opts.commonNamePrefix = prefix
		opts.certType = internal
	}
}

// ForCommonName creates an internal certificate with a given full common name
func ForCommonName(fullCommonName string) IssueOption {
	return func(opts *IssueOptions) {
		opts.commonNamePrefix = fullCommonName
		opts.certType = internal
		opts.fullCNProvided = true
	}
}

// NewCertOptions creates the IssueOptions for issuing a certificate
func NewCertOptions(options ...IssueOption) IssueOptions {
	opts := &IssueOptions{}
	for _, o := range options {
		o(opts)
	}

	return *opts
}

// NewCertOptionsWithFullName creates the IssueOptions for the issuing a certificate with a given full common name
func NewCertOptionsWithFullName(fullCommonName string, validity time.Duration) IssueOptions {
	opts := &IssueOptions{
		ValidityDuration: validity,
		fullCNProvided:   true,
		commonNamePrefix: fullCommonName,
	}
	return *opts
}

// NewCertOptionsWithTrustDomain creates the IssueOptions for the issuing a certificate with a given full common name
func NewCertOptionsWithTrustDomain(prefix string, trustDomain string, validity time.Duration, spiffeEnabled bool) IssueOptions {
	opts := &IssueOptions{
		ValidityDuration: validity,
		trustDomain:      trustDomain,
		commonNamePrefix: prefix,
		spiffeEnabled:    spiffeEnabled,
	}
	return *opts
}
