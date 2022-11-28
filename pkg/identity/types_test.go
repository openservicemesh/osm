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

func TestServiceIdentity_AsPrincipal(t *testing.T) {
	tests := []struct {
		name              string
		si                ServiceIdentity
		trustDomain       string
		spiffeEnabled     bool
		expectedPrincipal string
	}{
		{
			name:              "TrustDomain is appended to principal in CN format",
			si:                ServiceIdentity("foo.bar"),
			trustDomain:       "cluster.local",
			spiffeEnabled:     false,
			expectedPrincipal: "foo.bar.cluster.local",
		},
		{
			name:              "TrustDomain is in spiffe format",
			si:                ServiceIdentity("foo.bar"),
			trustDomain:       "cluster.local",
			spiffeEnabled:     true,
			expectedPrincipal: "spiffe://cluster.local/foo/bar",
		},
		{
			name:              "TrustDomain is wild card in CN format",
			si:                WildcardServiceIdentity,
			trustDomain:       "cluster.local",
			spiffeEnabled:     false,
			expectedPrincipal: "*",
		},
		{
			name:              "TrustDomain is wild card in spiffe format",
			si:                WildcardServiceIdentity,
			trustDomain:       "cluster.local",
			spiffeEnabled:     true,
			expectedPrincipal: "spiffe://cluster.local",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			got := tc.si.AsPrincipal(tc.trustDomain, tc.spiffeEnabled)
			assert.Equal(tc.expectedPrincipal, got)
		})
	}
}
