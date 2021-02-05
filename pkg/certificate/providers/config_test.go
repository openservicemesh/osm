package providers

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/configurator"
)

func TestGetCertificateManager(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().IsDebugServerEnabled().Return(false).AnyTimes()

	testCases := []struct {
		name string
		util *Config

		expectError bool
	}{
		{
			name: "tresor as the certificate manager",
			util: &Config{
				caBundleSecretName: "osm-ca-bundle",
				providerKind:       TresorKind,
				providerNamespace:  "osm-system",
				cfg:                mockConfigurator,
				kubeClient:         fake.NewSimpleClientset(),
			},
			expectError: false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			manager, _, err := tc.util.getCertificateManager()
			assert.NotNil(manager)
			assert.Equal(tc.expectError, err != nil)
		})
	}
}
