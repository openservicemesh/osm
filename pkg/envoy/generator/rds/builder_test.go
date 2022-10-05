package rds

import (
	"testing"

	mapset "github.com/deckarep/golang-set"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	tassert "github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/models"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestBuildInboundMeshRouteConfiguration(t *testing.T) {
	testCases := []struct {
		name                               string
		InboundMeshHTTPRouteConfigsPerPort map[int][]*trafficpolicy.InboundTrafficPolicy
		expectedRouteConfigFields          *xds_route.RouteConfiguration
	}{
		{
			name:                               "no route configs",
			InboundMeshHTTPRouteConfigsPerPort: nil,
			expectedRouteConfigFields:          nil,
		},
		{
			name: "basic inbound policy ",
			InboundMeshHTTPRouteConfigsPerPort: map[int][]*trafficpolicy.InboundTrafficPolicy{
				80: {
					{
						Name:      "bookstore-v1-default",
						Hostnames: []string{"bookstore-v1.default.svc.cluster.local"},
						Rules: []*trafficpolicy.Rule{
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
									WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
								},
								AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
							},
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
									WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
								},
								AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
							},
						},
					},
					{
						Name:      "bookstore-v2-default",
						Hostnames: []string{"bookstore-v2.default.svc.cluster.local"},
						Rules: []*trafficpolicy.Rule{
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
									WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
									RetryPolicy:      nil,
								},
								AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
							},
						},
					},
				},
			},
			expectedRouteConfigFields: &xds_route.RouteConfiguration{
				Name: "rds-inbound.80",
				VirtualHosts: []*xds_route.VirtualHost{
					{
						Name: "inbound_virtual-host|bookstore-v1.default.svc.cluster.local",
						Routes: []*xds_route.Route{
							{
								// corresponds to Rules[0]

								// Only the filter name is matched, not the marshalled config
								TypedPerFilterConfig: map[string]*any.Any{
									envoy.HTTPRBACFilterName: nil,
								},
							},
							{
								// corresponds to Rules[1]

								// Only the filter name is matched, not the marshalled config
								TypedPerFilterConfig: map[string]*any.Any{
									envoy.HTTPRBACFilterName: nil,
								},
							},
						},
					},
					{
						Name: "inbound_virtual-host|bookstore-v2.default.svc.cluster.local",
						Routes: []*xds_route.Route{
							{
								// corresponds to ingressPolicies[1].Rules[0]
							},
						},
					},
				},
			},
		},
		{
			name: "inbound policy with VirtualHost and Route level local rate limiting",
			InboundMeshHTTPRouteConfigsPerPort: map[int][]*trafficpolicy.InboundTrafficPolicy{
				80: {
					{
						Name:      "bookstore-v1-default",
						Hostnames: []string{"bookstore-v1.default.svc.cluster.local"},
						Rules: []*trafficpolicy.Rule{
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
									WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
									RateLimit: &policyv1alpha1.HTTPPerRouteRateLimitSpec{
										Local: &policyv1alpha1.HTTPLocalRateLimitSpec{
											Requests: 10,
											Unit:     "second",
										},
									},
								},
								AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
							},
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
									WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
									RateLimit: &policyv1alpha1.HTTPPerRouteRateLimitSpec{
										Local: &policyv1alpha1.HTTPLocalRateLimitSpec{
											Requests: 10,
											Unit:     "second",
										},
									},
								},
								AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
							},
						},
						RateLimit: &policyv1alpha1.RateLimitSpec{
							Local: &policyv1alpha1.LocalRateLimitSpec{
								HTTP: &policyv1alpha1.HTTPLocalRateLimitSpec{
									Requests: 100,
									Unit:     "minute",
								},
							},
						},
					},
				},
			},
			expectedRouteConfigFields: &xds_route.RouteConfiguration{
				Name: "rds-inbound.80",
				VirtualHosts: []*xds_route.VirtualHost{
					{
						Name: "inbound_virtual-host|bookstore-v1.default.svc.cluster.local",
						Routes: []*xds_route.Route{
							{
								// corresponds to ingressPolicies[0].Rules[0]

								// Only the filter name is matched, not the marshalled config
								TypedPerFilterConfig: map[string]*any.Any{
									envoy.HTTPRBACFilterName:           nil,
									envoy.HTTPLocalRateLimitFilterName: nil,
								},
							},
							{
								// corresponds to ingressPolicies[0].Rules[1]

								// Only the filter name is matched, not the marshalled config
								TypedPerFilterConfig: map[string]*any.Any{
									envoy.HTTPRBACFilterName:           nil,
									envoy.HTTPLocalRateLimitFilterName: nil,
								},
							},
						},
						// Only the filter name is matched, not the marshalled config
						TypedPerFilterConfig: map[string]*any.Any{
							envoy.HTTPLocalRateLimitFilterName: nil,
						},
					},
				},
			},
		},
		{
			name: "inbound policy with VirtualHost and Route level global rate limiting",
			InboundMeshHTTPRouteConfigsPerPort: map[int][]*trafficpolicy.InboundTrafficPolicy{
				80: {
					{
						Name:      "bookstore-v1-default",
						Hostnames: []string{"bookstore-v1.default.svc.cluster.local"},
						Rules: []*trafficpolicy.Rule{
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
									WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
									RateLimit: &policyv1alpha1.HTTPPerRouteRateLimitSpec{
										Global: &policyv1alpha1.HTTPGlobalPerRouteRateLimitSpec{
											Descriptors: []policyv1alpha1.HTTPGlobalRateLimitDescriptor{
												{
													Entries: []policyv1alpha1.HTTPGlobalRateLimitDescriptorEntry{
														{
															GenericKey: &policyv1alpha1.GenericKeyDescriptorEntry{},
														},
														{
															RemoteAddress: &policyv1alpha1.RemoteAddressDescriptorEntry{},
														},
													},
												},
												{
													Entries: []policyv1alpha1.HTTPGlobalRateLimitDescriptorEntry{
														{
															RequestHeader: &policyv1alpha1.RequestHeaderDescriptorEntry{},
														},
													},
												},
											},
										},
									},
								},
								AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
							},
						},
						RateLimit: &policyv1alpha1.RateLimitSpec{
							Global: &policyv1alpha1.GlobalRateLimitSpec{
								HTTP: &policyv1alpha1.HTTPGlobalRateLimitSpec{
									RateLimitService: policyv1alpha1.RateLimitServiceSpec{
										Host: "foo.bar",
										Port: 8080,
									},
									Descriptors: []policyv1alpha1.HTTPGlobalRateLimitDescriptor{
										{
											Entries: []policyv1alpha1.HTTPGlobalRateLimitDescriptorEntry{
												{
													HeaderValueMatch: &policyv1alpha1.HeaderValueMatchDescriptorEntry{
														Headers: []policyv1alpha1.HTTPHeaderMatcher{
															{
																Exact: "e",
															},
															{
																Prefix: "p",
															},
															{
																Suffix: "s",
															},
															{
																Regex: "r.*",
															},
															{
																Contains: "c",
															},
															{
																Present: pointer.BoolPtr(true),
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedRouteConfigFields: &xds_route.RouteConfiguration{
				Name: "rds-inbound.80",
				VirtualHosts: []*xds_route.VirtualHost{
					{
						Name: "inbound_virtual-host|bookstore-v1.default.svc.cluster.local",
						Routes: []*xds_route.Route{
							{
								// corresponds to Rules[0]

								// Only the filter name is matched, not the marshalled config
								TypedPerFilterConfig: map[string]*any.Any{
									envoy.HTTPRBACFilterName: nil,
								},

								Action: &xds_route.Route_Route{
									Route: &xds_route.RouteAction{
										// Only the length of the rate limit configs are matched
										// This maps to the number of descriptors in the RateLimit policy

										// Since the input RateLimit policy has 2 descriptors, we add 2 elements
										// to the expected list for length check performed in the test body
										RateLimits: []*xds_route.RateLimit{nil, nil},
									},
								},
							},
						},
						// Only the length of the rate limit configs are matched
						// This maps to the number of descriptors in the RateLimit policy

						// Since the input RateLimit policy has 1 descriptor, we add 1 element
						// to the expected list for length check performed in the test body
						RateLimits: []*xds_route.RateLimit{nil},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			rb := &routesBuilder{
				inboundPortSpecificRouteConfigs: tc.InboundMeshHTTPRouteConfigsPerPort,
				statsHeaders:                    nil,
				trustDomain:                     "cluster.local",
			}
			actual := rb.buildInboundMeshRouteConfiguration()

			if tc.expectedRouteConfigFields == nil {
				assert.Nil(actual)
				return
			}

			assert.NotNil(actual)
			for _, routeConfig := range actual {
				assert.Equal(tc.expectedRouteConfigFields.Name, routeConfig.Name)
				assert.Len(routeConfig.VirtualHosts, len(tc.expectedRouteConfigFields.VirtualHosts))

				for i, vh := range routeConfig.VirtualHosts {
					assert.Len(vh.Routes, len(tc.expectedRouteConfigFields.VirtualHosts[i].Routes))
					assert.Len(vh.RateLimits, len(tc.expectedRouteConfigFields.VirtualHosts[i].RateLimits))

					// Verify that the expected typed filters on the VirtualHost are present
					for filter := range tc.expectedRouteConfigFields.VirtualHosts[i].TypedPerFilterConfig {
						assert.Contains(vh.TypedPerFilterConfig, filter)
					}

					// Verify that the expected typed filters on the Route are present
					for j, route := range vh.Routes {
						for filter := range tc.expectedRouteConfigFields.VirtualHosts[i].Routes[j].TypedPerFilterConfig {
							assert.Contains(route.TypedPerFilterConfig, filter)
						}

						if tc.expectedRouteConfigFields.VirtualHosts[i].Routes[j].GetRoute() != nil {
							assert.Len(route.GetRoute().RateLimits, len(tc.expectedRouteConfigFields.VirtualHosts[i].Routes[j].GetRoute().RateLimits))
						} else {
							assert.Len(route.GetRoute().RateLimits, 0) // If not specified in the test, expect 0
						}
					}
				}
			}
		})
	}

	statsWASMTestCases := []struct {
		name         string
		statsHeaders map[string]string
	}{
		{
			name:         "response headers added when WASM enabled",
			statsHeaders: map[string]string{"header-1": "val1", "header-2": "val2"},
		},
		{
			name: "response headers not added when WASM disabled",
		},
	}

	testInboundHTTPRouteConfigsPerPort := map[int][]*trafficpolicy.InboundTrafficPolicy{
		80: {
			{
				Name:      "bookstore-v1-default",
				Hostnames: tests.BookstoreV1Hostnames,
				Rules: []*trafficpolicy.Rule{
					{
						Route: trafficpolicy.RouteWeightedClusters{
							HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
						AllowedPrincipals: mapset.NewSet(tests.BookbuyerServiceAccount.ToServiceIdentity().AsPrincipal("cluster.local")),
					},
					{
						Route: trafficpolicy.RouteWeightedClusters{
							HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
						AllowedPrincipals: mapset.NewSet(tests.BookbuyerServiceAccount.ToServiceIdentity().AsPrincipal("cluster.local")),
					},
				},
			},
		},
	}

	for _, tc := range statsWASMTestCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			rb := &routesBuilder{
				inboundPortSpecificRouteConfigs: testInboundHTTPRouteConfigsPerPort,
				proxy:                           &models.Proxy{},
				statsHeaders:                    tc.statsHeaders,
				trustDomain:                     "cluster.local",
			}
			actual := rb.buildInboundMeshRouteConfiguration()
			for _, routeConfig := range actual {
				assert.Len(routeConfig.ResponseHeadersToAdd, len(tc.statsHeaders))
			}
		})
	}
}

func TestBuildIngressRouteConfiguration(t *testing.T) {
	testCases := []struct {
		name                      string
		ingressPolicies           []*trafficpolicy.InboundTrafficPolicy
		expectedRouteConfigFields *xds_route.RouteConfiguration
	}{
		{
			name:                      "no ingress policies",
			ingressPolicies:           nil,
			expectedRouteConfigFields: nil,
		},
		{
			name: "multiple ingress policies",
			ingressPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name:      "bookstore-v1-default",
					Hostnames: []string{"bookstore-v1.default.svc.cluster.local"},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
							},
							AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
							},
							AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
						},
					},
				},
				{
					Name:      "foo.com",
					Hostnames: []string{"foo.com"},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
							},
							AllowedPrincipals: mapset.NewSet(identity.WildcardPrincipal),
						},
					},
				},
			},
			expectedRouteConfigFields: &xds_route.RouteConfiguration{
				Name: "rds-ingress",
				VirtualHosts: []*xds_route.VirtualHost{
					{
						Name: "ingress_virtual-host|bookstore-v1.default.svc.cluster.local",
						Routes: []*xds_route.Route{
							{
								// corresponds to ingressPolicies[0].Rules[0]
							},
							{
								// corresponds to ingressPolicies[0].Rules[1]
							},
						},
					},
					{
						Name: "ingress_virtual-host|foo.com",
						Routes: []*xds_route.Route{
							{
								// corresponds to ingressPolicies[1].Rules[0]
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			rb := &routesBuilder{
				ingressTrafficPolicies: tc.ingressPolicies,
				trustDomain:            "cluster.local",
			}
			actual := rb.buildIngressConfiguration()

			if tc.expectedRouteConfigFields == nil {
				assert.Nil(actual)
				return
			}

			assert.NotNil(actual)
			assert.Equal(tc.expectedRouteConfigFields.Name, actual.Name)
			assert.Len(actual.VirtualHosts, len(tc.expectedRouteConfigFields.VirtualHosts))

			for i, vh := range actual.VirtualHosts {
				assert.Len(vh.Routes, len(tc.expectedRouteConfigFields.VirtualHosts[i].Routes))
			}
		})
	}
}

func TestBuildOutboundMeshRouteConfiguration(t *testing.T) {
	testCases := []struct {
		name                     string
		portSpecificRouteConfigs map[int][]*trafficpolicy.OutboundTrafficPolicy
		expectedRouteConfigs     []*xds_route.RouteConfiguration
	}{
		{
			name: "multiple route configs per port",
			portSpecificRouteConfigs: map[int][]*trafficpolicy.OutboundTrafficPolicy{
				80: {
					{
						Name: "bookstore-v1.default.svc.cluster.local",
						Hostnames: []string{
							"bookstore-v1.default",
							"bookstore-v1.default.svc",
							"bookstore-v1.default.svc.cluster",
							"bookstore-v1.default.svc.cluster.local",
							"bookstore-v1.default:80",
							"bookstore-v1.default.svc:80",
							"bookstore-v1.default.svc.cluster:80",
							"bookstore-v1.default.svc.cluster.local:80",
						},
						Routes: []*trafficpolicy.RouteWeightedClusters{
							{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: service.ClusterName("default/bookstore-v1|80"), Weight: 100},
								}),
							},
						},
					},
					{
						Name: "bookstore-v2.default.svc.cluster.local",
						Hostnames: []string{
							"bookstore-v2.default",
							"bookstore-v2.default.svc",
							"bookstore-v2.default.svc.cluster",
							"bookstore-v2.default.svc.cluster.local",
							"bookstore-v2.default:80",
							"bookstore-v2.default.svc:80",
							"bookstore-v2.default.svc.cluster:80",
							"bookstore-v2.default.svc.cluster.local:80",
						},
						Routes: []*trafficpolicy.RouteWeightedClusters{
							{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: service.ClusterName("default/bookstore-v2|80"), Weight: 100},
								}),
							},
						},
					},
				},
				90: {
					{
						Name: "bookstore-v1.default.svc.cluster.local",
						Hostnames: []string{
							"bookstore-v1.default",
							"bookstore-v1.default.svc",
							"bookstore-v1.default.svc.cluster",
							"bookstore-v1.default.svc.cluster.local",
							"bookstore-v1.default:90",
							"bookstore-v1.default.svc:90",
							"bookstore-v1.default.svc.cluster:90",
							"bookstore-v1.default.svc.cluster.local:90",
						},
						Routes: []*trafficpolicy.RouteWeightedClusters{
							{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: service.ClusterName("default/bookstore-v1|90"), Weight: 100},
								}),
							},
						},
					},
				},
			},

			expectedRouteConfigs: []*xds_route.RouteConfiguration{
				{
					Name:             "rds-outbound.80",
					ValidateClusters: &wrappers.BoolValue{Value: false},
					VirtualHosts: []*xds_route.VirtualHost{
						{
							Name: "outbound_virtual-host|bookstore-v1.default.svc.cluster.local",
							Domains: []string{
								"bookstore-v1.default",
								"bookstore-v1.default.svc",
								"bookstore-v1.default.svc.cluster",
								"bookstore-v1.default.svc.cluster.local",
								"bookstore-v1.default:80",
								"bookstore-v1.default.svc:80",
								"bookstore-v1.default.svc.cluster:80",
								"bookstore-v1.default.svc.cluster.local:80",
							},
							Routes: []*xds_route.Route{
								{
									Match: &xds_route.RouteMatch{
										PathSpecifier: &xds_route.RouteMatch_SafeRegex{
											SafeRegex: &xds_matcher.RegexMatcher{
												EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
												Regex:      ".*",
											},
										},
										Headers: []*xds_route.HeaderMatcher{
											{
												Name: ":method",
												HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
													SafeRegexMatch: &xds_matcher.RegexMatcher{
														EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
														Regex:      ".*",
													},
												},
											},
										},
									},
									Action: &xds_route.Route_Route{
										Route: &xds_route.RouteAction{
											ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
												WeightedClusters: &xds_route.WeightedCluster{
													Clusters: []*xds_route.WeightedCluster_ClusterWeight{
														{
															Name:   "default/bookstore-v1|80",
															Weight: &wrappers.UInt32Value{Value: 100},
														},
													},
													TotalWeight: &wrappers.UInt32Value{Value: 100},
												},
											},
											Timeout: &duration.Duration{Seconds: 0},
										},
									},
								},
							},
						},
						{
							Name: "outbound_virtual-host|bookstore-v2.default.svc.cluster.local",
							Domains: []string{
								"bookstore-v2.default",
								"bookstore-v2.default.svc",
								"bookstore-v2.default.svc.cluster",
								"bookstore-v2.default.svc.cluster.local",
								"bookstore-v2.default:80",
								"bookstore-v2.default.svc:80",
								"bookstore-v2.default.svc.cluster:80",
								"bookstore-v2.default.svc.cluster.local:80",
							},
							Routes: []*xds_route.Route{
								{
									Match: &xds_route.RouteMatch{
										PathSpecifier: &xds_route.RouteMatch_SafeRegex{
											SafeRegex: &xds_matcher.RegexMatcher{
												EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
												Regex:      ".*",
											},
										},
										Headers: []*xds_route.HeaderMatcher{
											{
												Name: ":method",
												HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
													SafeRegexMatch: &xds_matcher.RegexMatcher{
														EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
														Regex:      ".*",
													},
												},
											},
										},
									},
									Action: &xds_route.Route_Route{
										Route: &xds_route.RouteAction{
											ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
												WeightedClusters: &xds_route.WeightedCluster{
													Clusters: []*xds_route.WeightedCluster_ClusterWeight{
														{
															Name:   "default/bookstore-v2|80",
															Weight: &wrappers.UInt32Value{Value: 100},
														},
													},
													TotalWeight: &wrappers.UInt32Value{Value: 100},
												},
											},
											Timeout: &duration.Duration{Seconds: 0},
										},
									},
								},
							},
						},
					},
				},
				{
					Name:             "rds-outbound.90",
					ValidateClusters: &wrappers.BoolValue{Value: false},
					VirtualHosts: []*xds_route.VirtualHost{
						{
							Name: "outbound_virtual-host|bookstore-v1.default.svc.cluster.local",
							Domains: []string{
								"bookstore-v1.default",
								"bookstore-v1.default.svc",
								"bookstore-v1.default.svc.cluster",
								"bookstore-v1.default.svc.cluster.local",
								"bookstore-v1.default:90",
								"bookstore-v1.default.svc:90",
								"bookstore-v1.default.svc.cluster:90",
								"bookstore-v1.default.svc.cluster.local:90",
							},
							Routes: []*xds_route.Route{
								{
									Match: &xds_route.RouteMatch{
										PathSpecifier: &xds_route.RouteMatch_SafeRegex{
											SafeRegex: &xds_matcher.RegexMatcher{
												EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
												Regex:      ".*",
											},
										},
										Headers: []*xds_route.HeaderMatcher{
											{
												Name: ":method",
												HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
													SafeRegexMatch: &xds_matcher.RegexMatcher{
														EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
														Regex:      ".*",
													},
												},
											},
										},
									},
									Action: &xds_route.Route_Route{
										Route: &xds_route.RouteAction{
											ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
												WeightedClusters: &xds_route.WeightedCluster{
													Clusters: []*xds_route.WeightedCluster_ClusterWeight{
														{
															Name:   "default/bookstore-v1|90",
															Weight: &wrappers.UInt32Value{Value: 100},
														},
													},
													TotalWeight: &wrappers.UInt32Value{Value: 100},
												},
											},
											Timeout: &duration.Duration{Seconds: 0},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                     "no HTTP route configs",
			portSpecificRouteConfigs: nil,
			expectedRouteConfigs:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			rb := &routesBuilder{
				outboundPortSpecificRouteConfigs: tc.portSpecificRouteConfigs,
			}
			actual := rb.buildOutboundMeshRouteConfiguration()
			assert.ElementsMatch(tc.expectedRouteConfigs, actual)
		})
	}
}

func TestBuildEgressRouteConfiguration(t *testing.T) {
	testCases := []struct {
		name                     string
		portSpecificRouteConfigs map[int][]*trafficpolicy.EgressHTTPRouteConfig
		expectedRouteConfigs     []*xds_route.RouteConfiguration
	}{
		{
			name: "multiple route configs per port",
			portSpecificRouteConfigs: map[int][]*trafficpolicy.EgressHTTPRouteConfig{
				80: {
					{
						Name: "foo.com",
						Hostnames: []string{
							"foo.com",
							"foo.com:80",
						},
						RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
									WeightedClusters: mapset.NewSetFromSlice([]interface{}{
										service.WeightedCluster{ClusterName: service.ClusterName("foo.com:80"), Weight: 100},
									}),
								},
							},
						},
					},
					{
						Name: "bar.com",
						Hostnames: []string{
							"bar.com",
							"bar.com:80",
						},
						RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
									WeightedClusters: mapset.NewSetFromSlice([]interface{}{
										service.WeightedCluster{ClusterName: service.ClusterName("bar.com:80"), Weight: 100},
									}),
								},
							},
						},
					},
				},
				90: {
					{
						Name: "baz.com",
						Hostnames: []string{
							"baz.com",
							"baz.com:90",
						},
						RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
									WeightedClusters: mapset.NewSetFromSlice([]interface{}{
										service.WeightedCluster{ClusterName: service.ClusterName("baz.com:90"), Weight: 100},
									}),
								},
							},
						},
					},
				},
			},
			expectedRouteConfigs: []*xds_route.RouteConfiguration{
				{
					Name:             "rds-egress.80",
					ValidateClusters: &wrappers.BoolValue{Value: false},
					VirtualHosts: []*xds_route.VirtualHost{
						{
							Name: "egress_virtual-host|foo.com",
							Domains: []string{
								"foo.com",
								"foo.com:80",
							},
							Routes: []*xds_route.Route{
								{
									Match: &xds_route.RouteMatch{
										PathSpecifier: &xds_route.RouteMatch_SafeRegex{
											SafeRegex: &xds_matcher.RegexMatcher{
												EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
												Regex:      ".*",
											},
										},
										Headers: []*xds_route.HeaderMatcher{
											{
												Name: ":method",
												HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
													SafeRegexMatch: &xds_matcher.RegexMatcher{
														EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
														Regex:      ".*",
													},
												},
											},
										},
									},
									Action: &xds_route.Route_Route{
										Route: &xds_route.RouteAction{
											ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
												WeightedClusters: &xds_route.WeightedCluster{
													Clusters: []*xds_route.WeightedCluster_ClusterWeight{
														{
															Name:   "foo.com:80",
															Weight: &wrappers.UInt32Value{Value: 100},
														},
													},
													TotalWeight: &wrappers.UInt32Value{Value: 100},
												},
											},
											Timeout: &duration.Duration{Seconds: 0},
										},
									},
								},
							},
						},
						{
							Name: "egress_virtual-host|bar.com",
							Domains: []string{
								"bar.com",
								"bar.com:80",
							},
							Routes: []*xds_route.Route{
								{
									Match: &xds_route.RouteMatch{
										PathSpecifier: &xds_route.RouteMatch_SafeRegex{
											SafeRegex: &xds_matcher.RegexMatcher{
												EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
												Regex:      ".*",
											},
										},
										Headers: []*xds_route.HeaderMatcher{
											{
												Name: ":method",
												HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
													SafeRegexMatch: &xds_matcher.RegexMatcher{
														EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
														Regex:      ".*",
													},
												},
											},
										},
									},
									Action: &xds_route.Route_Route{
										Route: &xds_route.RouteAction{
											ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
												WeightedClusters: &xds_route.WeightedCluster{
													Clusters: []*xds_route.WeightedCluster_ClusterWeight{
														{
															Name:   "bar.com:80",
															Weight: &wrappers.UInt32Value{Value: 100},
														},
													},
													TotalWeight: &wrappers.UInt32Value{Value: 100},
												},
											},
											Timeout: &duration.Duration{Seconds: 0},
										},
									},
								},
							},
						},
					},
				},
				{
					Name:             "rds-egress.90",
					ValidateClusters: &wrappers.BoolValue{Value: false},
					VirtualHosts: []*xds_route.VirtualHost{
						{
							Name: "egress_virtual-host|baz.com",
							Domains: []string{
								"baz.com",
								"baz.com:90",
							},
							Routes: []*xds_route.Route{
								{
									Match: &xds_route.RouteMatch{
										PathSpecifier: &xds_route.RouteMatch_SafeRegex{
											SafeRegex: &xds_matcher.RegexMatcher{
												EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
												Regex:      ".*",
											},
										},
										Headers: []*xds_route.HeaderMatcher{
											{
												Name: ":method",
												HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
													SafeRegexMatch: &xds_matcher.RegexMatcher{
														EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
														Regex:      ".*",
													},
												},
											},
										},
									},
									Action: &xds_route.Route_Route{
										Route: &xds_route.RouteAction{
											ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
												WeightedClusters: &xds_route.WeightedCluster{
													Clusters: []*xds_route.WeightedCluster_ClusterWeight{
														{
															Name:   "baz.com:90",
															Weight: &wrappers.UInt32Value{Value: 100},
														},
													},
													TotalWeight: &wrappers.UInt32Value{Value: 100},
												},
											},
											Timeout: &duration.Duration{Seconds: 0},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:                     "no HTTP route configs",
			portSpecificRouteConfigs: nil,
			expectedRouteConfigs:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			rb := &routesBuilder{
				egressPortSpecificRouteConfigs: tc.portSpecificRouteConfigs,
			}

			actual := rb.buildEgressRouteConfiguration()
			assert.ElementsMatch(tc.expectedRouteConfigs, actual)
		})
	}
}
