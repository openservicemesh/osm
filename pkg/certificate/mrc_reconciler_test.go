package certificate

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/compute/kube"
	"github.com/openservicemesh/osm/pkg/constants"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestShouldEnsureIssuerForMRC(t *testing.T) {
	type testCase struct {
		name           string
		mrc            *v1alpha2.MeshRootCertificate
		leaderMode     bool
		expectedReturn bool
	}
	testCases := []testCase{
		{
			name: "should retrieve CA",
			mrc: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "cluster.local",
					Intent:      v1alpha2.Passive,
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
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
				},
			},
			leaderMode:     true,
			expectedReturn: true,
		},
		{
			name: "should not retrieve CA, not in leader mode",
			mrc: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "cluster.local",
					Intent:      v1alpha2.Passive,
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
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
				},
			},
			leaderMode:     false,
			expectedReturn: false,
		},
		{
			name: "should not retrieve CA, intent is not passive",
			mrc: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "cluster.local",
					Intent:      v1alpha2.Passive,
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
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
				},
			},
			leaderMode:     true,
			expectedReturn: false,
		},
		{
			name: "should not retrieve CA, conditions already set",
			mrc: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "cluster.local",
					Intent:      v1alpha2.Passive,
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
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:    v1alpha2.Accepted,
							Status:  v1.ConditionTrue,
							Reason:  certificateAcceptedReason,
							Message: "certificate accepted",
						},
						{
							Type:    v1alpha2.ValidatingRollout,
							Status:  v1.ConditionFalse,
							Reason:  passiveStateValidatingReason,
							Message: "passive intent",
						},
						{
							Type:    v1alpha2.IssuingRollout,
							Status:  v1.ConditionFalse,
							Reason:  passiveStateIssuingReason,
							Message: "passive intent",
						},
					},
				},
			},
			leaderMode:     true,
			expectedReturn: false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			m := &Manager{
				leaderMode: tc.leaderMode,
			}

			ret := m.shouldEnsureIssuerForMRC(tc.mrc)
			assert.Equal(tc.expectedReturn, ret)
		})
	}
}

func TestUpdateMRCState(t *testing.T) {
	type testCase struct {
		name         string
		configClient *configFake.Clientset
		mrc          *v1alpha2.MeshRootCertificate
		expectedMRC  *v1alpha2.MeshRootCertificate
		leaderMode   bool
	}
	testCases := []testCase{
		{
			name:       "update MRC state in leader mode",
			leaderMode: true,
			configClient: configFake.NewSimpleClientset([]runtime.Object{&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "cluster.local",
					Intent:      v1alpha2.Passive,
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
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
				},
			}}...),
			mrc: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "cluster.local",
					Intent:      v1alpha2.Passive,
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
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
				},
			},
			expectedMRC: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "cluster.local",
					Intent:      v1alpha2.Passive,
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
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:    v1alpha2.Accepted,
							Status:  v1.ConditionTrue,
							Reason:  certificateAcceptedReason,
							Message: "certificate accepted",
						},
						{
							Type:    v1alpha2.ValidatingRollout,
							Status:  v1.ConditionFalse,
							Reason:  passiveStateValidatingReason,
							Message: "passive intent",
						},
						{
							Type:    v1alpha2.IssuingRollout,
							Status:  v1.ConditionFalse,
							Reason:  passiveStateIssuingReason,
							Message: "passive intent",
						},
					},
				},
			},
		},
		{
			name:       "MRC state not updated when manager not in leader mode",
			leaderMode: false,
			configClient: configFake.NewSimpleClientset([]runtime.Object{&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "cluster.local",
					Intent:      v1alpha2.Passive,
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
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
				},
			}}...),
			mrc: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "cluster.local",
					Intent:      v1alpha2.Passive,
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
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
				},
			},
			expectedMRC: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "cluster.local",
					Intent:      v1alpha2.Passive,
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
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStatePending,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			stop := make(chan struct{})
			defer close(stop)

			msgBroker := messaging.NewBroker(stop)

			ic, err := informers.NewInformerCollection("osm", stop, informers.WithConfigClient(tc.configClient, tests.OsmMeshConfigName, tests.OsmNamespace))
			assert.NoError(err)
			assert.NotNil(ic)

			err = ic.Add(informers.InformerKeyMeshRootCertificate, tc.mrc, t)
			assert.NoError(err)

			k8sClient := k8s.NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, ic, nil, nil, tc.configClient, msgBroker)
			assert.NotNil(k8sClient)

			computeClient := kube.NewClient(k8sClient)

			m := &Manager{
				leaderMode: tc.leaderMode,
				mrcClient:  &fakeMRCClient{computeClient},
			}

			err = m.updateMRCState(tc.mrc)
			assert.NoError(err)

			updatedMRC := computeClient.GetMeshRootCertificate(tc.expectedMRC.Name)
			assert.NotNil(updatedMRC)
			assert.Equal(tc.expectedMRC.Spec, updatedMRC.Spec)
			assert.Len(updatedMRC.Status.Conditions, len(tc.expectedMRC.Status.Conditions))

			for _, cond := range updatedMRC.Status.Conditions {
				found := false
				for _, expCond := range tc.expectedMRC.Status.Conditions {
					if expCond.Type == cond.Type {
						found = true
						assert.Equal(expCond.Status, cond.Status)
						assert.Equal(expCond.Reason, cond.Reason)
						assert.Equal(expCond.Message, cond.Message)
					}
				}
				assert.True(found)
			}
		})
	}
}
