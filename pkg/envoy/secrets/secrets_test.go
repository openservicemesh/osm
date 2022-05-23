package secrets

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestUnmarshalSDSCert(t *testing.T) {
	require := trequire.New(t)

	namespace := "randomNamespace"
	serviceName := "randomServiceName"
	meshService := service.MeshService{
		Namespace: namespace,
		Name:      serviceName,
	}

	si := identity.ServiceIdentity("randomServiceAccountName.randomNamespace.cluster.local")

	str := meshService.String()
	fmt.Println(str)

	testCases := []struct {
		name        string
		expectedErr bool
		str         string
	}{
		{
			name:        "successfully unmarshal service",
			expectedErr: false,
			str:         "root-cert-for-mtls-outbound:randomNamespace/randomServiceName",
		},
		{
			name:        "incomplete namespaced service name 1",
			expectedErr: true,
			str:         "root-cert-for-mtls-outbound:/svnc",
		},
		{
			name:        "incomplete namespaced service name 2",
			expectedErr: true,
			str:         "root-cert-for-mtls-outbound:svnc/",
		},
		{
			name:        "incomplete namespaced service name 3",
			expectedErr: true,
			str:         "root-cert-for-mtls-outbound:/svnc/",
		},
		{
			name:        "incomplete namespaced service name 3",
			expectedErr: true,
			str:         "root-cert-for-mtls-outbound:/",
		},
		{
			name:        "incomplete namespaced service name 3",
			expectedErr: true,
			str:         "",
		},
		{
			name:        "incomplete namespaced service name 3",
			expectedErr: true,
			str:         "root-cert-for-mtls-outbound:test",
		},
		{
			name:        "successfully unmarshal service account",
			expectedErr: false,
			str:         "root-cert-for-mtls-inbound:randomServiceAccountName.randomNamespace.cluster.local",
		},
		{
			name:        "incomplete namespaced service account name 1",
			expectedErr: true,
			str:         "root-cert-for-mtls-inbound:.svnc",
		},
		{
			name:        "incomplete namespaced service account name 2",
			expectedErr: true,
			str:         "root-cert-for-mtls-inbound:svnc.",
		},
		{
			name:        "incomplete namespaced service account name 3",
			expectedErr: true,
			str:         "service-cert:.svnc.",
		},
		{
			name:        "incomplete namespaced service account name 3",
			expectedErr: true,
			str:         "service-cert:.",
		},
		{
			name:        "incomplete namespaced service account name 3",
			expectedErr: true,
			str:         "service-cert:",
		},
		{
			name:        "incomplete namespaced service account name 3",
			expectedErr: true,
			str:         "service-cert:test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual, err := UnmarshalSDSCert(tc.str)

			if tc.expectedErr {
				assert.NotNil(err)
				return
			}
			require.Nil(err)

			switch v := actual.(type) {
			case *SDSServiceCert:
				assert.Equal(si, v.GetServiceIdentity())
			case *SDSInboundRootCert:
				assert.Equal(si, v.GetServiceIdentity())
			case *SDSOutboundRootCert:
				assert.Equal(meshService, v.GetMeshService())
			default:
				assert.Fail("unexpected type")
			}
		})
	}
}
