package k8s

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	fakeclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetHostnamesForServicePort(t *testing.T) {
	testCases := []struct {
		name              string
		service           service.MeshService
		localNamespace    bool
		expectedHostnames []string
	}{
		{
			name:           "hostnames corresponding to a service in the same namespace",
			service:        service.MeshService{Namespace: "ns1", Name: "s1", Port: 90},
			localNamespace: true,
			expectedHostnames: []string{
				"s1",
				"s1:90",
				"s1.ns1",
				"s1.ns1:90",
				"s1.ns1.svc",
				"s1.ns1.svc:90",
				"s1.ns1.svc.cluster",
				"s1.ns1.svc.cluster:90",
				"s1.ns1.svc.cluster.local",
				"s1.ns1.svc.cluster.local:90",
			},
		},
		{
			name:           "hostnames corresponding to a service in different namespace",
			service:        service.MeshService{Namespace: "ns1", Name: "s1", Port: 90},
			localNamespace: false,
			expectedHostnames: []string{
				"s1.ns1",
				"s1.ns1:90",
				"s1.ns1.svc",
				"s1.ns1.svc:90",
				"s1.ns1.svc.cluster",
				"s1.ns1.svc.cluster:90",
				"s1.ns1.svc.cluster.local",
				"s1.ns1.svc.cluster.local:90",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := GetHostnamesForService(tc.service, tc.localNamespace)
			assert.ElementsMatch(actual, tc.expectedHostnames)
			assert.Len(actual, len(tc.expectedHostnames))
		})
	}
}

func TestGetServiceFromHostname(t *testing.T) {
	testCases := []struct {
		name            string
		hostnames       []string
		expectedService string
		withController  bool
		namespaceFound  bool
	}{
		{
			name: "gets the service name from hostname",
			hostnames: []string{
				tests.BookbuyerServiceName,
				fmt.Sprintf("%s:%d", tests.BookbuyerServiceName, tests.ServicePort),
				fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
				fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
				fmt.Sprintf("%s.%s.svc.cluster.local", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
			},
			expectedService: tests.BookbuyerServiceName,
		},
		{
			name: "distinguishes ambiguous hostname using controller (namespace not found)",
			hostnames: []string{
				fmt.Sprintf("my-subdomain.%s", tests.BookbuyerServiceName),
				fmt.Sprintf("my-subdomain.%s:%d", tests.BookbuyerServiceName, tests.ServicePort),
				fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
			},
			withController:  true,
			expectedService: tests.BookbuyerServiceName,
		},
		{
			name: "distinguishes ambiguous hostname using controller (namespace found)",
			hostnames: []string{
				fmt.Sprintf("my-subdomain.%s.%s", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("my-subdomain.%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
				fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
			},
			withController:  true,
			namespaceFound:  true,
			expectedService: tests.BookbuyerServiceName,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			var c Controller
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			if tc.withController {
				mockController := NewMockController(mockCtrl)

				var ns *corev1.Namespace

				if tc.namespaceFound {
					ns = &corev1.Namespace{}
				}
				mockController.EXPECT().GetNamespace(gomock.Any()).Return(ns).AnyTimes()
				c = mockController
			}

			for _, hostname := range tc.hostnames {
				actual := GetServiceFromHostname(c, hostname)
				assert.Equal(tc.expectedService, actual, "Hostname: %s", hostname)
			}
		})
	}
}

func TestGetKubernetesServerVersionNumber(t *testing.T) {
	testCases := []struct {
		name            string
		kubeClient      kubernetes.Interface
		version         string
		expectedVersion []int // of the form [1, 19, 0] for v1.19.0
		expectError     bool
	}{
		{
			name:            "invalid kubeClient should error",
			kubeClient:      nil,
			expectedVersion: nil,
			expectError:     true,
		},
		{
			name:            "invalid server version should error",
			kubeClient:      fakeclient.NewSimpleClientset(),
			version:         "foo",
			expectedVersion: nil,
			expectError:     true,
		},
		{
			name:            "valid server version",
			kubeClient:      fakeclient.NewSimpleClientset(),
			version:         "v1.19.0",
			expectedVersion: []int{1, 19, 0},
			expectError:     false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			if tc.kubeClient != nil {
				tc.kubeClient.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{
					GitVersion: tc.version,
				}
			}

			version, err := GetKubernetesServerVersionNumber(tc.kubeClient)
			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedVersion, version)
		})
	}
}

func TestNamespacedNameFrom(t *testing.T) {
	testCases := []struct {
		name      string
		in        string
		out       types.NamespacedName
		expectErr bool
	}{
		{
			name:      "valid namespaced name",
			in:        "foo/bar",
			out:       types.NamespacedName{Namespace: "foo", Name: "bar"},
			expectErr: false,
		},
		{
			name:      "invalid namespaced name with no separator",
			in:        "foobar",
			out:       types.NamespacedName{},
			expectErr: true,
		},
		{
			name:      "invalid namespaced name with multiple separators",
			in:        "foo/bar/baz",
			out:       types.NamespacedName{},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual, err := NamespacedNameFrom(tc.in)
			assert.Equal(tc.out, actual)
			assert.Equal(tc.expectErr, err != nil)
		})
	}
}

func TestGetSubdomainFromHostname(t *testing.T) {
	testCases := []struct {
		name              string
		hostnames         []string
		expectedSubdomain string
		withController    bool
		foundNamespace    bool
	}{
		{
			name: "gets the subdomain from hostname (subdomain=my-subdomain)",
			hostnames: []string{
				fmt.Sprintf("my-subdomain.%s.%s", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("my-subdomain.%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
				fmt.Sprintf("my-subdomain.%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("my-subdomain.%s.%s.svc:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
				fmt.Sprintf("my-subdomain.%s.%s.svc.cluster.local", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("my-subdomain.%s.%s.svc.cluster.local:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
			},
			expectedSubdomain: "my-subdomain",
		},
		{
			name: "gets the subdomain from hostname (empty subdomain)",
			hostnames: []string{
				tests.BookbuyerServiceName,
				fmt.Sprintf("%s:%d", tests.BookbuyerServiceName, tests.ServicePort),
				fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
				fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
				fmt.Sprintf("%s.%s.svc.cluster.local", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
			},
			expectedSubdomain: "",
		},
		{
			name: "distinguishes ambiguous hostname using controller (namespace not found)",
			hostnames: []string{
				fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
			},
			withController:    true,
			expectedSubdomain: "",
		},
		{
			name: "distinguishes ambiguous hostname using controller (namespace not found)",
			hostnames: []string{
				fmt.Sprintf("my-subdomain.%s", tests.BookbuyerServiceName),
				fmt.Sprintf("my-subdomain.%s:%d", tests.BookbuyerServiceName, tests.ServicePort),
			},
			withController:    true,
			expectedSubdomain: "my-subdomain",
		},
		{
			name: "distinguishes ambiguous hostname using controller (namespace found)",
			hostnames: []string{
				fmt.Sprintf("my-subdomain.%s.%s", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("my-subdomain.%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
			},
			withController:    true,
			foundNamespace:    true,
			expectedSubdomain: "my-subdomain",
		},
		{
			name: "distinguishes ambiguous hostname using controller (namespace found)",
			hostnames: []string{
				fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s:%d", tests.BookbuyerServiceName, tests.Namespace, tests.ServicePort),
			},
			withController:    true,
			foundNamespace:    true,
			expectedSubdomain: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			var c Controller
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			if tc.withController {
				mockController := NewMockController(mockCtrl)

				var ns *corev1.Namespace

				if tc.foundNamespace {
					ns = &corev1.Namespace{}
				}

				mockController.EXPECT().GetNamespace(gomock.Any()).Return(ns).AnyTimes()
				c = mockController
			}

			for _, hostname := range tc.hostnames {
				actual := GetSubdomainFromHostname(c, hostname)
				assert.Equal(tc.expectedSubdomain, actual, "Hostname: %s", hostname)
			}
		})
	}
}
