package ingress

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pkg/errors"
	tassert "github.com/stretchr/testify/assert"
	networkingV1 "k8s.io/api/networking/v1"
	networkingV1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/discovery"
	fakeDiscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
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

func TestGetIngressNetworkingV1AndVebeta1(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockKubeController := k8s.NewMockController(mockCtrl)
	assert := tassert.New(t)

	mockKubeController.EXPECT().IsMonitoredNamespace(gomock.Any()).Return(true).AnyTimes()
	meshSvc := service.MeshService{Name: "foo", Namespace: "test"}

	testCases := []struct {
		name               string
		ingressResource    runtime.Object
		version            string
		expectedMatchCount int
	}{
		{
			name: "ingress v1 is not ignored",
			ingressResource: &networkingV1.Ingress{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "networking.k8s.io/v1",
					Kind:       "Ingress",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-1",
					Namespace: meshSvc.Namespace,
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
													Name: meshSvc.Name,
													Port: networkingV1.ServiceBackendPort{
														Number: 80,
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
			version:            networkingV1.SchemeGroupVersion.String(),
			expectedMatchCount: 1,
		},
		{
			name: "ingress v1 is ignored using a label",
			ingressResource: &networkingV1.Ingress{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "networking.k8s.io/v1",
					Kind:       "Ingress",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-1",
					Namespace: meshSvc.Namespace,
					Labels:    map[string]string{constants.IgnoreLabel: "true"}, // ignored
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
													Name: meshSvc.Name,
													Port: networkingV1.ServiceBackendPort{
														Number: 80,
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
			version:            networkingV1.SchemeGroupVersion.String(),
			expectedMatchCount: 0,
		},
		{
			name: "ingress v1beta1 is not ignored",
			ingressResource: &networkingV1beta1.Ingress{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "networking.k8s.io/v1beta1",
					Kind:       "Ingress",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-1",
					Namespace: meshSvc.Namespace,
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
												ServiceName: meshSvc.Name,
												ServicePort: intstr.IntOrString{
													Type:   intstr.Int,
													IntVal: 80,
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
			version:            networkingV1beta1.SchemeGroupVersion.String(),
			expectedMatchCount: 1,
		},
		{
			name: "ingress v1beta1 is ignored using a label",
			ingressResource: &networkingV1beta1.Ingress{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "networking.k8s.io/v1beta1",
					Kind:       "Ingress",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-1",
					Namespace: meshSvc.Namespace,
					Labels:    map[string]string{constants.IgnoreLabel: "true"}, // ignored
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
												ServiceName: meshSvc.Name,
												ServicePort: intstr.IntOrString{
													Type:   intstr.Int,
													IntVal: 80,
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
			version:            networkingV1beta1.SchemeGroupVersion.String(),
			expectedMatchCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(tc.ingressResource)
			fakeClient.Discovery().(*fakeDiscovery.FakeDiscovery).Resources = []*metav1.APIResourceList{
				{
					GroupVersion: networkingV1.SchemeGroupVersion.String(),
					APIResources: []metav1.APIResource{
						{Kind: "Ingress"},
					},
				},
				{
					GroupVersion: networkingV1beta1.SchemeGroupVersion.String(),
					APIResources: []metav1.APIResource{
						{Kind: "Ingress"},
					},
				},
			}

			c, err := NewIngressClient(fakeClient, mockKubeController, make(chan struct{}), nil)
			assert.Nil(err)

			switch tc.version {
			case networkingV1.SchemeGroupVersion.String():
				ingresses, err := c.GetIngressNetworkingV1(meshSvc)
				assert.Nil(err)
				assert.Len(ingresses, tc.expectedMatchCount)

			case networkingV1beta1.SchemeGroupVersion.String():
				ingresses, err := c.GetIngressNetworkingV1beta1(meshSvc)
				assert.Nil(err)
				assert.Len(ingresses, tc.expectedMatchCount)
			}
		})
	}
}
