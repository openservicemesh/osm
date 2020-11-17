package catalog

import (
	"errors"

	"k8s.io/apimachinery/pkg/types"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
)

var errEventNotHandled = errors.New("event not handled")

// releaseCertificate is an Announcement handler, which on receiving a PodDeleted event
// it releases the xDS certificate for the Envoy for that Pod.
func (mc *MeshCatalog) releaseCertificate(ann announcements.Announcement) error {
	whatWeGot := ann.Type
	whatWeCanHandle := announcements.PodDeleted
	if whatWeCanHandle != whatWeGot {
		log.Error().Msgf("releaseCertificate function received an announcement with type %s; it can only handle %s", whatWeGot, whatWeCanHandle)
		return errEventNotHandled
	}

	if podUID, ok := ann.ReferencedObjectID.(types.UID); ok {
		if podIface, ok := mc.podUIDToCN.Load(podUID); ok {
			endpointCN := podIface.(certificate.CommonName)
			log.Warn().Msgf("Pod with UID %s found in Mesh Catalog; Releasing certificate %s", podUID, endpointCN)
			mc.certManager.ReleaseCertificate(endpointCN)
		} else {
			log.Warn().Msgf("Pod with UID %s not found in Mesh Catalog", podUID)
		}
	}

	return nil
}

// updateRelatedProxies is an Announcement handler, which augments the handling of PodDeleted events
// and leverages broadcastToAllProxies() to let all proxies know that something has changed.
// TODO: The use of broadcastToAllProxies() needs to be deprecated in favor of more granular approach.
func (mc *MeshCatalog) updateRelatedProxies(ann announcements.Announcement) error {
	whatWeGot := ann.Type
	whatWeCanHandle := announcements.PodDeleted
	if whatWeCanHandle != whatWeGot {
		log.Error().Msgf("updateRelatedProxies function received an announcement with type %s; it can only handle %s", whatWeGot, whatWeCanHandle)
		return errEventNotHandled
	}

	// TODO: the function below updates all proxies; understand what proxies need to be updated and update only these
	mc.broadcastToAllProxies(ann)

	return nil
}
