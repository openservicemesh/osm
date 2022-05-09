package service

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestServerName(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name     string
		service  MeshService
		expected string
	}{
		{
			name: "no subdomain",
			service: MeshService{
				Namespace: "namespace-here",
				Name:      "service-name-here",
			},
			expected: "service-name-here.namespace-here.svc.cluster.local",
		},
		{
			name: "subdomain",
			service: MeshService{
				Namespace: "namespace-here",
				Name:      "subdomain-0.service-name-here",
			},
			expected: "subdomain-0.service-name-here.namespace-here.svc.cluster.local",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(tt.expected, tt.service.ServerName())
		})
	}
}

func TestEquals(t *testing.T) {
	testCases := []struct {
		name           string
		service        MeshService
		anotherService MeshService
		isEqual        bool
	}{
		{
			name: "services are equal",
			service: MeshService{
				Namespace: "default",
				Name:      "bookbuyer",
			},
			anotherService: MeshService{
				Namespace: "default",
				Name:      "bookbuyer",
			},
			isEqual: true,
		},
		{
			name: "services are NOT equal",
			service: MeshService{
				Namespace: "default",
				Name:      "bookbuyer",
			},
			anotherService: MeshService{
				Namespace: "default",
				Name:      "bookstore",
			},
			isEqual: false,
		},
		{
			name: "works even when unexported fields aren't initialized",
			service: MeshService{
				Namespace: "default",
				Name:      "bookbuyer",
			},
			anotherService: MeshService{
				Namespace: "default",
				Name:      "bookbuyer",
			},
			isEqual: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			tc.service.Subdomain()                         // sets service.subdomain and subdomainPopulated (both unexported)
			tc.anotherService.ProviderKey()                // sets service.providerKey (unexported)
			actual := tc.service.Equals(tc.anotherService) // These should still be equal
			assert.Equal(actual, tc.isEqual)
		})
	}
}

func TestString(t *testing.T) {
	testCases := []struct {
		name          string
		service       MeshService
		serviceString string
	}{
		{
			name: "service in default namespace",
			service: MeshService{
				Namespace: "default",
				Name:      "bookbuyer",
			},
			serviceString: "default/bookbuyer",
		},
		{
			name: "service in custom namespace",
			service: MeshService{
				Namespace: "bookbuyer-ns",
				Name:      "bookbuyer",
			},
			serviceString: "bookbuyer-ns/bookbuyer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := tc.service.String()
			assert.Equal(actual, tc.serviceString)
		})
	}
}
