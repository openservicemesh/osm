package certificate

import "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

func getMRCCondition(mrc *v1alpha2.MeshRootCertificate, conditionType v1alpha2.MeshRootCertificateConditionType) *v1alpha2.MeshRootCertificateCondition {
	for _, condition := range mrc.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}
