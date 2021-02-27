package catalog

import (
	v1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/dispatcher"
)

// releaseCertificateHandler releases certificates based on podDelete events
// returns a stop channel which can be used to stop the inner handler
func (mc *MeshCatalog) releaseCertificateHandler() chan struct{} {
	podDeleteSubscription := dispatcher.GetPubSubInstance().Subscribe(dispatcher.PodDeleted)
	stop := make(chan struct{})

	go func() {
		for {
			select {
			case <-stop:
				return
			case podDeletedMsg := <-podDeleteSubscription:
				psubMessage, castOk := podDeletedMsg.(dispatcher.PubSubMessage)
				if !castOk {
					log.Error().Msgf("Error casting PubSubMessage: %v", psubMessage)
					continue
				}

				// guaranteed can only be a PodDeleted event
				deletedPodObj, castOk := psubMessage.OldObj.(*v1.Pod)
				if !castOk {
					log.Error().Msgf("Failed to cast to *v1.Pod: %v", psubMessage.OldObj)
					continue
				}

				podUID := deletedPodObj.GetObjectMeta().GetUID()
				if podIface, ok := mc.podUIDToCN.Load(podUID); ok {
					endpointCN := podIface.(certificate.CommonName)
					log.Warn().Msgf("Pod with UID %s found in Mesh Catalog; Releasing certificate %s", podUID, endpointCN)
					mc.certManager.ReleaseCertificate(endpointCN)

					// Request a broadcast update, just for security.
					// Dispatcher code also handles PodDelete, so probably the two will get coalesced.
					dispatcher.GetPubSubInstance().Publish(dispatcher.PubSubMessage{
						AnnouncementType: dispatcher.ScheduleProxyBroadcast,
						NewObj:           nil,
						OldObj:           nil,
					})
				} else {
					log.Warn().Msgf("Pod with UID %s not found in Mesh Catalog", podUID)
				}
			}
		}
	}()

	return stop
}
