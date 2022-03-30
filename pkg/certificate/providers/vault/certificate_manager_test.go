package vault

import (
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/builtin/logical/pki"
	vhttp "github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/sdk/logical"
	vault "github.com/hashicorp/vault/vault"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

var vaultRole = "default_role"
var _ = Describe("Test client helpers", func() {

	Context("Test creating a Certificate from Hashi Vault Secret", func() {
		It("creates a Certificate struct from Hashi Vault Secret struct", func() {

			cn := certificate.CommonName("foo.bar.co.uk")

			secret := &api.Secret{
				Data: map[string]interface{}{
					certificateField:  "xx",
					privateKeyField:   "yy",
					issuingCAField:    "zz",
					serialNumberField: "123",
				},
			}

			expiration := time.Now().Add(1 * time.Hour)

			actual := newCert(cn, secret, expiration)

			expected := &certificate.Certificate{
				IssuingCA:    pem.RootCertificate("zz"),
				PrivateKey:   pem.PrivateKey("yy"),
				CertChain:    pem.Certificate("xx"),
				Expiration:   expiration,
				CommonName:   "foo.bar.co.uk",
				SerialNumber: "123",
			}

			Expect(actual).To(Equal(expected))
		})
	})
})

func TestNew(t *testing.T) {
	token, addr := mockVault(t)

	testCases := []struct {
		description string
		vaultaddr   string
		token       string
		role        string
		wantErr     bool
	}{
		{
			description: "valid inputs",
			vaultaddr:   addr,
			role:        vaultRole,
			token:       token,
			wantErr:     false,
		},
		{
			description: "error with all empty inputs",
			vaultaddr:   "",
			role:        "",
			token:       "",
			wantErr:     true,
		},
		{
			description: "error with no role",
			vaultaddr:   addr,
			role:        "",
			token:       token,
			wantErr:     true,
		},
		{
			description: "error with no addr",
			vaultaddr:   "",
			role:        vaultRole,
			token:       token,
			wantErr:     true,
		},
		{
			description: "error with no token",
			vaultaddr:   addr,
			role:        vaultRole,
			token:       "",
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tassert := assert.New(t)
			_, err := New(tc.vaultaddr, tc.token, tc.role)
			if tc.wantErr {
				tassert.Error(err, "expected error, got nil")
			} else {
				tassert.NoError(err, "did not expect error, got %v", err)
			}
		})
	}
}

func TestIssueCertificate(t *testing.T) {
	var commonName certificate.CommonName = "localhost"
	var validityPeriod = time.Hour

	token, addr := mockVault(t)

	cm, err := New(addr, token, vaultRole)
	if err != nil {
		t.Fatalf("did not expect error, got %v", err)
	}
	mockVaultwithPKI(t, cm)

	testCases := []struct {
		description string
		cn          certificate.CommonName
		vP          time.Duration
		wantErr     bool
	}{
		{
			description: "valid inputs",
			cn:          commonName,
			vP:          validityPeriod,
			wantErr:     false,
		},
		{
			description: "error with invalid common name",
			cn:          certificate.CommonName(" "),
			vP:          time.Duration(-1),
			wantErr:     true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tassert := assert.New(t)
			_, err = cm.IssueCertificate(tc.cn, tc.vP)
			if tc.wantErr {
				tassert.Error(err, "expected error, got nil")
			} else {
				tassert.NoError(err, "did not expect error, got %v", err)
			}
		})
	}
}

func mockVault(t *testing.T) (string, string) {
	coreConfig := &vault.CoreConfig{
		LogicalBackends: map[string]logical.Factory{
			"pki": pki.Factory,
		},
	}
	core, _, rootToken := vault.TestCoreUnsealedWithConfig(t, coreConfig)
	ln, addr := vhttp.TestServer(t, core)

	t.Cleanup(func() {
		err := ln.Close()
		if err != nil {
			t.Errorf("could not close net listener")
		}
		err = core.Shutdown()
		if err != nil {
			t.Errorf("could not close vault core backend")
		}
	})

	return rootToken, addr
}

func mockVaultwithPKI(t *testing.T, cm *CertManager) {
	err := cm.client.Sys().Mount("pki", &api.MountInput{
		Type: "pki",
		Config: api.MountConfigInput{
			DefaultLeaseTTL: "24h",
			MaxLeaseTTL:     "48h",
		},
	})
	if err != nil {
		t.Fatalf("could not mount PKI secrets backend, %v", err)
	}

	resp, err := cm.client.Logical().Write("pki/root/generate/internal", map[string]interface{}{
		"common_name": "localhost",
	})
	if err != nil {
		t.Fatalf("could not generate ca info, %v", err)
	}
	if resp == nil {
		t.Fatal("expected ca info, response from vault was nil")
	}

	_, err = cm.client.Logical().Write("pki/roles/default_role", map[string]interface{}{})

	req := cm.client.NewRequest("POST", "/v1/pki_int/roles/default_role")
	req.BodyBytes = []byte(`{
		"max_ttl": "48h"
	  }`)

	if err != nil {
		t.Fatalf("could not create default_role in vault, %v", err)
	}
}
