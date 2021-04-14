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
			ServiceIdentity{
				kind:           KubernetesServiceAccount,
				serviceAccount: K8sServiceAccount{Name: "foo", Namespace: "bar"},
				trustDomain:    ClusterLocalTrustDomain,
			},
		},
		{
			K8sServiceAccount{Name: "foo", Namespace: "bar"},
			"cluster.baz.one.two.three.four",
			ServiceIdentity{
				kind:           KubernetesServiceAccount,
				serviceAccount: K8sServiceAccount{Name: "foo", Namespace: "bar"},
				trustDomain:    "cluster.baz.one.two.three.four",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing NewFromKubernetesServiceAccount for test case: %v", tc), func(t *testing.T) {
			si := NewFromKubernetesServiceAccount(tc.svcAccount, tc.trustDomain)
			assert.Equal(si, tc.expectedServiceIdentity)
		})
	}

	svcIdent, err := NewFromRFC1123("foo.bar.baz.one.two.three", KubernetesServiceAccount)
	assert.Nil(err)
	assert.Equal(svcIdent.String(), "foo.bar.baz.one.two.three")
}
