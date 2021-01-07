package identity

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/service"
)

func TestGetKubernetesServiceIdentity(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		svcAccount              service.K8sServiceAccount
		trustDomain             string
		expectedServiceIdentity ServiceIdentity
	}{
		{
			service.K8sServiceAccount{Name: "foo", Namespace: "bar"},
			"cluster.local",
			ServiceIdentity("foo.bar.cluster.local"),
		},
		{
			service.K8sServiceAccount{Name: "foo", Namespace: "bar"},
			"cluster.baz",
			ServiceIdentity("foo.bar.cluster.baz"),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing GetKubernetesServiceIdentity for test case: %v", tc), func(t *testing.T) {
			si := GetKubernetesServiceIdentity(tc.svcAccount, tc.trustDomain)
			assert.Equal(si, tc.expectedServiceIdentity)
		})
	}
}
