package osm

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/models"
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
			cn := fmt.Sprintf("%s.%s.%s", tc.uuid, models.KindSidecar, tc.identity)

			kind, uuid, si, err := getCertificateCommonNameMeta(cn)

			assert.Equal(tc.err, err)

			if err == nil {
				assert.Equal(models.KindSidecar, kind)
				assert.Equal(tc.uuid, uuid)
				assert.Equal(tc.identity, si)
			}
		})
	}
}

func TestNewFromSpiffe(t *testing.T) {
	tests := []struct {
		name      string
		spiffeids []*url.URL
		want      string
		err       bool
	}{
		{
			name: "should extract the url properly",
			spiffeids: []*url.URL{
				{
					Scheme: "spiffe",
					Host:   "testdomain.local",
					Path:   "2f0acefe-ef08-4381-aae8-0ad42d02e402/sidecar/bookstore-v1/bookstore",
				},
			},
			want: "bookstore-v1.bookstore",
			err:  false,
		},
		{
			name: "should extract the url properly with forward slash",
			spiffeids: []*url.URL{
				{
					Scheme: "spiffe",
					Host:   "testdomain.local",
					Path:   "/2f0acefe-ef08-4381-aae8-0ad42d02e402/sidecar/bookstore-v1/bookstore",
				},
			},
			want: "bookstore-v1.bookstore",
			err:  false,
		},
		{
			name: "should error if not enough parts",
			spiffeids: []*url.URL{
				{
					Scheme: "spiffe",
					Host:   "testdomain.local",
					Path:   "notenough/bookstore-v1/bookstore",
				},
			},
			want: "",
			err:  true,
		},
		{
			name: "should error if too many parts",
			spiffeids: []*url.URL{
				{
					Scheme: "spiffe",
					Host:   "testdomain.local",
					Path:   "2f0acefe-ef08-4381-aae8-0ad42d02e402/sidecar/bookstore-v1/bookstore/toomany",
				},
			},
			want: "",
			err:  true,
		},
		{
			name: "should error if more than one uri",
			spiffeids: []*url.URL{
				{
					Scheme: "spiffe",
					Host:   "testdomain.local",
					Path:   "2f0acefe-ef08-4381-aae8-0ad42d02e402/sidecar/bookstore-v1/bookstore",
				},
				{
					Scheme: "spiffe",
					Host:   "testdomain.local",
					Path:   "2f0acefe-ef08-4381-aae8-0ad42d02e402/sidecar/bookstore-v1/bookstore",
				},
			},
			want: "",
			err:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)
			si, err := extractSpiffeID(tt.spiffeids)
			if tt.err {
				assert.Error(err)
				return
			}

			assert.NoError(err)
			assert.Equal(tt.want, si.String())
		})
	}
}
