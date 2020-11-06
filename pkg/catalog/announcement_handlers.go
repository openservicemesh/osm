package catalog

import (
	"errors"

	"k8s.io/apimachinery/pkg/types"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
)

var errEventNotHandled = errors.New("event not handled")

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
