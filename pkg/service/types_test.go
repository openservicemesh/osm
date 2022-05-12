package service

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
)

func TestClusterNameString(t *testing.T) {
	assert := tassert.New(t)

	clusterNameStr := uuid.New().String()
	cn := ClusterName(clusterNameStr)
	assert.Equal(cn.String(), clusterNameStr)
}

func TestMeshNameString(t *testing.T) {
	assert := tassert.New(t)

	namespace := uuid.New().String()
	name := uuid.New().String()
	ms := MeshService{
		Namespace: namespace,
		Name:      name,
	}

	assert.Equal(ms.String(), fmt.Sprintf("%s/%s", namespace, name))
	assert.Equal(ms.FQDN(), fmt.Sprintf("%s.%s.svc.cluster.local", name, namespace))
}

func TestMeshServiceCluster(t *testing.T) {
	testCases := []struct {
		name                     string
		meshSvc                  MeshService
		expectedClusterName      string
		expectedLocalClusterName string
	}{
		{
			name: "envoy cluster and local cluster name",
			meshSvc: MeshService{
				Namespace:  "ns1",
				Name:       "s1",
				Port:       80,
				TargetPort: 90,
			},
			expectedClusterName:      "ns1/s1|90",
			expectedLocalClusterName: "ns1/s1|90|local",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			assert.Equal(tc.expectedClusterName, tc.meshSvc.EnvoyClusterName())
			assert.Equal(tc.expectedLocalClusterName, tc.meshSvc.EnvoyLocalClusterName())
		})
	}
}
