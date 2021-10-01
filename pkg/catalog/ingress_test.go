package catalog

import (
	"fmt"
	"net"
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
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/policy"

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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
				Name:       "foo",
				Namespace:  "testns",
				TargetPort: uint16(fakeIngressPort),
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
									ClusterName: "testns/foo|80|local",
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
									ClusterName: "testns/foo|80|local",
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
		ingressV1                   []*networkingV1.Ingress
		ingressBackend              *policyV1alpha1.IngressBackend
		expectedPolicy              *trafficpolicy.IngressTrafficPolicy
		expectError                 bool
	}{
		{
			name:                        "HTTP ingress with k8s ingress enabled",
			ingressBackendPolicyEnabled: false,
			enableHTTPSIngress:          false,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", Protocol: "http", TargetPort: 80},
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
										ClusterName: "testns/foo|80|local",
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
						Name:     "ingress_testns/foo_80_http",
						Protocol: "http",
						Port:     80,
					},
				},
			},
			expectError: false,
		},
		{
			name:                        "HTTPS ingress with k8s ingress enabled",
			ingressBackendPolicyEnabled: false,
			enableHTTPSIngress:          true,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", Protocol: "http", TargetPort: 80},
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
										ClusterName: "testns/foo|80|local",
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
						Name:                     "ingress_testns/foo_80_https",
						Protocol:                 "https",
						Port:                     80,
						SkipClientCertValidation: true,
					},
					{
						Name:                     "ingress_testns/foo_80_https_with_sni",
						Protocol:                 "https",
						Port:                     80,
						SkipClientCertValidation: true,
						ServerNames:              []string{"foo.testns.svc.cluster.local"},
					},
				},
			},
			expectError: false,
		},
		{
			name:                        "No ingress routes",
			ingressBackendPolicyEnabled: false,
			enableHTTPSIngress:          false,
			meshSvc:                     service.MeshService{Name: "foo", Namespace: "testns", Protocol: "http", TargetPort: 80},
			ingressBackend:              nil,
			expectedPolicy:              nil,
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
			expectedPolicy: &trafficpolicy.IngressTrafficPolicy{
				HTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
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
								AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
							},
						},
					},
				},
				TrafficMatches: []*trafficpolicy.IngressTrafficMatch{
					{
						Name:           "ingress_testns/foo_80_http",
						Protocol:       "http",
						Port:           80,
						SourceIPRanges: []string{"10.0.0.10/32"}, // Endpoint of 'ingressSourceSvc' referenced as a source
					},
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
			expectedPolicy: &trafficpolicy.IngressTrafficPolicy{
				HTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
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
								AllowedServiceIdentities: mapset.NewSet(identity.ServiceIdentity("ingressGw.ingressGwNs.cluster.local")),
							},
						},
					},
				},
				TrafficMatches: []*trafficpolicy.IngressTrafficMatch{
					{
						Name:                     "ingress_testns/foo_80_https",
						Protocol:                 "https",
						Port:                     80,
						SourceIPRanges:           []string{"10.0.0.10/32"}, // Endpoint of 'ingressSourceSvc' referenced as a source
						SkipClientCertValidation: false,
						ServerNames:              []string{"foo.org"},
					},
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
			expectedPolicy: &trafficpolicy.IngressTrafficPolicy{
				HTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
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
								AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
							},
						},
					},
				},
				TrafficMatches: []*trafficpolicy.IngressTrafficMatch{
					{
						Name:                     "ingress_testns/foo_80_https",
						Protocol:                 "https",
						Port:                     80,
						SourceIPRanges:           []string{"10.0.0.10/32"}, // Endpoint of 'ingressSourceSvc' referenced as a source
						SkipClientCertValidation: true,
					},
				},
			},
			expectError: false,
		},
		{
			name:                        "Specifying a source service without endpoints in an IngressBackend should error",
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
			expectedPolicy: nil,
			expectError:    true,
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
			expectedPolicy: &trafficpolicy.IngressTrafficPolicy{
				HTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{
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
								AllowedServiceIdentities: mapset.NewSet(identity.WildcardServiceIdentity),
							},
						},
					},
				},
				TrafficMatches: []*trafficpolicy.IngressTrafficMatch{
					{
						Name:           "ingress_testns/foo_80_http",
						Protocol:       "http",
						Port:           80,
						SourceIPRanges: []string{"10.0.0.0/10", "20.0.0.0/10"}, // 'IPRange' referenced as a source
					},
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
			expectedPolicy: nil,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockIngressMonitor := ingress.NewMockMonitor(mockCtrl)
			mockServiceProvider := service.NewMockProvider(mockCtrl)
			mockEndpointsProvider := endpoint.NewMockProvider(mockCtrl)
			mockCfg := configurator.NewMockConfigurator(mockCtrl)
			mockPolicyController := policy.NewMockController(mockCtrl)
			mockKubeController := k8s.NewMockController(mockCtrl)

			meshCatalog := &MeshCatalog{
				ingressMonitor:     mockIngressMonitor,
				serviceProviders:   []service.Provider{mockServiceProvider},
				endpointsProviders: []endpoint.Provider{mockEndpointsProvider},
				configurator:       mockCfg,
				policyController:   mockPolicyController,
				kubeController:     mockKubeController,
			}

			// Note: if AnyTimes() is used with a mock function, it implies the function may or may not be called
			// depending on the test case.
			mockCfg.EXPECT().GetFeatureFlags().Return(configV1alpha1.FeatureFlags{EnableIngressBackendPolicy: tc.ingressBackendPolicyEnabled}).Times(1)
			mockCfg.EXPECT().UseHTTPSIngress().Return(tc.enableHTTPSIngress).Times(1).AnyTimes()
			mockIngressMonitor.EXPECT().GetIngressNetworkingV1(gomock.Any()).Return(tc.ingressV1, nil).AnyTimes()
			mockIngressMonitor.EXPECT().GetIngressNetworkingV1beta1(gomock.Any()).Return(nil, nil).AnyTimes()
			mockPolicyController.EXPECT().GetIngressBackendPolicy(tc.meshSvc).Return(tc.ingressBackend).AnyTimes()
			mockServiceProvider.EXPECT().GetID().Return("mock").AnyTimes()
			mockEndpointsProvider.EXPECT().ListEndpointsForService(ingressSourceSvc).Return(ingressBackendSvcEndpoints).AnyTimes()
			mockEndpointsProvider.EXPECT().ListEndpointsForService(sourceSvcWithoutEndpoints).Return(nil).AnyTimes()
			mockEndpointsProvider.EXPECT().GetID().Return("mock").AnyTimes()
			mockKubeController.EXPECT().UpdateStatus(gomock.Any()).Return(nil, nil).AnyTimes()

			actual, err := meshCatalog.GetIngressTrafficPolicy(tc.meshSvc)
			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedPolicy, actual)
		})
	}
}
