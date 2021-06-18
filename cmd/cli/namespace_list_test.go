package main

import (
	"bytes"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestNamespaceList(t *testing.T) {
	tests := []struct {
		name       string
		meshName   string
		namespaces []*corev1.Namespace
		expected   string
	}{
		{
			name:     "no namespaces no mesh specified",
			expected: "No namespaces in any mesh\n",
		},
		{
			name:     "no namespaces with mesh specified",
			meshName: "my-mesh",
			expected: "No namespaces in mesh [my-mesh]\n",
		},
		{
			name: "one namespace injection not set",
			namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "my-mesh",
						},
					},
				},
			},
			expected: "NAMESPACE\tMESH\tSIDECAR-INJECTION\nns\tmy-mesh\t-\n",
		},
		{
			name: "one namespace injection enabled",
			namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "my-mesh",
						},
						Annotations: map[string]string{
							constants.SidecarInjectionAnnotation: "enabled",
						},
					},
				},
			},
			expected: "NAMESPACE\tMESH\tSIDECAR-INJECTION\nns\tmy-mesh\tenabled\n",
		},
		{
			name: "one namespace injection ignored",
			namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "my-mesh",
							constants.IgnoreLabel:                      "any value",
						},
						Annotations: map[string]string{
							constants.SidecarInjectionAnnotation: "enabled",
						},
					},
				},
			},
			expected: "NAMESPACE\tMESH\tSIDECAR-INJECTION\nns\tmy-mesh\tdisabled (ignored)\n",
		},
		{
			name: "two namespaces different meshes no mesh specified",
			namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns1",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "my-mesh1",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns2",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "my-mesh2",
						},
					},
				},
			},
			expected: "NAMESPACE\tMESH\tSIDECAR-INJECTION\nns1\tmy-mesh1\t-\nns2\tmy-mesh2\t-\n",
		},
		{
			name:     "two namespaces different meshes with mesh specified",
			meshName: "my-mesh2",
			namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns1",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "my-mesh1",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns2",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "my-mesh2",
						},
					},
				},
			},
			expected: "NAMESPACE\tMESH\tSIDECAR-INJECTION\nns2\tmy-mesh2\t-\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)

			buf := bytes.NewBuffer(nil)

			objs := make([]runtime.Object, len(test.namespaces))
			for i := range test.namespaces {
				objs[i] = test.namespaces[i]
			}

			cmd := namespaceListCmd{
				out:       buf,
				meshName:  test.meshName,
				clientSet: fake.NewSimpleClientset(objs...),
			}

			assert.Nil(cmd.run())

			expected := bytes.NewBuffer(nil)
			expTw := newTabWriter(expected)
			_, err := expTw.Write([]byte(test.expected))
			assert.Nil(err)
			assert.Nil(expTw.Flush())

			assert.Equal(expected.String(), buf.String())
		})
	}
}
