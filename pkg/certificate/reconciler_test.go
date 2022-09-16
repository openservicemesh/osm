package certificate

import (
	"context"
	"fmt"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/compute/kube"
	"github.com/openservicemesh/osm/pkg/constants"
	fakeConfigClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestCheckAndUpdate(t *testing.T) {
	a := tassert.New(t)

	mrcClient := &fakeMRCClient{}
	checkStatus := func(ctx context.Context, mrc *v1alpha2.MeshRootCertificate) (bool, error) {
		if mrc.Status.State != constants.MRCStatePending {
			return false, fmt.Errorf("incorrect rotation state")
		}
		return (mrc.Status.ComponentStatuses.Bootstrap == constants.MRCComponentStatusValidating &&
				mrc.Status.ComponentStatuses.Gateway == constants.MRCComponentStatusValidating &&
				mrc.Status.ComponentStatuses.Sidecar == constants.MRCComponentStatusValidating &&
				mrc.Status.ComponentStatuses.XDSControlPlane == constants.MRCComponentStatusValidating &&
				mrc.Status.ComponentStatuses.Webhooks == constants.MRCComponentStatusValidating),
			nil
	}
	updateStatus := func(ctx context.Context, mrc *v1alpha2.MeshRootCertificate) error {
		mrc.Status.State = constants.MRCStateValidating
		mrc.Status.Conditions = append(mrc.Status.Conditions, v1alpha2.MeshRootCertificateCondition{
			Type:   constants.MRCConditionTypeValidatingRollout,
			Status: "True",
			Reason: "CertificatePassivelyInUse",
		})
		if mrcClient == nil {
			return fmt.Errorf("error")
		}
		_, err := mrcClient.UpdateMeshRootCertificateStatus(mrc)
		a.NoError(err)
		return nil
	}
	mrcName := "osm-mesh-root-certificate"

	testCases := []struct {
		name             string
		currentResource  v1alpha2.MeshRootCertificate
		expectedResource v1alpha2.MeshRootCertificate
	}{
		{
			name: "successfully updated MRC",
			currentResource: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mrcName,
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Tresor: &v1alpha2.TresorProviderSpec{
							CA: v1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
				},
				Intent: constants.MRCIntentPassive,
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
					ComponentStatuses: v1alpha2.MeshRootCertificateComponentStatuses{
						Webhooks:        constants.MRCComponentStatusValidating,
						XDSControlPlane: constants.MRCComponentStatusValidating,
						Sidecar:         constants.MRCComponentStatusValidating,
						Bootstrap:       constants.MRCComponentStatusValidating,
						Gateway:         constants.MRCComponentStatusValidating,
					},
				},
			},
			expectedResource: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mrcName,
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Tresor: &v1alpha2.TresorProviderSpec{
							CA: v1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
				},
				Intent: constants.MRCIntentPassive,
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidating,
					ComponentStatuses: v1alpha2.MeshRootCertificateComponentStatuses{
						Webhooks:        constants.MRCComponentStatusValidating,
						XDSControlPlane: constants.MRCComponentStatusValidating,
						Sidecar:         constants.MRCComponentStatusValidating,
						Bootstrap:       constants.MRCComponentStatusValidating,
						Gateway:         constants.MRCComponentStatusValidating,
					},
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: "True",
							Reason: "CertificatePassivelyInUse",
						},
					},
				},
			},
		},
		{
			name: "MRC not updated. checkStatus errors",
			currentResource: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mrcName,
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Tresor: &v1alpha2.TresorProviderSpec{
							CA: v1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
				},
				Intent: constants.MRCIntentPassive,
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidating,
					ComponentStatuses: v1alpha2.MeshRootCertificateComponentStatuses{
						Webhooks:        constants.MRCComponentStatusValidating,
						XDSControlPlane: constants.MRCComponentStatusValidating,
						Sidecar:         constants.MRCComponentStatusValidating,
						Bootstrap:       constants.MRCComponentStatusValidating,
						Gateway:         constants.MRCComponentStatusValidating,
					},
				},
			},
			expectedResource: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mrcName,
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Tresor: &v1alpha2.TresorProviderSpec{
							CA: v1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
				},
				Intent: constants.MRCIntentPassive,
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidating,
					ComponentStatuses: v1alpha2.MeshRootCertificateComponentStatuses{
						Webhooks:        constants.MRCComponentStatusValidating,
						XDSControlPlane: constants.MRCComponentStatusValidating,
						Sidecar:         constants.MRCComponentStatusValidating,
						Bootstrap:       constants.MRCComponentStatusValidating,
						Gateway:         constants.MRCComponentStatusValidating,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stop := make(chan struct{})
			defer close(stop)

			configClient := fakeConfigClientset.NewSimpleClientset(&tc.currentResource)
			ic, err := informers.NewInformerCollection("osm-system", stop, informers.WithConfigClient(configClient, tests.OsmMeshConfigName, "osm-system"))
			_ = ic.Add(informers.InformerKeyMeshRootCertificate, tc.currentResource, t)
			a.NoError(err)
			k8sClient := k8s.NewClient("osm-system", tests.OsmMeshConfigName, ic, nil, configClient, messaging.NewBroker(stop))
			kubeClient := kube.NewClient(k8sClient)
			mrcClient.Interface = kubeClient
			a.NotNil(mrcClient.Interface)

			mrcReconciler := MRCReconciler{mrcName, updateStatus, checkStatus}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Start mrc reconciliation loop
			mrcReconciler.CheckAndUpdate(ctx, mrcClient, 5*time.Millisecond)
			time.Sleep(1 * time.Second)

			updatedMRC := mrcClient.GetMeshRootCertificate(mrcName)
			a.Equal(&tc.expectedResource, updatedMRC)
		})
	}
}
