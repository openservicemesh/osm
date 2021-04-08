package ingress

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"

	"github.com/pkg/errors"
)

type fakeDiscoveryClient struct {
	discovery.ServerResourcesInterface
	resources map[string]metav1.APIResourceList
	err       error
}

// ServerResourcesForGroupVersion returns the supported resources for a group and version.
func (f *fakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	resp := f.resources[groupVersion]
	return &resp, f.err
}

func TestGetSupportedIngressVersions(t *testing.T) {
	assert := tassert.New(t)

	type testCase struct {
		name             string
		discoveryClient  discovery.ServerResourcesInterface
		expectedVersions map[string]bool
		exepectError     bool
	}

	testCases := []testCase{
		{
			name: "k8s API server supports both ingress v1 and v1beta",
			discoveryClient: &fakeDiscoveryClient{
				resources: map[string]metav1.APIResourceList{
					"networking.k8s.io/v1": {APIResources: []metav1.APIResource{
						{Kind: "Ingress"},
						{Kind: "NetworkPolicy"},
					}},
					"networking.k8s.io/v1beta1": {APIResources: []metav1.APIResource{
						{Kind: "Ingress"},
					}},
				},
				err: nil,
			},
			expectedVersions: map[string]bool{
				"networking.k8s.io/v1":      true,
				"networking.k8s.io/v1beta1": true,
			},
			exepectError: false,
		},
		{
			name: "k8s API server supports only ingress v1beta1",
			discoveryClient: &fakeDiscoveryClient{
				resources: map[string]metav1.APIResourceList{
					"networking.k8s.io/v1": {APIResources: []metav1.APIResource{
						{Kind: "NetworkPolicy"},
					}},
					"networking.k8s.io/v1beta1": {APIResources: []metav1.APIResource{
						{Kind: "Ingress"},
					}},
				},
				err: nil,
			},
			expectedVersions: map[string]bool{
				"networking.k8s.io/v1":      false,
				"networking.k8s.io/v1beta1": true,
			},
			exepectError: false,
		},
		{
			name: "k8s API server supports only ingress v1",
			discoveryClient: &fakeDiscoveryClient{
				resources: map[string]metav1.APIResourceList{
					"networking.k8s.io/v1": {APIResources: []metav1.APIResource{
						{Kind: "Ingress"},
					}},
					"networking.k8s.io/v1beta1": {APIResources: []metav1.APIResource{
						{Kind: "NetworkPolicy"},
					}},
				},
				err: nil,
			},
			expectedVersions: map[string]bool{
				"networking.k8s.io/v1":      true,
				"networking.k8s.io/v1beta1": false,
			},
			exepectError: false,
		},
		{
			name: "k8s API server returns an error",
			discoveryClient: &fakeDiscoveryClient{
				resources: map[string]metav1.APIResourceList{},
				err:       errors.New("fake error"),
			},
			expectedVersions: nil,
			exepectError:     true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			versions, err := getSupportedIngressVersions(tc.discoveryClient)

			assert.Equal(tc.exepectError, err != nil)
			assert.Equal(tc.expectedVersions, versions)
		})
	}
}
