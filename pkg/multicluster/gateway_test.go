package multicluster

import (
	"fmt"
	"strings"
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/envoy"
)

func TestMulticlusterHelpers(t *testing.T) {
	assert := tassert.New(t)
	serviceAccount := "-svc-account-"
	namespace := "-namespace-"

	actualCN := GetMulticlusterGatewaySubjectCommonName(serviceAccount, namespace)
	expectedSuffix := ".gateway.-svc-account-.-namespace-.cluster.local"
	assert.True(strings.HasSuffix(actualCN.String(), expectedSuffix), fmt.Sprintf("Expected the Proxy Cert's Common Name to end with %s", expectedSuffix))

	// Is the kind of proxy properly encoded in this certificate?
	actualProxyKind, err := envoy.GetKindFromProxyCertificate(actualCN)
	assert.Nil(err, fmt.Sprintf("Expected error to be nil; It was %+v", err))
	expectedProxyKind := envoy.KindGateway
	assert.Equal(expectedProxyKind, actualProxyKind, fmt.Sprintf("Expected proxy kind to be %s; it was actually %s", expectedProxyKind, actualProxyKind))
}
