package certificate

import (
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
)

// componentsAreIssuing returns true if all certificate component statuses are `issuing`
func componentsAreIssuing(mrc *v1alpha2.MeshRootCertificate) bool {
	return mrc.Status.ComponentStatuses.Bootstrap == constants.MRCComponentStatusIssuing &&
		mrc.Status.ComponentStatuses.Gateway == constants.MRCComponentStatusIssuing &&
		mrc.Status.ComponentStatuses.Sidecar == constants.MRCComponentStatusIssuing &&
		mrc.Status.ComponentStatuses.Webhooks == constants.MRCComponentStatusIssuing &&
		mrc.Status.ComponentStatuses.XDSControlPlane == constants.MRCComponentStatusIssuing
}

// componentsHaveError returns true if any certificate component statuses are `error`
func componentsHaveError(mrc *v1alpha2.MeshRootCertificate) bool {
	return mrc.Status.ComponentStatuses.Bootstrap == constants.MRCComponentStatusError &&
		mrc.Status.ComponentStatuses.Gateway == constants.MRCComponentStatusError &&
		mrc.Status.ComponentStatuses.Sidecar == constants.MRCComponentStatusError &&
		mrc.Status.ComponentStatuses.Webhooks == constants.MRCComponentStatusError &&
		mrc.Status.ComponentStatuses.XDSControlPlane == constants.MRCComponentStatusError
}
