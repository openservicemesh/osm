package main

import (
	"context"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
)

func Test_rotateCmd_findCurrentActive(t *testing.T) {
	tests := []struct {
		name        string
		existingMrc []*v1alpha2.MeshRootCertificate
		wantErr     bool
	}{
		{
			name:    "no mrcs returns error",
			wantErr: true,
		},
		{
			name:    "mrcs in bad state returns error",
			wantErr: true,
			existingMrc: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "badstate",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.PassiveIntent,
					},
				},
			},
		},
		{
			name:    "rotation in progress",
			wantErr: true,
			existingMrc: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "passive",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.PassiveIntent,
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "active",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.ActiveIntent,
					},
				},
			},
		},
		{
			name:    "two active should error",
			wantErr: true,
			existingMrc: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "active",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.ActiveIntent,
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "active2",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.ActiveIntent,
					},
				},
			},
		},
		{
			name:    "only one active should return name",
			wantErr: false,
			existingMrc: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "active",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.ActiveIntent,
					},
				},
			},
		},
		{
			name:    "only one active should return name with some inactive",
			wantErr: false,
			existingMrc: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "active",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.ActiveIntent,
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "inactive1",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.InactiveIntent,
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "inactive2",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.InactiveIntent,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)
			r := &rotateCmd{
				clientSet:    fake.NewSimpleClientset(),
				configClient: fakeConfig.NewSimpleClientset(),
			}

			for _, mrc := range tt.existingMrc {
				_, err := r.configClient.ConfigV1alpha2().MeshRootCertificates(settings.Namespace()).Create(context.Background(), mrc, v1.CreateOptions{})
				assert.NoError(err)
			}
			mrc, err := r.findCurrentInUse()

			if tt.wantErr {
				assert.Nil(mrc)
				assert.NotNil(err)
			} else {
				assert.Equal("active", mrc.Name)
				assert.Equal(v1alpha2.ActiveIntent, mrc.Spec.Intent)
			}
		})
	}
}
