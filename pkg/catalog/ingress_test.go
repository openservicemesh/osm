package catalog

import (
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	networkingV1 "k8s.io/api/networking/v1"
	networkingV1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	configV1alpha1 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/ingress"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

var (
	fakeIngressPort int32 = 80
)

func TestGetIngressPoliciesNetworkingV1beta1(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockIngressMonitor := ingress.NewMockMonitor(mockCtrl)
	meshCatalog := &MeshCatalog{
		ingressMonitor: mockIngressMonitor,
	}

	type testCase struct {
		name                    string
		svc                     service.MeshService
		ingresses               []*networkingV1beta1.Ingress
		expectedTrafficPolicies []*trafficpolicy.InboundTrafficPolicy
		excpectError            bool
	}

	testCases := []testCase{
		{
			name: "Ingress rule with multiple rules and no default backend",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeImplementationSpecific))),
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeExact))),
												Path:     "/fake2-path1",
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-1.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake2-path1",
									PathMatchType: trafficpolicy.PathMatchExact,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Ingress rule with multiple rules and a default backend",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Backend: &networkingV1beta1.IngressBackend{
							ServiceName: "foo",
							ServicePort: intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: fakeIngressPort,
							},
						},
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypePrefix))),
												Path:     "/fake1-path1",
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypePrefix))),
												Path:     "/fake2-path1",
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|*",
					Hostnames: []string{
						"*",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          `/fake1-path1(/.*)?$`,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-1.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          `/fake2-path1(/.*)?$`,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Multiple ingresses matching different hosts",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeImplementationSpecific))),
												Path:     "/fake1-path1",
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-2",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeImplementationSpecific))),
												Path:     "/fake2-path1",
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-2.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake2-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Multiple ingresses matching same hosts with different rules",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeImplementationSpecific))),
												Path:     `/fake1-path1.*`,
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-2",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeImplementationSpecific))),
												Path:     `/fake1-path2(/.*)?$`,
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          `/fake1-path1.*`,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          `/fake1-path2(/.*)?$`,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Ingress rule with unset pathType must default to ImplementationSpecific",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeImplementationSpecific))),
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												// PathType is unset, will default to ImplementationSpecific
												Path: "/fake2-path1",
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-1.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake2-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Ingress rule with invalid pathType must be ignored",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeImplementationSpecific))),
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												// PathType is invalid, this will be ignored and logged as an error
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr("invalid")),
												Path:     "/fake2-path1",
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Wildcard path / with Prefix type should be matched as a string prefix",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												Path:     "/",
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypePrefix))),
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Prefix path type with trailing slash should be stripped",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												Path:     "/foo/", // Trailing slash should be stripped in the route programmed
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypePrefix))),
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/foo(/.*)?$",
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockIngressMonitor.EXPECT().GetIngressNetworkingV1beta1(tc.svc).Return(tc.ingresses, nil).Times(1)

			actualPolicies, err := meshCatalog.getIngressPoliciesNetworkingV1beta1(tc.svc)

			assert.Equal(tc.excpectError, err != nil)
			assert.ElementsMatch(tc.expectedTrafficPolicies, actualPolicies)
		})
	}
}

func TestGetIngressPoliciesNetworkingV1(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockIngressMonitor := ingress.NewMockMonitor(mockCtrl)
	meshCatalog := &MeshCatalog{
		ingressMonitor: mockIngressMonitor,
	}

	type testCase struct {
		name                    string
		svc                     service.MeshService
		ingresses               []*networkingV1.Ingress
		expectedTrafficPolicies []*trafficpolicy.InboundTrafficPolicy
		excpectError            bool
	}

	testCases := []testCase{
		{
			name: "Ingress rule with multiple rules and no default backend",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
														},
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeExact))),
												Path:     "/fake2-path1",
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-1.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake2-path1",
									PathMatchType: trafficpolicy.PathMatchExact,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Ingress rule with multiple rules and a default backend",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						DefaultBackend: &networkingV1.IngressBackend{
							Service: &networkingV1.IngressServiceBackend{
								Name: "foo",
								Port: networkingV1.ServiceBackendPort{
									Number: fakeIngressPort,
								},
							},
						},
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypePrefix))),
												Path:     "/fake1-path1",
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
														},
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypePrefix))),
												Path:     "/fake2-path1",
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|*",
					Hostnames: []string{
						"*",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          `/fake1-path1(/.*)?$`,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-1.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          `/fake2-path1(/.*)?$`,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Multiple ingresses matching different hosts",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Path:     "/fake1-path1",
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-2",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Path:     "/fake2-path1",
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-2.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake2-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Multiple ingresses matching same hosts with different rules",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Path:     `/fake1-path1.*`,
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-2",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Path:     `/fake1-path2(/.*)?$`,
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          `/fake1-path1.*`,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          `/fake1-path2(/.*)?$`,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Ingress rule with unset pathType must default to ImplementationSpecific",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
														},
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												// PathType is unset, will default to ImplementationSpecific
												Path: "/fake2-path1",
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-1.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake2-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Ingress rule with invalid pathType must be ignored",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
														},
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												// PathType is invalid, this will be ignored and logged as an error
												PathType: (*networkingV1.PathType)(pointer.StringPtr("invalid")),
												Path:     "/fake2-path1",
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Wildcard path / with Prefix type should be matched as a string prefix",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												Path:     "/",
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypePrefix))),
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Prefix path type with trailing slash should be stripped",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingresses: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												Path:     "/foo/", // Trailing slash should be stripped in the route programmed
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypePrefix))),
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/foo(/.*)?$",
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockIngressMonitor.EXPECT().GetIngressNetworkingV1(tc.svc).Return(tc.ingresses, nil).Times(1)

			actualPolicies, err := meshCatalog.getIngressPoliciesNetworkingV1(tc.svc)
			assert.Equal(tc.excpectError, err != nil)
			assert.ElementsMatch(tc.expectedTrafficPolicies, actualPolicies)
		})
	}
}

func TestGetIngressPoliciesFromK8s(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockIngressMonitor := ingress.NewMockMonitor(mockCtrl)
	meshCatalog := &MeshCatalog{
		ingressMonitor: mockIngressMonitor,
	}

	type testCase struct {
		name                    string
		svc                     service.MeshService
		ingressesV1beta1        []*networkingV1beta1.Ingress
		ingressesV1             []*networkingV1.Ingress
		expectedTrafficPolicies []*trafficpolicy.InboundTrafficPolicy
		excpectError            bool
	}

	testCases := []testCase{
		{
			name: "Ingress v1 with multiple rules",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingressesV1: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
														},
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeExact))),
												Path:     "/fake2-path1",
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-1.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake2-path1",
									PathMatchType: trafficpolicy.PathMatchExact,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Ingress v1beta1 with with multiple rules",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingressesV1beta1: []*networkingV1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeImplementationSpecific))),
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeExact))),
												Path:     "/fake2-path1",
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-1.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake2-path1",
									PathMatchType: trafficpolicy.PathMatchExact,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name: "Ingress v1 and v1beta both present",
			svc: service.MeshService{
				Name:          "foo",
				Namespace:     "testns",
				ClusterDomain: constants.LocalDomain,
			},
			ingressesV1: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			ingressesV1beta1: []*networkingV1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-2",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1beta1.IngressSpec{
						Rules: []networkingV1beta1.IngressRule{
							{
								Host: "fake2.com",
								IngressRuleValue: networkingV1beta1.IngressRuleValue{
									HTTP: &networkingV1beta1.HTTPIngressRuleValue{
										Paths: []networkingV1beta1.HTTPIngressPath{
											{
												PathType: (*networkingV1beta1.PathType)(pointer.StringPtr(string(networkingV1beta1.PathTypeExact))),
												Path:     "/fake2-path1",
												Backend: networkingV1beta1.IngressBackend{
													ServiceName: "foo",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: fakeIngressPort,
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
			expectedTrafficPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "ingress-1.testns|fake1.com",
					Hostnames: []string{
						"fake1.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake1-path1",
									PathMatchType: trafficpolicy.PathMatchPrefix,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
				{
					Name: "ingress-2.testns|fake2.com",
					Hostnames: []string{
						"fake2.com",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/fake2-path1",
									PathMatchType: trafficpolicy.PathMatchExact,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "testns/foo/local",
									Weight:      100,
								}),
							},
							AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
						},
					},
				},
			},
			excpectError: false,
		},
		{
			name:                    "No ingresses",
			excpectError:            false,
			expectedTrafficPolicies: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockIngressMonitor.EXPECT().GetIngressNetworkingV1(tc.svc).Return(tc.ingressesV1, nil).Times(1)
			mockIngressMonitor.EXPECT().GetIngressNetworkingV1beta1(tc.svc).Return(tc.ingressesV1beta1, nil).Times(1)

			actualPolicies, err := meshCatalog.getIngressPoliciesFromK8s(tc.svc)

			assert.Equal(tc.excpectError, err != nil)
			assert.ElementsMatch(tc.expectedTrafficPolicies, actualPolicies)
		})
	}
}

func TestGetIngressPolicyName(t *testing.T) {
	assert := tassert.New(t)
	testCases := []struct {
		name         string
		namespace    string
		host         string
		expectedName string
	}{
		{
			name:         "bookbuyer",
			namespace:    "default",
			host:         "*",
			expectedName: "bookbuyer.default|*",
		},
		{
			name:         "bookbuyer",
			namespace:    "bookbuyer-ns",
			host:         "foobar.com",
			expectedName: "bookbuyer.bookbuyer-ns|foobar.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := getIngressTrafficPolicyName(tc.name, tc.namespace, tc.host)
			assert.Equal(tc.expectedName, actual)
		})
	}
}

func TestGetIngressTrafficPolicy(t *testing.T) {
	testCases := []struct {
		name                        string
		ingressBackendPolicyEnabled bool
		enableHTTPSIngress          bool
		meshSvc                     service.MeshService
		ingressV1                   []*networkingV1.Ingress
		ingressBackend              *policyV1alpha1.IngressBackend
		expectedPolicy              *trafficpolicy.IngressTrafficPolicy
		expectError                 bool
	}{
		{
			name:                        "HTTP ingress with k8s ingress enabled",
			ingressBackendPolicyEnabled: false,
			enableHTTPSIngress:          false,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", ClusterDomain: "local"},
			ingressV1: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			ingressBackend: nil,
			expectedPolicy: &trafficpolicy.IngressTrafficPolicy{
				HTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
					{
						Name: "ingress-1.testns|fake1.com",
						Hostnames: []string{
							"fake1.com",
						},
						Rules: []*trafficpolicy.Rule{
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
										Path:          "/fake1-path1",
										PathMatchType: trafficpolicy.PathMatchPrefix,
										Methods:       []string{constants.WildcardHTTPMethod},
									},
									WeightedClusters: mapset.NewSet(service.WeightedCluster{
										ClusterName: "testns/foo/local",
										Weight:      100,
									}),
								},
								AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
							},
						},
					},
				},
				TrafficMatches: []*trafficpolicy.IngressTrafficMatch{
					{
						Name:     "ingress_testns/foo/local_80_http",
						Protocol: "http",
						Port:     80,
					},
				},
			},
		},
		{
			name:                        "HTTPS ingress with k8s ingress enabled",
			ingressBackendPolicyEnabled: false,
			enableHTTPSIngress:          true,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", ClusterDomain: "local"},
			ingressV1: []*networkingV1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-1",
						Namespace: "testns",
						Annotations: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "enabled",
						},
					},
					Spec: networkingV1.IngressSpec{
						Rules: []networkingV1.IngressRule{
							{
								Host: "fake1.com",
								IngressRuleValue: networkingV1.IngressRuleValue{
									HTTP: &networkingV1.HTTPIngressRuleValue{
										Paths: []networkingV1.HTTPIngressPath{
											{
												Path:     "/fake1-path1",
												PathType: (*networkingV1.PathType)(pointer.StringPtr(string(networkingV1.PathTypeImplementationSpecific))),
												Backend: networkingV1.IngressBackend{
													Service: &networkingV1.IngressServiceBackend{
														Name: "foo",
														Port: networkingV1.ServiceBackendPort{
															Number: fakeIngressPort,
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
			ingressBackend: nil,
			expectedPolicy: &trafficpolicy.IngressTrafficPolicy{
				HTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
					{
						Name: "ingress-1.testns|fake1.com",
						Hostnames: []string{
							"fake1.com",
						},
						Rules: []*trafficpolicy.Rule{
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
										Path:          "/fake1-path1",
										PathMatchType: trafficpolicy.PathMatchPrefix,
										Methods:       []string{constants.WildcardHTTPMethod},
									},
									WeightedClusters: mapset.NewSet(service.WeightedCluster{
										ClusterName: "testns/foo/local",
										Weight:      100,
									}),
								},
								AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
							},
						},
					},
				},
				TrafficMatches: []*trafficpolicy.IngressTrafficMatch{
					{
						Name:                     "ingress_testns/foo/local_80_https",
						Protocol:                 "https",
						Port:                     80,
						SkipClientCertValidation: true,
					},
					{
						Name:                     "ingress_testns/foo/local_80_https_with_sni",
						Protocol:                 "https",
						Port:                     80,
						SkipClientCertValidation: true,
						ServerNames:              []string{"foo.testns.svc.cluster.local"},
					},
				},
			},
		},
		{
			name:                        "No ingress routes",
			ingressBackendPolicyEnabled: false,
			enableHTTPSIngress:          false,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", ClusterDomain: "local"},
			ingressBackend:              nil,
			expectedPolicy:              nil,
		},
	}

	portToProtocolMapping := map[uint32]string{80: "http", 443: "tcp"}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockIngressMonitor := ingress.NewMockMonitor(mockCtrl)
			mockServiceProvider := service.NewMockProvider(mockCtrl)
			mockCfg := configurator.NewMockConfigurator(mockCtrl)

			meshCatalog := &MeshCatalog{
				ingressMonitor:   mockIngressMonitor,
				serviceProviders: []service.Provider{mockServiceProvider},
				configurator:     mockCfg,
			}

			mockCfg.EXPECT().GetFeatureFlags().Return(configV1alpha1.FeatureFlags{EnableIngressBackendPolicy: tc.ingressBackendPolicyEnabled}).Times(1)
			mockCfg.EXPECT().UseHTTPSIngress().Return(tc.enableHTTPSIngress).Times(1).AnyTimes()                                            // may or may not be called
			mockIngressMonitor.EXPECT().GetIngressNetworkingV1(gomock.Any()).Return(tc.ingressV1, nil).AnyTimes()                           // may or may not be called
			mockIngressMonitor.EXPECT().GetIngressNetworkingV1beta1(gomock.Any()).Return(nil, nil).AnyTimes()                               // may or may not be called
			mockServiceProvider.EXPECT().GetTargetPortToProtocolMappingForService(tc.meshSvc).Return(portToProtocolMapping, nil).AnyTimes() // may or may not be called

			actual, err := meshCatalog.GetIngressTrafficPolicy(tc.meshSvc)
			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedPolicy, actual)
		})
	}
}
