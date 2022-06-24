package providers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
)

// MRCComposer is a composer object that allows consumers
// to observe MRCs (via List() and Watch()), generate
// `certificate.Provider`s from those MRCs, and update the
// MRCs
type MRCComposer struct {
	informerCollection *informers.InformerCollection
	configClient       configClientset.Interface
	MRCProviderGenerator
}

// List returns the MRCs stored in the informerCollection's store
func (m *MRCComposer) List() ([]*v1alpha2.MeshRootCertificate, error) {
	// informers return slice of pointers so we'll convert them to value types before returning
	mrcPtrs := m.informerCollection.List(informers.InformerKeyMeshRootCertificate)
	var mrcs []*v1alpha2.MeshRootCertificate
	for _, mrcPtr := range mrcPtrs {
		if mrcPtr == nil {
			continue
		}
		mrc, ok := mrcPtr.(*v1alpha2.MeshRootCertificate)
		if !ok {
			continue
		}
		mrcs = append(mrcs, mrc)
	}

	return mrcs, nil
}

// Watch returns a channel that receives events whenever MRCs are added, updated, and deleted
// from the informerCollection's MRC store. Channels returned from multiple invocations of
// Watch() are unique and have no coordination with each other. Events are guaranteed
// to be ordered for any particular resources, but NOT across different resources.
func (m *MRCComposer) Watch(ctx context.Context) (<-chan certificate.MRCEvent, error) {
	eventChan := make(chan certificate.MRCEvent)
	m.informerCollection.AddEventHandler(informers.InformerKeyMeshRootCertificate, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			mrc := obj.(*v1alpha2.MeshRootCertificate)
			log.Debug().Msgf("received MRC add event for MRC %s/%s", mrc.GetNamespace(), mrc.GetName())
			eventChan <- certificate.MRCEvent{
				Type:   certificate.MRCEventAdded,
				NewMRC: mrc,
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			newMRC := newObj.(*v1alpha2.MeshRootCertificate)
			oldMRC := oldObj.(*v1alpha2.MeshRootCertificate)
			log.Debug().Msgf("received MRC update event for MRC %s/%s", newMRC.GetNamespace(), newMRC.GetName())
			eventChan <- certificate.MRCEvent{
				Type:   certificate.MRCEventUpdated,
				OldMRC: oldMRC,
				NewMRC: newMRC,
			}
		},
		// We don't care about deletes because the only deletes that should
		// happen come from the control plane cleaning up an old MRC. Our
		// ValdatingWebhookConfiguration should prevent deletes from users
		DeleteFunc: func(obj interface{}) {},
	})

	return eventChan, nil
}

// Update returns an updated MRC. If no MRC is provided, the MRC is obtained using the ID and
// namespace. The status of the MRC is set to the provided status and the updated MRC is
// returned.
func (m *MRCComposer) Update(id, ns, status string, mrc *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	var err error
	updatedMRC := mrc
	if updatedMRC == nil {
		updatedMRC, err = m.configClient.ConfigV1alpha2().MeshRootCertificates(ns).Get(context.Background(), id, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	updatedMRC.Status.State = status

	return m.configClient.ConfigV1alpha2().MeshRootCertificates(updatedMRC.Namespace).UpdateStatus(context.Background(), updatedMRC, metav1.UpdateOptions{})
}
