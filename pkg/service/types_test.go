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
