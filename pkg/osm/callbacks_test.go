package osm

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
)

func TestGetCertificateCommonNameMeta(t *testing.T) {
	testCases := []struct {
		name     string
		uuid     uuid.UUID
		identity identity.ServiceIdentity
		err      error
	}{
		{
			name:     "valid cn",
			uuid:     uuid.New(),
			identity: identity.New("foo", "bar"),
		},
		{
			name:     "invalid uuid",
			uuid:     uuid.Nil,
			identity: identity.New("foo", "bar"),
		},
		{
			name:     "invalid identity",
			uuid:     uuid.New(),
			identity: identity.New("foo", ""),
			err:      errInvalidCertificateCN,
		},
		{
			name: "no identity",
			uuid: uuid.New(),
			err:  errInvalidCertificateCN,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", tc.uuid, envoy.KindSidecar, tc.identity))

			kind, uuid, si, err := getCertificateCommonNameMeta(cn)

			assert.Equal(tc.err, err)

			if err == nil {
				assert.Equal(envoy.KindSidecar, kind)
				assert.Equal(tc.uuid, uuid)
				assert.Equal(tc.identity, si)
			}
		})
	}
}
