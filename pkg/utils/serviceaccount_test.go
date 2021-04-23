package utils

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestSvcAccountToK8sSvcAccount(t *testing.T) {
	assert := tassert.New(t)

	sa := tests.NewServiceAccountFixture(tests.BookbuyerServiceAccountName, tests.Namespace)
	svcAccount := SvcAccountToK8sSvcAccount(sa)
	expectedSvcAccount := identity.K8sServiceAccount{
		Name:      tests.BookbuyerServiceAccountName,
		Namespace: tests.Namespace,
	}

	assert.Equal(svcAccount, expectedSvcAccount)
}
