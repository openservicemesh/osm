package catalog

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestGetRetryPolicy(t *testing.T) {
	var thresholdUintVal uint32 = 3
	thresholdTimeoutDuration := metav1.Duration{Duration: time.Duration(5 * time.Second)}
	thresholdBackoffDuration := metav1.Duration{Duration: time.Duration(1 * time.Second)}

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
	retrySrc := identity.ServiceIdentity("sa1.ns")

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
							PerTryTimeout:            &thresholdTimeoutDuration,
							NumRetries:               &thresholdUintVal,
							RetryBackoffBaseInterval: &thresholdBackoffDuration,
						},
					},
				},
			},
			destSvc: service.MeshService{Name: "s1", Namespace: "b"},
			expectedRetryPolicy: &policyV1alpha1.RetryPolicySpec{
				RetryOn:                  "5xx",
				PerTryTimeout:            &thresholdTimeoutDuration,
				NumRetries:               &thresholdUintVal,
				RetryBackoffBaseInterval: &thresholdBackoffDuration,
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
							PerTryTimeout:            &thresholdTimeoutDuration,
							NumRetries:               &thresholdUintVal,
							RetryBackoffBaseInterval: &thresholdBackoffDuration,
						},
					},
				},
			},
			destSvc: service.MeshService{Name: "s1", Namespace: "b"},
			expectedRetryPolicy: &policyV1alpha1.RetryPolicySpec{
				RetryOn:                  "5xx",
				PerTryTimeout:            &thresholdTimeoutDuration,
				NumRetries:               &thresholdUintVal,
				RetryBackoffBaseInterval: &thresholdBackoffDuration,
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
							PerTryTimeout:            &thresholdTimeoutDuration,
							NumRetries:               &thresholdUintVal,
							RetryBackoffBaseInterval: &thresholdBackoffDuration,
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
							PerTryTimeout:            &thresholdTimeoutDuration,
							NumRetries:               &thresholdUintVal,
							RetryBackoffBaseInterval: &thresholdBackoffDuration,
						},
					},
				},
			},
			destSvc: service.MeshService{Name: "s1", Namespace: "b"},
			expectedRetryPolicy: &policyV1alpha1.RetryPolicySpec{
				RetryOn:                  "5xx",
				PerTryTimeout:            &thresholdTimeoutDuration,
				NumRetries:               &thresholdUintVal,
				RetryBackoffBaseInterval: &thresholdBackoffDuration,
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
							PerTryTimeout:            &thresholdTimeoutDuration,
							NumRetries:               &thresholdUintVal,
							RetryBackoffBaseInterval: &thresholdBackoffDuration,
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

			mockCfg.EXPECT().GetFeatureFlags().Return(v1alpha2.FeatureFlags{EnableRetryPolicy: tc.retryPolicyFlag}).Times(1)
			mockPolicyController.EXPECT().ListRetryPolicies(gomock.Any()).Return(tc.retryCRDs).Times(1)

			res := mc.getRetryPolicy(retrySrc, tc.destSvc)
			assert.Equal(tc.expectedRetryPolicy, res)
		})
	}
}
