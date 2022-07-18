package providers

import (
	"context"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
)

// MRCComposer is a composer object that allows consumers
// to observe MRCs (via List() and Watch()) as well as generate
// `certificate.Provider`s from those MRCs
type MRCComposer struct {
	informerCollection *informers.InformerCollection
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
				Type: certificate.MRCEventAdded,
				MRC:  mrc,
			}
		},
		// We don't really care about the previous version
		// since the "state machine" of the MRC is well defined
		UpdateFunc: func(_, newObj interface{}) {
			mrc := newObj.(*v1alpha2.MeshRootCertificate)
			log.Debug().Msgf("received MRC update event for MRC %s/%s", mrc.GetNamespace(), mrc.GetName())
			eventChan <- certificate.MRCEvent{
				Type: certificate.MRCEventUpdated,
				MRC:  mrc,
			}
		},
		// We don't care about deletes because the only deletes that should
		// happen come from the control plane cleaning up an old MRC. Our
		// ValdatingWebhookConfiguration should prevent deletes from users
		DeleteFunc: func(obj interface{}) {},
	})

	return eventChan, nil
}
