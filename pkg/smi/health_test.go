package smi

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"

	"github.com/openservicemesh/osm/pkg/k8s"
)

func TestRequiredAPIResourcesExist(t *testing.T) {
	type testCase struct {
		name            string
		discoveryClient discovery.ServerResourcesInterface
		expect          bool
	}

	testCases := []testCase{
		{
			name: "all SMI API resources exist",
			discoveryClient: &k8s.FakeDiscoveryClient{
				Resources: map[string]metav1.APIResourceList{
					"specs.smi-spec.io/v1alpha4": {APIResources: []metav1.APIResource{
						{Kind: "HTTPRouteGroup"},
						{Kind: "TCPRoute"},
					}},
					"access.smi-spec.io/v1alpha3": {APIResources: []metav1.APIResource{
						{Kind: "TrafficTarget"},
					}},
					"split.smi-spec.io/v1alpha2": {APIResources: []metav1.APIResource{
						{Kind: "TrafficSplit"},
					}},
				},
				Err: nil,
			},
			expect: true,
		},
		{
			name: "TrafficSplit does not exist",
			discoveryClient: &k8s.FakeDiscoveryClient{
				Resources: map[string]metav1.APIResourceList{
					"specs.smi-spec.io/v1alpha4": {APIResources: []metav1.APIResource{
						{Kind: "HTTPRouteGroup"},
						{Kind: "TCPRoute"},
					}},
					"access.smi-spec.io/v1alpha3": {APIResources: []metav1.APIResource{
						{Kind: "TrafficTarget"},
					}},
					"split.smi-spec.io/v1alpha2": {APIResources: []metav1.APIResource{}},
				},
				Err: nil,
			},
			expect: false,
		},
		{
			name: "HTTPRouteGroup does not exist",
			discoveryClient: &k8s.FakeDiscoveryClient{
				Resources: map[string]metav1.APIResourceList{
					"specs.smi-spec.io/v1alpha4": {APIResources: []metav1.APIResource{
						{Kind: "TCPRoute"},
					}},
					"access.smi-spec.io/v1alpha3": {APIResources: []metav1.APIResource{
						{Kind: "TrafficTarget"},
					}},
					"split.smi-spec.io/v1alpha2": {APIResources: []metav1.APIResource{
						{Kind: "TrafficSplit"},
					}},
				},
				Err: nil,
			},
			expect: false,
		},
		{
			name: "TCPRoute does not exist",
			discoveryClient: &k8s.FakeDiscoveryClient{
				Resources: map[string]metav1.APIResourceList{
					"specs.smi-spec.io/v1alpha4": {APIResources: []metav1.APIResource{
						{Kind: "HTTPRouteGroup"},
					}},
					"access.smi-spec.io/v1alpha3": {APIResources: []metav1.APIResource{
						{Kind: "TrafficTarget"},
					}},
					"split.smi-spec.io/v1alpha2": {APIResources: []metav1.APIResource{
						{Kind: "TrafficSplit"},
					}},
				},
				Err: nil,
			},
			expect: false,
		},
		{
			name: "TrafficTarget does not exist",
			discoveryClient: &k8s.FakeDiscoveryClient{
				Resources: map[string]metav1.APIResourceList{
					"specs.smi-spec.io/v1alpha4": {APIResources: []metav1.APIResource{
						{Kind: "HTTPRouteGroup"},
						{Kind: "TCPRoute"},
					}},
					"access.smi-spec.io/v1alpha3": {APIResources: []metav1.APIResource{}},
					"split.smi-spec.io/v1alpha2": {APIResources: []metav1.APIResource{
						{Kind: "TrafficSplit"},
					}},
				},
				Err: nil,
			},
			expect: false,
		},
		{
			name: "specs API group does not exit",
			discoveryClient: &k8s.FakeDiscoveryClient{
				Resources: map[string]metav1.APIResourceList{
					"access.smi-spec.io/v1alpha3": {APIResources: []metav1.APIResource{
						{Kind: "TrafficTarget"},
					}},
					"split.smi-spec.io/v1alpha2": {APIResources: []metav1.APIResource{
						{Kind: "TrafficSplit"},
					}},
				},
				Err: nil,
			},
			expect: false,
		},
		{
			name: "access API group does not exist",
			discoveryClient: &k8s.FakeDiscoveryClient{
				Resources: map[string]metav1.APIResourceList{
					"specs.smi-spec.io/v1alpha4": {APIResources: []metav1.APIResource{
						{Kind: "HTTPRouteGroup"},
						{Kind: "TCPRoute"},
					}},
					"split.smi-spec.io/v1alpha2": {APIResources: []metav1.APIResource{
						{Kind: "TrafficSplit"},
					}},
				},
				Err: nil,
			},
			expect: false,
		},
		{
			name: "split API group does not exist",
			discoveryClient: &k8s.FakeDiscoveryClient{
				Resources: map[string]metav1.APIResourceList{
					"specs.smi-spec.io/v1alpha4": {APIResources: []metav1.APIResource{
						{Kind: "HTTPRouteGroup"},
						{Kind: "TCPRoute"},
					}},
					"access.smi-spec.io/v1alpha3": {APIResources: []metav1.APIResource{
						{Kind: "TrafficTarget"},
					}},
				},
				Err: nil,
			},
			expect: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			c := HealthChecker{DiscoveryClient: tc.discoveryClient}
			actual := c.requiredAPIResourcesExist()
			assert.Equal(tc.expect, actual)
		})
	}
}

func TestGetID(t *testing.T) {
	assert := tassert.New(t)

	discoveryClient := &k8s.FakeDiscoveryClient{
		Resources: map[string]metav1.APIResourceList{
			"specs.smi-spec.io/v1alpha4": {APIResources: []metav1.APIResource{
				{Kind: "HTTPRouteGroup"},
				{Kind: "TCPRoute"},
			}},
			"access.smi-spec.io/v1alpha3": {APIResources: []metav1.APIResource{
				{Kind: "TrafficTarget"},
			}},
			"split.smi-spec.io/v1alpha2": {APIResources: []metav1.APIResource{
				{Kind: "TrafficSplit"},
			}},
		},
		Err: nil,
	}
	c := HealthChecker{DiscoveryClient: discoveryClient}
	actual := c.GetID()
	assert.Equal("SMI", actual)
}

func TestLiveness(t *testing.T) {
	assert := tassert.New(t)

	discoveryClient := &k8s.FakeDiscoveryClient{
		Resources: map[string]metav1.APIResourceList{
			"specs.smi-spec.io/v1alpha4": {APIResources: []metav1.APIResource{
				{Kind: "HTTPRouteGroup"},
				{Kind: "TCPRoute"},
			}},
			"access.smi-spec.io/v1alpha3": {APIResources: []metav1.APIResource{
				{Kind: "TrafficTarget"},
			}},
			"split.smi-spec.io/v1alpha2": {APIResources: []metav1.APIResource{
				{Kind: "TrafficSplit"},
			}},
		},
		Err: nil,
	}
	c := HealthChecker{DiscoveryClient: discoveryClient}
	actual := c.Liveness()
	assert.True(actual)
}

func TestReadiness(t *testing.T) {
	assert := tassert.New(t)

	discoveryClient := &k8s.FakeDiscoveryClient{
		Resources: map[string]metav1.APIResourceList{
			"specs.smi-spec.io/v1alpha4": {APIResources: []metav1.APIResource{
				{Kind: "HTTPRouteGroup"},
				{Kind: "TCPRoute"},
			}},
			"access.smi-spec.io/v1alpha3": {APIResources: []metav1.APIResource{
				{Kind: "TrafficTarget"},
			}},
			"split.smi-spec.io/v1alpha2": {APIResources: []metav1.APIResource{
				{Kind: "TrafficSplit"},
			}},
		},
		Err: nil,
	}
	c := HealthChecker{DiscoveryClient: discoveryClient}
	actual := c.Readiness()
	assert.True(actual)
}
