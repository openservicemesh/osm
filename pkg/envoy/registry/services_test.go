package registry

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestKubernetesServicesToMeshServices(t *testing.T) {
	testCases := []struct {
		name                 string
		k8sServices          []*v1.Service
		k8sEndpoints         *v1.Endpoints
		expectedMeshServices []service.MeshService
		subdomainFilter      string
	}{
		{
			name: "k8s services to mesh services",
			k8sServices: []*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "10.0.0.1",
						Ports: []v1.ServicePort{{
							Name: "p1",
							Port: 80,
						}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns2",
						Name:      "s2",
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "10.0.0.1",
						Ports: []v1.ServicePort{{
							Name: "p2",
							Port: 80,
						}},
					},
				},
			},
			expectedMeshServices: []service.MeshService{
				{
					Namespace: "ns1",
					Name:      "s1",
					Protocol:  "http",
					Port:      80,
				},
				{
					Namespace: "ns2",
					Name:      "s2",
					Protocol:  "http",
					Port:      80,
				},
			},
		},
		{
			name: "k8s services to mesh services (subdomain filter)",
			k8sServices: []*v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1-headless",
					},
					Spec: v1.ServiceSpec{
						Ports: []v1.ServicePort{{
							Name: "p1",
							Port: 80,
						}},
					},
				},
			},
			k8sEndpoints: &v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1-headless",
					Namespace: "ns1",
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{
								IP:       "8.8.8.8",
								Hostname: "pod-0",
							},
							{
								IP:       "8.8.8.8",
								Hostname: "pod-1",
							},
						},
					},
				},
			},
			subdomainFilter: "pod-1",
			expectedMeshServices: []service.MeshService{
				{
					Namespace: "ns1",
					Name:      "pod-1.s1-headless",
					Protocol:  "http",
					Port:      80,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			stop := make(chan struct{})
			defer close(stop)
			var objs []runtime.Object
			for _, svc := range tc.k8sServices {
				objs = append(objs, svc)
			}
			if tc.k8sEndpoints != nil {
				objs = append(objs, tc.k8sEndpoints)
			}

			ic, err := informers.NewInformerCollection("test", stop, informers.WithKubeClient(testclient.NewSimpleClientset(objs...)))
			assert.NoError(err)
			k8sClient := k8s.NewClient("ns", tests.OsmMeshConfigName, ic, nil, nil)

			actual := kubernetesServicesToMeshServices(k8sClient, tc.k8sServices, tc.subdomainFilter)
			assert.ElementsMatch(tc.expectedMeshServices, actual)
		})
	}
}
