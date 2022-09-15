package catalog

import (
	"net"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/endpoint"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestGetIngressTrafficPolicy(t *testing.T) {
	// Common test variables
	ingressSourceSvc := service.MeshService{Name: "ingressGateway", Namespace: "IngressGatewayNs"}
	ingressBackendSvcEndpoints := []endpoint.Endpoint{
		{IP: net.ParseIP("10.0.0.10"), Port: 80},
		{IP: net.ParseIP("10.0.0.10"), Port: 90},
	}
	sourceSvcWithoutEndpoints := service.MeshService{Name: "unknown", Namespace: "IngressGatewayNs"}

	testCases := []struct {
		name                        string
		ingressBackendPolicyEnabled bool
		enableHTTPSIngress          bool
		meshSvc                     service.MeshService
		ingressBackend              *policyV1alpha1.IngressBackend
		expectedHTTPRoutePolicies   []*trafficpolicy.InboundTrafficPolicy
		expectedTrafficMatches      []*trafficpolicy.IngressTrafficMatch
		expectError                 bool
	}{
		{
			name:                        "No ingress routes",
			ingressBackendPolicyEnabled: false,
			enableHTTPSIngress:          false,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", Protocol: "http", TargetPort: 80},
			ingressBackend:              nil,
			expectedTrafficMatches:      nil,
			expectedHTTPRoutePolicies:   nil,
			expectError:                 false,
		},
		{
			name:                        "HTTP ingress using the IngressBackend API",
			ingressBackendPolicyEnabled: true,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", Protocol: "http", TargetPort: 80},
			ingressBackend: &policyV1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "testns",
				},
				Spec: policyV1alpha1.IngressBackendSpec{
					Backends: []policyV1alpha1.BackendSpec{
						{
							Name: "foo",
							Port: policyV1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyV1alpha1.IngressSourceSpec{
						{
							Kind:      policyV1alpha1.KindService,
							Name:      ingressSourceSvc.Name,
							Namespace: ingressSourceSvc.Namespace,
						},
					},
				},
			},
			expectedHTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "testns/foo_from_ingress-backend-1",
					Hostnames: []string{
						"*",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo|80|local",
									Weight:      100,
								}),
							},
							AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
						},
					},
				},
			},
			expectedTrafficMatches: []*trafficpolicy.IngressTrafficMatch{
				{
					Name:           "ingress_testns/foo_80_http",
					Protocol:       "http",
					Port:           80,
					SourceIPRanges: []string{"10.0.0.10/32"}, // Endpoint of 'ingressSourceSvc' referenced as a source
				},
			},
			expectError: false,
		},
		{
			name:                        "HTTPS ingress with mTLS using the IngressBackend API",
			ingressBackendPolicyEnabled: true,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", Protocol: "http", TargetPort: 80},
			ingressBackend: &policyV1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "testns",
				},
				Spec: policyV1alpha1.IngressBackendSpec{
					Backends: []policyV1alpha1.BackendSpec{
						{
							Name: "foo",
							Port: policyV1alpha1.PortSpec{
								Number:   80,
								Protocol: "https",
							},
							TLS: policyV1alpha1.TLSSpec{
								SkipClientCertValidation: false,
								SNIHosts:                 []string{"foo.org"},
							},
						},
					},
					Sources: []policyV1alpha1.IngressSourceSpec{
						{
							Kind:      policyV1alpha1.KindService,
							Name:      ingressSourceSvc.Name,
							Namespace: ingressSourceSvc.Namespace,
						},
						{
							Kind: policyV1alpha1.KindAuthenticatedPrincipal,
							Name: "ingressGw.ingressGwNs.cluster.local",
						},
					},
				},
			},
			expectedHTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "testns/foo_from_ingress-backend-1",
					Hostnames: []string{
						"*",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo|80|local",
									Weight:      100,
								}),
							},
							AllowedPrincipals: mapset.NewSet("ingressGw.ingressGwNs.cluster.local"),
						},
					},
				},
			},
			expectedTrafficMatches: []*trafficpolicy.IngressTrafficMatch{
				{
					Name:                     "ingress_testns/foo_80_https",
					Protocol:                 "https",
					Port:                     80,
					SourceIPRanges:           []string{"10.0.0.10/32"}, // Endpoint of 'ingressSourceSvc' referenced as a source
					SkipClientCertValidation: false,
					ServerNames:              []string{"foo.org"},
				},
			},
			expectError: false,
		},
		{
			name:                        "HTTPS ingress with TLS using the IngressBackend API",
			ingressBackendPolicyEnabled: true,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", Protocol: "http", TargetPort: 80},
			ingressBackend: &policyV1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "testns",
				},
				Spec: policyV1alpha1.IngressBackendSpec{
					Backends: []policyV1alpha1.BackendSpec{
						{
							Name: "foo",
							Port: policyV1alpha1.PortSpec{
								Number:   80,
								Protocol: "https",
							},
							TLS: policyV1alpha1.TLSSpec{
								SkipClientCertValidation: true,
							},
						},
					},
					Sources: []policyV1alpha1.IngressSourceSpec{
						{
							Kind:      policyV1alpha1.KindService,
							Name:      ingressSourceSvc.Name,
							Namespace: ingressSourceSvc.Namespace,
						},
						{
							Kind: policyV1alpha1.KindAuthenticatedPrincipal,
							Name: "ingressGw.ingressGwNs.cluster.local",
						},
					},
				},
			},
			expectedHTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "testns/foo_from_ingress-backend-1",
					Hostnames: []string{
						"*",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo|80|local",
									Weight:      100,
								}),
							},
							AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
						},
					},
				},
			},
			expectedTrafficMatches: []*trafficpolicy.IngressTrafficMatch{
				{
					Name:                     "ingress_testns/foo_80_https",
					Protocol:                 "https",
					Port:                     80,
					SourceIPRanges:           []string{"10.0.0.10/32"}, // Endpoint of 'ingressSourceSvc' referenced as a source
					SkipClientCertValidation: true,
				},
			},
			expectError: false,
		},
		{
			name:                        "Specifying a source service without endpoints in an IngressBackend should error for trafficMatches",
			ingressBackendPolicyEnabled: true,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", Protocol: "http", TargetPort: 80},
			ingressBackend: &policyV1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "testns",
				},
				Spec: policyV1alpha1.IngressBackendSpec{
					Backends: []policyV1alpha1.BackendSpec{
						{
							Name: "foo",
							Port: policyV1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyV1alpha1.IngressSourceSpec{
						{
							Kind:      policyV1alpha1.KindService,
							Name:      sourceSvcWithoutEndpoints.Name, // This service does not exist, so it's endpoints won't as well
							Namespace: sourceSvcWithoutEndpoints.Namespace,
						},
					},
				},
			},
			expectedTrafficMatches: nil,
			expectedHTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "testns/foo_from_ingress-backend-1",
					Hostnames: []string{
						"*",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo|80|local",
									Weight:      100,
								}),
							},
							AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
						},
					},
				},
			},
			expectError: true,
		},
		{
			name:                        "HTTP ingress with IPRange as a source using the IngressBackend API",
			ingressBackendPolicyEnabled: true,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", Protocol: "http", TargetPort: 80},
			ingressBackend: &policyV1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "testns",
				},
				Spec: policyV1alpha1.IngressBackendSpec{
					Backends: []policyV1alpha1.BackendSpec{
						{
							Name: "foo",
							Port: policyV1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyV1alpha1.IngressSourceSpec{
						{
							Kind: policyV1alpha1.KindIPRange,
							Name: "10.0.0.0/10",
						},
						{
							Kind: policyV1alpha1.KindIPRange,
							Name: "20.0.0.0/10",
						},
						{
							Kind: policyV1alpha1.KindIPRange,
							Name: "invalid", // should be ignored
						},
					},
				},
			},
			expectedHTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "testns/foo_from_ingress-backend-1",
					Hostnames: []string{
						"*",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo|80|local",
									Weight:      100,
								}),
							},
							AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
						},
					},
				},
			},
			expectedTrafficMatches: []*trafficpolicy.IngressTrafficMatch{
				{
					Name:           "ingress_testns/foo_80_http",
					Protocol:       "http",
					Port:           80,
					SourceIPRanges: []string{"10.0.0.0/10", "20.0.0.0/10"}, // 'IPRange' referenced as a source
				},
			},
			expectError: false,
		},
		{
			name:                        "MeshService.TargetPort does not match ingress backend port",
			ingressBackendPolicyEnabled: true,
			// meshSvc.TargetPort does not match ingressBackend.Spec.Backends[].Port.Number
			meshSvc: service.MeshService{Name: "foo", Namespace: "testns", Protocol: "http", TargetPort: 90},
			ingressBackend: &policyV1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "testns",
				},
				Spec: policyV1alpha1.IngressBackendSpec{
					Backends: []policyV1alpha1.BackendSpec{
						{
							Name: "foo",
							Port: policyV1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyV1alpha1.IngressSourceSpec{
						{
							Kind: policyV1alpha1.KindIPRange,
							Name: "10.0.0.0/10",
						},
						{
							Kind: policyV1alpha1.KindIPRange,
							Name: "20.0.0.0/10",
						},
						{
							Kind: policyV1alpha1.KindIPRange,
							Name: "invalid", // should be ignored
						},
					},
				},
			},
			expectedTrafficMatches: nil,
			expectedHTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "testns/foo_from_ingress-backend-1",
					Hostnames: []string{
						"*",
					},
					Rules: nil,
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockProvider := compute.NewMockInterface(mockCtrl)

			meshCatalog := &MeshCatalog{
				Interface: mockProvider,
			}

			// Note: if AnyTimes() is used with a mock function, it implies the function may or may not be called
			// depending on the test case.
			mockProvider.EXPECT().GetIngressBackendPolicyForService(tc.meshSvc).Return(tc.ingressBackend).AnyTimes()
			mockProvider.EXPECT().ListEndpointsForService(ingressSourceSvc).Return(ingressBackendSvcEndpoints).AnyTimes()
			mockProvider.EXPECT().ListEndpointsForService(sourceSvcWithoutEndpoints).Return(nil).AnyTimes()
			mockProvider.EXPECT().UpdateIngressBackendStatus(gomock.Any()).Return(nil, nil).AnyTimes()

			actualHTTPRoutePolicies := meshCatalog.GetIngressHTTPRoutePoliciesForSvc(tc.meshSvc)
			actualTrafficMatches, err := meshCatalog.GetIngressTrafficMatchesForSvc(tc.meshSvc)
			assert.Equal(tc.expectError, err != nil)

			assert.Equal(tc.expectedHTTPRoutePolicies, actualHTTPRoutePolicies)
			assert.Equal(tc.expectedTrafficMatches, actualTrafficMatches)
		})
	}
}
