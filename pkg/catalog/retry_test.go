package catalog

import (
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestGetRetryPolicy(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockServiceProvider := service.NewMockProvider(mockCtrl)
	mockEndpointsProvider := endpoint.NewMockProvider(mockCtrl)
	mockCfg := configurator.NewMockConfigurator(mockCtrl)
	mockPolicyController := policy.NewMockController(mockCtrl)
	mockKubeController := k8s.NewMockController(mockCtrl)
	mc := &MeshCatalog{
		serviceProviders:   []service.Provider{mockServiceProvider},
		endpointsProviders: []endpoint.Provider{mockEndpointsProvider},
		configurator:       mockCfg,
		policyController:   mockPolicyController,
		kubeController:     mockKubeController,
	}
	retrySrc := identity.ServiceIdentity("sa1.ns.cluster.local")

	testcases := []struct {
		name                string
		retryPolicyFlag     bool
		retryCRDs           []*policyV1alpha1.Retry
		destSvc             service.MeshService
		expectedRetryPolicy *policyV1alpha1.RetryPolicySpec
	}{
		{
			name:                "No retry policies",
			retryPolicyFlag:     true,
			retryCRDs:           nil,
			expectedRetryPolicy: nil,
		},
		{
			name:            "One retry policy for service",
			retryPolicyFlag: true,
			retryCRDs: []*policyV1alpha1.Retry{
				{
					Spec: policyV1alpha1.RetrySpec{
						Source: policyV1alpha1.RetrySrcDstSpec{
							Kind:      "Service",
							Name:      "sa1",
							Namespace: "ns",
						},
						Destinations: []policyV1alpha1.RetrySrcDstSpec{
							{
								Kind:      "Service",
								Name:      "s1",
								Namespace: "b",
							},
						},
						RetryPolicy: policyV1alpha1.RetryPolicySpec{
							RetryOn:                  "5xx",
							PerTryTimeout:            "2ns",
							NumRetries:               3,
							RetryBackoffBaseInterval: "4s",
						},
					},
				},
			},
			destSvc: service.MeshService{Name: "s1", Namespace: "b"},
			expectedRetryPolicy: &policyV1alpha1.RetryPolicySpec{
				RetryOn:                  "5xx",
				PerTryTimeout:            "2ns",
				NumRetries:               3,
				RetryBackoffBaseInterval: "4s",
			},
		},
		{
			name:            "Retry policy with multiple destinations",
			retryPolicyFlag: true,
			retryCRDs: []*policyV1alpha1.Retry{
				{
					Spec: policyV1alpha1.RetrySpec{
						Source: policyV1alpha1.RetrySrcDstSpec{
							Kind:      "Service",
							Name:      "sa1",
							Namespace: "ns",
						},
						Destinations: []policyV1alpha1.RetrySrcDstSpec{
							{
								Kind:      "Service",
								Name:      "s1",
								Namespace: "b",
							},
							{
								Kind:      "Service",
								Name:      "c",
								Namespace: "b",
							},
							{
								Kind:      "Service",
								Name:      "c",
								Namespace: "d",
							},
						},
						RetryPolicy: policyV1alpha1.RetryPolicySpec{
							RetryOn:                  "5xx",
							PerTryTimeout:            "2ns",
							NumRetries:               3,
							RetryBackoffBaseInterval: "4s",
						},
					},
				},
			},
			destSvc: service.MeshService{Name: "s1", Namespace: "b"},
			expectedRetryPolicy: &policyV1alpha1.RetryPolicySpec{
				RetryOn:                  "5xx",
				PerTryTimeout:            "2ns",
				NumRetries:               3,
				RetryBackoffBaseInterval: "4s",
			},
		},
		{
			name:            "Multiple retry policies for same src and dest",
			retryPolicyFlag: true,
			retryCRDs: []*policyV1alpha1.Retry{
				{
					Spec: policyV1alpha1.RetrySpec{
						Source: policyV1alpha1.RetrySrcDstSpec{
							Kind:      "Service",
							Name:      "sa1",
							Namespace: "ns",
						},
						Destinations: []policyV1alpha1.RetrySrcDstSpec{
							{
								Kind:      "Service",
								Name:      "s1",
								Namespace: "b",
							},
						},
						RetryPolicy: policyV1alpha1.RetryPolicySpec{
							RetryOn:                  "5xx",
							PerTryTimeout:            "2ns",
							NumRetries:               3,
							RetryBackoffBaseInterval: "4s",
						},
					},
				},
				{
					Spec: policyV1alpha1.RetrySpec{
						Source: policyV1alpha1.RetrySrcDstSpec{
							Kind:      "Service",
							Name:      "sa1",
							Namespace: "ns",
						},
						Destinations: []policyV1alpha1.RetrySrcDstSpec{
							{
								Kind:      "Service",
								Name:      "s1",
								Namespace: "b",
							},
						},
						RetryPolicy: policyV1alpha1.RetryPolicySpec{
							RetryOn:                  "4xx",
							PerTryTimeout:            "4s",
							NumRetries:               6,
							RetryBackoffBaseInterval: "7us",
						},
					},
				},
			},
			destSvc: service.MeshService{Name: "s1", Namespace: "b"},
			expectedRetryPolicy: &policyV1alpha1.RetryPolicySpec{
				RetryOn:                  "5xx",
				PerTryTimeout:            "2ns",
				NumRetries:               3,
				RetryBackoffBaseInterval: "4s",
			},
		},
		{
			name:            "No Retry policy for destination",
			retryPolicyFlag: true,
			retryCRDs: []*policyV1alpha1.Retry{
				{
					Spec: policyV1alpha1.RetrySpec{
						Source: policyV1alpha1.RetrySrcDstSpec{
							Kind:      "Service",
							Name:      "sa1",
							Namespace: "ns",
						},
						Destinations: []policyV1alpha1.RetrySrcDstSpec{
							{
								Kind:      "Service",
								Name:      "s12",
								Namespace: "b",
							},
							{
								Kind:      "Service",
								Name:      "c",
								Namespace: "b",
							},
							{
								Kind:      "Service",
								Name:      "c",
								Namespace: "d",
							},
						},
						RetryPolicy: policyV1alpha1.RetryPolicySpec{
							RetryOn:                  "5xx",
							PerTryTimeout:            "2ns",
							NumRetries:               3,
							RetryBackoffBaseInterval: "4s",
						},
					},
				},
			},
			destSvc:             service.MeshService{Name: "s1", Namespace: "b"},
			expectedRetryPolicy: nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			mockCfg.EXPECT().GetFeatureFlags().Return(v1alpha3.FeatureFlags{EnableRetryPolicy: tc.retryPolicyFlag}).Times(1)
			mockPolicyController.EXPECT().ListRetryPolicies(gomock.Any()).Return(tc.retryCRDs).Times(1)

			res := mc.getRetryPolicy(retrySrc, tc.destSvc)
			assert.Equal(tc.expectedRetryPolicy, res)
		})
	}
}
