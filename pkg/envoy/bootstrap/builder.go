package bootstrap

import "github.com/openservicemesh/osm/pkg/models"

func (b *Builder) SetXDSHost(xdsHost string) {
	b.XDSHost = xdsHost
}

func (b *Builder) SetNodeID(nodeID string) {
	b.NodeID = nodeID
}

func (b *Builder) SetTLSMinProtocolVersion(tlsMinProtocolVersion string) {
	b.TLSMinProtocolVersion = tlsMinProtocolVersion
}

func (b *Builder) SetTLSMaxProtocolVersion(tlsMaxProtocolVersion string) {
	b.TLSMaxProtocolVersion = tlsMaxProtocolVersion
}

func (b *Builder) SetCipherSuites(cipherSuites []string) {
	b.CipherSuites = cipherSuites
}

func (b *Builder) SetECDHCurves(ecdhCurves []string) {
	b.ECDHCurves = ecdhCurves
}

func (b *Builder) SetOriginalHealthProbes(originalHealthProbes models.HealthProbes) {
	b.OriginalHealthProbes = originalHealthProbes
}

func NewBuilder() *Builder {
	return &Builder{}
}