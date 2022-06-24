package providers

/*
import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
)

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *MRCCompatClient) List() ([]*v1alpha2.MeshRootCertificate, error) {
	return []*v1alpha2.MeshRootCertificate{
		c.mrc,
	}, nil
}

// Watch is a basic Watch implementation for the MRC attached to the compat client
func (c *MRCCompatClient) Watch(ctx context.Context) (<-chan certificate.MRCEvent, error) {
	ch := make(chan certificate.MRCEvent)
	go func() {
		ch <- certificate.MRCEvent{
			Type: certificate.MRCEventAdded,
			MRC:  c.mrc,
		}
		close(ch)
	}()

	return ch, nil
}

func (c *MRCCompatClient) Update(mrc *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	return c.configClient.ConfigV1alpha2().MeshRootCertificates(mrc.Namespace).UpdateStatus(context.Background(), mrc, metav1.UpdateOptions{})
}

func (m *MRCCompatClient) Get(id string, ns string) (*v1alpha2.MeshRootCertificate, error) {
	return m.configClient.ConfigV1alpha2().MeshRootCertificates(ns).Get(context.Background(), id, metav1.GetOptions{})
}*/
