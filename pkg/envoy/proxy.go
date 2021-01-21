package envoy

import (
	"fmt"
	"net"
	"time"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
)

// Proxy is a representation of an Envoy proxy connected to the xDS server.
// This should at some point have a 1:1 match to an Endpoint (which is a member of a meshed service).
type Proxy struct {
	// The Subject Common Name of the certificate used for Envoy to XDS communication.
	xDSCertificateCommonName certificate.CommonName

	// The Serial Number of the certificate used for Envoy to XDS communication.
	xDSCertificateSerialNumber certificate.SerialNumber

	net.Addr
	announcements chan announcements.Announcement

	// The time this Proxy connected to the OSM control plane
	connectedAt time.Time

	lastSentVersion    map[TypeURI]uint64
	lastAppliedVersion map[TypeURI]uint64
	lastNonce          map[TypeURI]string

	// Records metadata around the Kubernetes Pod on which this Envoy Proxy is installed.
	// This could be nil if the Envoy is not operating in a Kubernetes cluster (VM for example)
	// NOTE: This field may be not be set at the time Proxy struct is initialized. This would
	// eventually be set when the metadata arrives via the xDS protocol.
	PodMetadata *PodMetadata
}

// PodMetadata is a struct holding information on the Pod on which a given Envoy proxy is installed
// This struct is initialized *eventually*, when the metadata arrives via xDS.
type PodMetadata struct {
	UID            string
	Namespace      string
	IP             string
	ServiceAccount string
	Cluster        string
	EnvoyNodeID    string
}

// HasPodMetadata answers the question - has the Pod metadata been recorded for the given Envoy proxy
func (p *Proxy) HasPodMetadata() bool {
	return p.PodMetadata != nil
}

// SetLastAppliedVersion records the version of the given Envoy proxy that was last acknowledged.
func (p *Proxy) SetLastAppliedVersion(typeURI TypeURI, version uint64) {
	p.lastAppliedVersion[typeURI] = version
}

// GetLastAppliedVersion returns the last version successfully applied to the given Envoy proxy.
func (p Proxy) GetLastAppliedVersion(typeURI TypeURI) uint64 {
	return p.lastAppliedVersion[typeURI]
}

// GetLastSentVersion returns the last sent version.
func (p Proxy) GetLastSentVersion(typeURI TypeURI) uint64 {
	return p.lastSentVersion[typeURI]
}

// IncrementLastSentVersion increments last sent version.
func (p *Proxy) IncrementLastSentVersion(typeURI TypeURI) uint64 {
	p.lastSentVersion[typeURI]++
	return p.GetLastSentVersion(typeURI)
}

// SetLastSentVersion records the version of the given config last sent to the proxy.
func (p *Proxy) SetLastSentVersion(typeURI TypeURI, ver uint64) {
	p.lastSentVersion[typeURI] = ver
}

// GetLastSentNonce returns last sent nonce.
func (p *Proxy) GetLastSentNonce(typeURI TypeURI) string {
	nonce, ok := p.lastNonce[typeURI]
	if !ok {
		p.lastNonce[typeURI] = ""
		return ""
	}
	return nonce
}

// SetNewNonce sets and returns a new nonce.
func (p *Proxy) SetNewNonce(typeURI TypeURI) string {
	p.lastNonce[typeURI] = fmt.Sprintf("%d", time.Now().UnixNano())
	return p.lastNonce[typeURI]
}

// GetPodUID returns the UID of the pod, which the connected Envoy proxy is fronting.
func (p Proxy) GetPodUID() string {
	if p.PodMetadata == nil {
		return ""
	}
	return p.PodMetadata.UID
}

// GetCertificateCommonName returns the Subject Common Name from the mTLS certificate of the Envoy proxy connected to xDS.
func (p Proxy) GetCertificateCommonName() certificate.CommonName {
	return p.xDSCertificateCommonName
}

// GetCertificateSerialNumber returns the Serial Number of the certificate for the connected Envoy proxy.
func (p Proxy) GetCertificateSerialNumber() certificate.SerialNumber {
	return p.xDSCertificateSerialNumber
}

// GetConnectedAt returns the timestamp of when the given proxy connected to the control plane.
func (p Proxy) GetConnectedAt() time.Time {
	return p.connectedAt
}

// GetIP returns the IP address of the Envoy proxy connected to xDS.
func (p Proxy) GetIP() net.Addr {
	return p.Addr
}

// GetAnnouncementsChannel returns the announcement channel for the given Envoy proxy.
func (p Proxy) GetAnnouncementsChannel() chan announcements.Announcement {
	return p.announcements
}

// NewProxy creates a new instance of an Envoy proxy connected to the xDS servers.
func NewProxy(certCommonName certificate.CommonName, certSerialNumber certificate.SerialNumber, ip net.Addr) *Proxy {
	return &Proxy{
		xDSCertificateCommonName:   certCommonName,
		xDSCertificateSerialNumber: certSerialNumber,

		Addr: ip,

		connectedAt: time.Now(),

		announcements:      make(chan announcements.Announcement),
		lastNonce:          make(map[TypeURI]string),
		lastSentVersion:    make(map[TypeURI]uint64),
		lastAppliedVersion: make(map[TypeURI]uint64),
	}
}
