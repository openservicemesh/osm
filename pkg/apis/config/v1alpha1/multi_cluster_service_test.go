package v1alpha1

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestMultiClusterServiceStringer tests MultiClusterService Stringer interface implementation.
func TestMultiClusterServiceStringer(t *testing.T) {
	assert := tassert.New(t)

	multiclusterService := MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "name",
		},

		Spec: MultiClusterServiceSpec{
			Clusters: []ClusterSpec{
				{
					Address: "1.2.3.4:8080",
					Name:    "remote-cluster-1",
				},
				{
					Address: "5.6.7.8:8080",
					Name:    "remote-cluster-2",
				},
			},
			ServiceAccount: "sa",
		},
	}

	assert.Equal(fmt.Sprintf("MCS=%s", multiclusterService), "MCS=namespace/name with SA=sa")
}
