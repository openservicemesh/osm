package catalog

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetApexServicesForBackendService(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testSplit2 := split.TrafficSplit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
		},
		Spec: split.TrafficSplitSpec{
			Service: "apex-split-1",
			Backends: []split.TrafficSplitBackend{
				{
					Service: tests.BookstoreV1ServiceName,
					Weight:  tests.Weight10,
				},
				{
					Service: tests.BookstoreV2ServiceName,
					Weight:  tests.Weight90,
				},
			},
		},
	}

	testSplit3 := split.TrafficSplit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "bar",
		},
		Spec: split.TrafficSplitSpec{
			Service: "apex-split-1",
			Backends: []split.TrafficSplitBackend{
				{
					Service: tests.BookstoreV1ServiceName,
					Weight:  tests.Weight10,
				},
				{
					Service: tests.BookstoreV2ServiceName,
					Weight:  tests.Weight90,
				},
			},
		},
	}

	testCases := []struct {
		name          string
		trafficsplits []*split.TrafficSplit
		expected      []service.MeshService
	}{
		{
			name:          "single traffic split match",
			trafficsplits: []*split.TrafficSplit{&tests.TrafficSplit},
			expected:      []service.MeshService{tests.BookstoreApexService},
		},
		{
			name:          "no traffic split match",
			trafficsplits: []*split.TrafficSplit{&testSplit3},
			expected:      []service.MeshService{},
		},
		{
			name:          "multiple traffic split matches",
			trafficsplits: []*split.TrafficSplit{&tests.TrafficSplit, &testSplit2},
			expected:      []service.MeshService{tests.BookstoreApexService, {Name: "apex-split-1", Namespace: "default"}},
		},
		{
			name:          "no traffic splits present, so no backeds returned",
			trafficsplits: []*split.TrafficSplit{},
			expected:      []service.MeshService{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockKubeController := k8s.NewMockController(mockCtrl)
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)
			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
			}
			mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficsplits).AnyTimes()
			actual := mc.getApexServicesForBackendService(tests.BookstoreV1Service)
			assert.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestIsTrafficSplitBackendService(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testSplit2 := split.TrafficSplit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
		},
		Spec: split.TrafficSplitSpec{
			Service: "apex-split-1",
			Backends: []split.TrafficSplitBackend{
				{
					Service: tests.BookstoreV1ServiceName,
					Weight:  tests.Weight10,
				},
				{
					Service: tests.BookstoreV2ServiceName,
					Weight:  tests.Weight90,
				},
			},
		},
	}

	testSplit3 := split.TrafficSplit{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "bar",
		},
		Spec: split.TrafficSplitSpec{
			Service: "apex-split-1",
			Backends: []split.TrafficSplitBackend{
				{
					Service: tests.BookstoreV1ServiceName,
					Weight:  tests.Weight10,
				},
				{
					Service: tests.BookstoreV2ServiceName,
					Weight:  tests.Weight90,
				},
			},
		},
	}

	testCases := []struct {
		name           string
		trafficsplits  []*split.TrafficSplit
		backendService service.MeshService
		expected       bool
	}{
		{
			name:           "bookstore-v1 is an backend service",
			trafficsplits:  []*split.TrafficSplit{&tests.TrafficSplit},
			backendService: tests.BookstoreV1Service,
			expected:       true,
		},
		{
			name:           "bookstore-apex is not a backend service",
			trafficsplits:  []*split.TrafficSplit{&testSplit2},
			backendService: tests.BookstoreApexService,
			expected:       false,
		},
		{
			name:           "bookstore-v1 is not a backend service",
			trafficsplits:  []*split.TrafficSplit{&testSplit2},
			backendService: tests.BookstoreApexService,
			expected:       false,
		},
		{
			name:           "bookstore-apex is not a backend service across multiple traffic splits",
			trafficsplits:  []*split.TrafficSplit{&testSplit2, &testSplit3},
			backendService: tests.BookstoreApexService,
			expected:       false,
		},
		{
			name:           "bookstore-v1 is an backend service across multiple traffic splits",
			trafficsplits:  []*split.TrafficSplit{&testSplit2, &tests.TrafficSplit, &testSplit3},
			backendService: tests.BookstoreV1Service,
			expected:       true,
		},
		{
			name:           "no traffic splits present, must return false",
			trafficsplits:  []*split.TrafficSplit{},
			backendService: tests.BookstoreV1Service,
			expected:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockKubeController := k8s.NewMockController(mockCtrl)
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)
			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
			}
			mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficsplits).AnyTimes()
			actual := mc.isTrafficSplitBackendService(tc.backendService)
			assert.Equal(tc.expected, actual)
		})
	}
}

func TestListServiceAccountsForService(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := kubernetes.NewMockController(mockCtrl)
	mc := &MeshCatalog{
		kubeController: mockKubeController,
	}

	testCases := []struct {
		svc                 service.MeshService
		expectedSvcAccounts []service.K8sServiceAccount
		expectedError       error
	}{
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			[]service.K8sServiceAccount{{Name: "sa-1", Namespace: "ns-1"}, {Name: "sa-2", Namespace: "ns-1"}},
			nil,
		},
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			[]service.K8sServiceAccount{{Name: "sa-1", Namespace: "ns-1"}, {Name: "sa-2", Namespace: "ns-1"}},
			nil,
		},
		{
			service.MeshService{Name: "foo", Namespace: "ns-1"},
			nil,
			errors.New("test error"),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			mockKubeController.EXPECT().ListServiceAccountsForService(tc.svc).Return(tc.expectedSvcAccounts, tc.expectedError).Times(1)

			svcAccounts, err := mc.ListServiceAccountsForService(tc.svc)
			assert.ElementsMatch(svcAccounts, tc.expectedSvcAccounts)
			assert.Equal(err, tc.expectedError)
		})
	}
}

func TestGetPortToProtocolMappingForService(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type endpointProviderConfig struct {
		provider          *endpoint.MockProvider
		portToProtocolMap map[uint32]string
		err               error
	}

	testCases := []struct {
		name                      string
		providerConfigs           []endpointProviderConfig
		expectedPortToProtocolMap map[uint32]string
		expectError               bool
	}{
		{
			// Test case 1
			name: "multiple providers correctly returning the same port:protocol mapping",
			providerConfigs: []endpointProviderConfig{
				{
					// provider 1
					provider:          endpoint.NewMockProvider(mockCtrl),
					portToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
					err:               nil,
				},
				{
					// provider 2
					provider:          endpoint.NewMockProvider(mockCtrl),
					portToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
					err:               nil,
				},
			},
			expectedPortToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
			expectError:               false,
		},

		{
			// Test case 2
			name: "multiple providers incorrectly returning different port:protocol mapping",
			providerConfigs: []endpointProviderConfig{
				{
					// provider 1
					provider:          endpoint.NewMockProvider(mockCtrl),
					portToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
					err:               nil,
				},
				{
					// provider 2
					provider:          endpoint.NewMockProvider(mockCtrl),
					portToProtocolMap: map[uint32]string{80: "tcp", 90: "http"},
					err:               nil,
				},
			},
			expectedPortToProtocolMap: nil,
			expectError:               true,
		},

		{
			// Test case 3
			name: "single provider correctly returning port:protocol mapping",
			providerConfigs: []endpointProviderConfig{
				{
					// provider 1
					provider:          endpoint.NewMockProvider(mockCtrl),
					portToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
					err:               nil,
				},
			},
			expectedPortToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
			expectError:               false,
		},
	}

	testSvc := service.MeshService{Name: "foo", Namespace: "bar"}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			// Create a list of providers for catalog and mock their calls based on the given config
			var allProviders []endpoint.Provider
			for _, providerConfig := range tc.providerConfigs {
				allProviders = append(allProviders, providerConfig.provider)
				providerConfig.provider.EXPECT().GetTargetPortToProtocolMappingForService(testSvc).Return(providerConfig.portToProtocolMap, providerConfig.err).Times(1)
			}

			mc := &MeshCatalog{
				endpointsProviders: allProviders,
			}

			actualPortToProtocolMap, err := mc.GetTargetPortToProtocolMappingForService(testSvc)

			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedPortToProtocolMap, actualPortToProtocolMap)
		})
	}
}

func TestGetPortToProtocolMappingForResolvableService(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	svc := service.MeshService{
		Namespace: "foo",
		Name:      "bar",
	}
	appProtocolTCP := "tcp"
	appProtocolHTTP := "http"

	testCases := []struct {
		name                      string
		service                   *corev1.Service
		expectedPortToProtocolMap map[uint32]string
		expectError               bool
	}{
		{
			// Test case 1
			name: "service with no appProtocol specified",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      svc.Name,
					Namespace: svc.Namespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "port1",
							TargetPort: intstr.IntOrString{
								Type:   intstr.String,
								IntVal: 8080,
							},
							Port: 80,
						},
						{
							Name: "port2",
							TargetPort: intstr.IntOrString{
								Type:   intstr.String,
								IntVal: 9090,
							},
							Protocol: corev1.ProtocolTCP,
							Port:     90,
						},
					},
				},
			},
			expectedPortToProtocolMap: map[uint32]string{80: "http", 90: "http"},
			expectError:               false,
		},

		{
			// Test case 2
			name: "service with appProtocol specified",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      svc.Name,
					Namespace: svc.Namespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "port1",
							TargetPort: intstr.IntOrString{
								Type:   intstr.String,
								IntVal: 8080,
							},
							AppProtocol: &appProtocolHTTP,
							Port:        80,
						},
						{
							Name: "port2",
							TargetPort: intstr.IntOrString{
								Type:   intstr.String,
								IntVal: 9090,
							},
							Port:        90,
							AppProtocol: &appProtocolTCP,
						},
					},
				},
			},
			expectedPortToProtocolMap: map[uint32]string{80: "http", 90: "tcp"},
			expectError:               false,
		},

		{
			// Test case 3
			name: "service with appProtocol and named port specified",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      svc.Name,
					Namespace: svc.Namespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "http-port1",
							TargetPort: intstr.IntOrString{
								Type:   intstr.String,
								IntVal: 8080,
							},
							AppProtocol: &appProtocolTCP, // takes precedence over 'Name'
							Port:        80,
						},
					},
				},
			},
			expectedPortToProtocolMap: map[uint32]string{80: "tcp"},
			expectError:               false,
		},

		{
			// Test case 4
			name:                      "service doesn't exist",
			service:                   nil,
			expectedPortToProtocolMap: nil,
			expectError:               true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockKubeController := kubernetes.NewMockController(mockCtrl)
			mc := &MeshCatalog{
				kubeController: mockKubeController,
			}

			mockKubeController.EXPECT().GetService(svc).Return(tc.service).Times(1)

			actualPortToProtocolMap, err := mc.GetPortToProtocolMappingForService(svc)

			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedPortToProtocolMap, actualPortToProtocolMap)
		})
	}
}
