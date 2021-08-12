package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	fakePolicyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
)

func TestPolicyCheckConflictRun(t *testing.T) {
	testNs := "test"

	testCases := []struct {
		name                  string
		resourceKind          string
		namespaces            []string
		existingResources     []runtime.Object
		expectErr             bool
		expectedRegexMatchOut string
	}{
		{
			name:         "No conflicts among IngressBackend resources",
			resourceKind: "IngressBackend",
			namespaces:   []string{testNs},
			existingResources: []runtime.Object{
				&policyv1alpha1.IngressBackend{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-1",
						Namespace: testNs,
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
				&policyv1alpha1.IngressBackend{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-2",
						Namespace: testNs,
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
			},
			expectErr:             false,
			expectedRegexMatchOut: "No conflicts among IngressBackend resources in namespace",
		},
		{
			name:         "Conflicts among IngressBackend resources",
			resourceKind: "IngressBackend",
			namespaces:   []string{testNs},
			existingResources: []runtime.Object{
				&policyv1alpha1.IngressBackend{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-1",
						Namespace: testNs,
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
				&policyv1alpha1.IngressBackend{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-2",
						Namespace: testNs,
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
			},
			expectErr:             false,
			expectedRegexMatchOut: "IngressBackend .* conflicts with .*",
		},
		{
			name:              "Invalid resource kind results in an error",
			resourceKind:      "invalid",
			namespaces:        []string{testNs},
			existingResources: nil,
			expectErr:         true,
		},
		{
			name:         "Passing multiple namespaces is an invalid input for the IngressBackend conflict check",
			resourceKind: "IngressBackend",
			namespaces:   []string{testNs, "extra"},
			existingResources: []runtime.Object{
				&policyv1alpha1.IngressBackend{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-1",
						Namespace: testNs,
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
				&policyv1alpha1.IngressBackend{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-2",
						Namespace: testNs,
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
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			stdout := new(bytes.Buffer)

			cmd := &policyCheckConflictsCmd{
				stdout:       stdout,
				namespaces:   tc.namespaces,
				resourceKind: tc.resourceKind,
			}

			switch tc.resourceKind {
			case "IngressBackend":
				cmd.policyClient = fakePolicyClientset.NewSimpleClientset(tc.existingResources...)
			}

			err := cmd.run()
			a.Equal(tc.expectErr, err != nil, err)
			if err != nil {
				return
			}

			a.Regexp(tc.expectedRegexMatchOut, stdout.String())
		})
	}
}
