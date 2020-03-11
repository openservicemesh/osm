package utils

// CertificateCommonNameMeta is the type that stores the metadata present in the CommonName field in a proxy's certificate
type CertificateCommonNameMeta struct {
	UUID               string
	ServiceAccountName string
	Namespace          string
	SubDomain          string
}
