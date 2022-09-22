package certificate

import (
	"context"
	"fmt"
	"time"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"k8s.io/client-go/util/retry"
)

type deferredCertificateStatusUpdate struct {
	certificateTypes         []CertificateType
	wait                     time.Duration
	mrcClient                MRCClient
	targetMRCName            string
	targetCertificateStatus  v1alpha2.MeshRootCertificateComponentStatus
	certificateStatusUpdates map[CertificateType]func() error
}

// a setFunc is a function that sets the certStatus of an MRC, returning true if the in-memory set was completed
// and false if not. A return value of false indicates that the k8s update of the MRC resource should not occur
type setFunc func(mrc *v1alpha2.MeshRootCertificate, status v1alpha2.MeshRootCertificateComponentStatus) bool

func (m *Manager) toValidatingRollout(ctx context.Context, from string, mrc *v1alpha2.MeshRootCertificate, mrcClient MRCClient) error {
	// Set the NewMRC's CA to the validating issuer
	client, ca, err := mrcClient.GetCertIssuerForMRC(mrc)
	if err != nil {
		return err
	}
	c := &issuer{Issuer: client, ID: mrc.Name, CertificateAuthority: ca, TrustDomain: mrc.Spec.TrustDomain}
	m.mu.Lock()
	m.validatingIssuer = c
	m.mu.Unlock()

	// TODO: Set `wait` to real value from MRC once API changes are merged
	// TODO: Change `targetCertificateStatus` to constant
	d := newDeferredCertificateStatusUpdate(30*time.Minute, m.ownedCertTypes, mrcClient, mrc.Name, v1alpha2.MeshRootCertificateComponentStatus("Validating"))

	go d.do(ctx)

	return nil
}

func newDeferredCertificateStatusUpdate(wait time.Duration, certificateTypes []CertificateType, mrcClient MRCClient, targetMRCName string, targetCertStatus v1alpha2.MeshRootCertificateComponentStatus) *deferredCertificateStatusUpdate {
	d := &deferredCertificateStatusUpdate{
		wait:                    wait,
		certificateTypes:        certificateTypes,
		mrcClient:               mrcClient,
		targetMRCName:           targetMRCName,
		targetCertificateStatus: targetCertStatus,
	}

	d.certificateStatusUpdates = map[CertificateType]func() error{
		CertificateTypeSidecar: d.genUpdateFunc(func(mrc *v1alpha2.MeshRootCertificate, status v1alpha2.MeshRootCertificateComponentStatus) bool {
			if mrc.Status.ComponentStatuses.Sidecar == status {
				return false
			}
			mrc.Status.ComponentStatuses.Sidecar = status

			return true
		}),
		CertificateTypeBootstrap: d.genUpdateFunc(func(mrc *v1alpha2.MeshRootCertificate, status v1alpha2.MeshRootCertificateComponentStatus) bool {
			if mrc.Status.ComponentStatuses.Bootstrap == status {
				return false
			}
			mrc.Status.ComponentStatuses.Bootstrap = status

			return true
		}),
		CertificateTypeMutatingWebhook: d.genUpdateFunc(func(mrc *v1alpha2.MeshRootCertificate, status v1alpha2.MeshRootCertificateComponentStatus) bool {
			// TODO: Change to mutatingWebhook specific field when MRC API changes are merged
			if mrc.Status.ComponentStatuses.Webhooks == status {
				return false
			}
			mrc.Status.ComponentStatuses.Webhooks = status

			return true
		}),
		CertificateTypeValidatingWebhook: d.genUpdateFunc(func(mrc *v1alpha2.MeshRootCertificate, status v1alpha2.MeshRootCertificateComponentStatus) bool {
			// TODO: Change to validatingWebhook specific field when MRC API changes are merged
			if mrc.Status.ComponentStatuses.Webhooks == status {
				return false
			}
			mrc.Status.ComponentStatuses.Webhooks = status

			return true
		}),
		CertificateTypeXDSControlPlane: d.genUpdateFunc(func(mrc *v1alpha2.MeshRootCertificate, status v1alpha2.MeshRootCertificateComponentStatus) bool {
			if mrc.Status.ComponentStatuses.XDSControlPlane == status {
				return false
			}
			mrc.Status.ComponentStatuses.XDSControlPlane = status

			return true
		}),
		CertificateTypeGateway: d.genUpdateFunc(func(mrc *v1alpha2.MeshRootCertificate, status v1alpha2.MeshRootCertificateComponentStatus) bool {
			if mrc.Status.ComponentStatuses.Gateway == status {
				return false
			}
			mrc.Status.ComponentStatuses.Gateway = status

			return true
		}),
	}

	return d
}

func (d *deferredCertificateStatusUpdate) genUpdateFunc(set setFunc) func() error {
	return func() error {
		mrc := d.mrcClient.GetMeshRootCertificate(d.targetMRCName)
		if mrc == nil {
			return fmt.Errorf("no MRC found with name %s in the osm control plane namespace", d.targetMRCName)
		}

		if !set(mrc, d.targetCertificateStatus) {
			return nil
		}

		_, err := d.mrcClient.UpdateMeshRootCertificateStatus(mrc)
		if err != nil {
			log.Error().Err(err).Msgf("Error updating MRC status")
			return err
		}

		return nil
	}
}

func (d *deferredCertificateStatusUpdate) do(ctx context.Context) {
	t := time.NewTimer(d.wait)
	select {
	case <-ctx.Done():
		t.Stop()
	case <-t.C:
		for _, certType := range d.certificateTypes {
			retry.RetryOnConflict(retry.DefaultRetry, func() error {
				return d.certificateStatusUpdates[certType]()
			})
		}
	}
}
