package identity

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestServiceIdentityType(t *testing.T) {
	assert := tassert.New(t)

	// Test String()
	si := ServiceIdentity("foo.bar.cluster.local")
	assert.Equal("foo.bar.cluster.local", si.String())

	// Test wildcard()
	wildcard := ServiceIdentity("*")
	assert.True(wildcard.IsWildcard())
	notWildcard := ServiceIdentity("foo.bar.cluster.local")
	assert.False(notWildcard.IsWildcard())

	// Test ToK8sServiceAccount()
	assert.Equal(K8sServiceAccount{Name: "foo", Namespace: "bar"}, si.ToK8sServiceAccount())
}

func TestK8sServiceAccountType(t *testing.T) {
	assert := tassert.New(t)

	// Test String()
	svcAccount := K8sServiceAccount{Name: "foo", Namespace: "bar"}
	assert.Equal("bar/foo", svcAccount.String())

	// Test ToServiceIdentity
	assert.Equal(ServiceIdentity("foo.bar.cluster.local"), svcAccount.ToServiceIdentity())
}
