package providers

import (
	"context"
	"fmt"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
)

// ListMeshRootCertificates returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *MRCCompatClient) ListMeshRootCertificates() ([]*v1alpha2.MeshRootCertificate, error) {
	return []*v1alpha2.MeshRootCertificate{
		c.mrc,
	}, nil
}

// Watch is a basic Watch implementation for the MRC attached to the compat client
func (c *MRCCompatClient) Watch(ctx context.Context) (<-chan certificate.MRCEvent, error) {
	ch := make(chan certificate.MRCEvent)
	go func() {
		ch <- certificate.MRCEvent{
			MRCName: c.mrc.Name,
		}
		close(ch)
	}()

	return ch, nil
}

// UpdateMeshRootCertificate is not implemented on the compat client and always returns an error
func (c *MRCCompatClient) UpdateMeshRootCertificate(mrc *v1alpha2.MeshRootCertificate) error {
	return fmt.Errorf("cannot call UpdateMeshRootCertificate for %s mrc on the compat client", mrc.Name)
}
