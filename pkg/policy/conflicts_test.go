package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
)

func TestDetectIngressBackendConflicts(t *testing.T) {
	testCases := []struct {
		name              string
		x                 policyv1alpha1.IngressBackend
		y                 policyv1alpha1.IngressBackend
		conflictsExpected int
	}{
		{
			name: "single backend conflict",
			x: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			y: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-2",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			conflictsExpected: 1,
		},
		{
			name: "Unique backends per policy",
			x: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			y: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-2",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend2",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			conflictsExpected: 0,
		},
		{
			name: "multiple backends conflict",
			x: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
						{
							Name: "backend2",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			y: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-2",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
						{
							Name: "backend2",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			conflictsExpected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			conflicts := DetectIngressBackendConflicts(tc.x, tc.y)
			a.Len(conflicts, tc.conflictsExpected)
		})
	}
}
