package identity

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestGetKubernetesServiceIdentity(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		svcAccount              K8sServiceAccount
		trustDomain             string
		expectedServiceIdentity ServiceIdentity
	}{
		{
			K8sServiceAccount{Name: "foo", Namespace: "bar"},
			"cluster.local",
			ServiceIdentity{"foo", "bar", "cluster.local"},
		},
		{
			K8sServiceAccount{Name: "foo", Namespace: "bar"},
			"cluster.baz",
			ServiceIdentity{"foo", "bar", "cluster.baz"},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing GetKubernetesServiceIdentity for test case: %v", tc), func(t *testing.T) {
			si := GetKubernetesServiceIdentity(tc.svcAccount, tc.trustDomain)
			assert.Equal(si, tc.expectedServiceIdentity)
		})
	}

	assert.Equal(ServiceIdentity{"foo", "", ""}.String(), "foo")
}

func TestServiceIdentityToString(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		identity       ServiceIdentity
		expectedString string
	}{
		{
			ServiceIdentity{
				ServiceAccount: "sa",
			},
			"sa",
		},
		{
			ServiceIdentity{
				ServiceAccount: "sa",
				Namespace:      "ns",
			},
			"sa.ns",
		},
		{
			ServiceIdentity{
				ServiceAccount: "sa",
				Namespace:      "ns",
				ClusterDomain:  "a.n.y",
			},
			"sa.ns.a.n.y",
		},
	}

	for _, tc := range testCases {
		assert.Equal(tc.identity.String(), tc.expectedString)
	}
}

func TestStringToServiceIdentity(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		identityString   string
		expectedIdentity ServiceIdentity
	}{
		{
			"sa",
			ServiceIdentity{
				ServiceAccount: "sa",
			},
		},
		{
			"sa.ns",
			ServiceIdentity{
				ServiceAccount: "sa",
				Namespace:      "ns",
			},
		},
		{
			"sa.ns.a.n.y",
			ServiceIdentity{
				ServiceAccount: "sa",
				Namespace:      "ns",
				ClusterDomain:  "a.n.y",
			},
		},
	}

	for _, tc := range testCases {
		assert.Equal(NewServiceIdentityFromString(tc.identityString), tc.expectedIdentity)
	}
}
