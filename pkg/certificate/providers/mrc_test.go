package providers

import (
	"context"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/compute/kube"
	fakeConfigClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// TODO: implement test for UpdateFunc when UpdateMeshRootCertificate is implemented
func TestWatch(t *testing.T) {
	assert := tassert.New(t)

	stop := make(chan struct{})
	configClient := fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "osm-mesh-root-certificate",
			Namespace: "osm-system",
		},
		Spec: v1alpha2.MeshRootCertificateSpec{
			Provider: v1alpha2.ProviderSpec{
				CertManager: &v1alpha2.CertManagerProviderSpec{
					IssuerName:  "test-name",
					IssuerKind:  "ClusterIssuer",
					IssuerGroup: "cert-manager.io",
				},
			},
		},
	})

	client, err := k8s.NewClient("osm-system", "osm-mesh-config", messaging.NewBroker(stop),
		k8s.WithKubeClient(fake.NewSimpleClientset(), "osm"),
		k8s.WithConfigClient(configClient),
	)
	assert.Nil(err)
	computeClient := kube.NewClient(client)

	mrcComposer := MRCComposer{
		MRCProviderGenerator: MRCProviderGenerator{
			Interface: computeClient,
		},
	}
	mrcEvent, err := mrcComposer.Watch(context.Background())
	assert.NotNil(mrcEvent)
	assert.Nil(err)
}
