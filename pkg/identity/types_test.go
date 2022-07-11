package identity

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestServiceIdentityType(t *testing.T) {
	assert := tassert.New(t)

	// Test String()
	si := ServiceIdentity("foo.bar")
	assert.Equal("foo.bar", si.String())

	// Test wildcard()
	wildcard := ServiceIdentity("*")
	assert.True(wildcard.IsWildcard())
	notWildcard := ServiceIdentity("foo.bar")
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
	assert.Equal(ServiceIdentity("foo.bar"), svcAccount.ToServiceIdentity())
}

func TestToServiceIdentity(t *testing.T) {
	testCases := []struct {
		svcAccount              K8sServiceAccount
		expectedServiceIdentity ServiceIdentity
	}{
		{
			K8sServiceAccount{Name: "foo", Namespace: "bar"},
			ServiceIdentity("foo.bar"),
		},
		{
			K8sServiceAccount{Name: "foo", Namespace: "baz"},
			ServiceIdentity("foo.baz"),
		},
	}

	for _, tc := range testCases {
		assert := tassert.New(t)

		si := tc.svcAccount.ToServiceIdentity()
		assert.Equal(si, tc.expectedServiceIdentity)
	}
}
