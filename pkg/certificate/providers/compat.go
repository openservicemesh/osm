package providers

import (
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
)

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *MRCCompatClient) List() ([]*v1alpha2.MeshRootCertificate, error) {
	return []*v1alpha2.MeshRootCertificate{
		c.mrc,
	}, nil
}

// AddEventHandler is a no-op for the legacy client. The previous client could not handle changes, but we need this
// method to implement the certificate.MRCClient interface.
func (c *MRCCompatClient) AddEventHandler(cache.ResourceEventHandler) {}

// provider.Tresor = &v1alpha2.TresorProviderSpec{SecretName: opts.SecretName}
