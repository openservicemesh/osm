package providers

import (
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
)

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *MRCCompatClient) List() ([]*v1alpha2.MeshRootCertificate, error) {
	return []*v1alpha2.MeshRootCertificate{
		c.mrc,
	}, nil
}
