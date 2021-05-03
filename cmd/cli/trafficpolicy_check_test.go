package main

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	fakeAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned/fake"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
)

func TestUnmarshalNamespacedPod(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		namespacedPod     string
		expectedNamespace string
		expectedPodName   string
		expectError       bool // true if error is expected, false otherwise
	}{
		{"foo/bar", "foo", "bar", false},
		{"foo", metav1.NamespaceDefault, "foo", false},
		{"", "", "", true},
		{"foo/bar/baz", "", "", true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing if %s", tc.namespacedPod), func(t *testing.T) {
			actualNamespace, actualPodName, err := unmarshalNamespacedPod(tc.namespacedPod)
			assert.Equal(actualNamespace, tc.expectedNamespace)
			assert.Equal(actualPodName, tc.expectedPodName)
			assert.Equal(err != nil, tc.expectError)
		})
	}
}

func TestIsPermissiveModeEnabled(t *testing.T) {
	assert := tassert.New(t)
	fakeClient := fake.NewSimpleClientset()
	out := new(bytes.Buffer)

	cmd := trafficPolicyCheckCmd{
		clientSet: fakeClient,
		out:       out,
	}

	testCases := []struct {
		configMap   corev1.ConfigMap
		enabled     bool
		expectError bool
	}{
		{
			corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "osm-system",
					Name:      osmConfigMapName,
				},
				Data: map[string]string{
					configurator.PermissiveTrafficPolicyModeKey: "true",
				},
			},
			true,
			false,
		},
		{
			corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "osm-system",
					Name:      osmConfigMapName,
				},
				Data: map[string]string{
					configurator.PermissiveTrafficPolicyModeKey: "false",
				},
			},
			false,
			false,
		},
		{
			corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "osm-system",
					Name:      osmConfigMapName,
				},
				Data: map[string]string{
					configurator.PermissiveTrafficPolicyModeKey: "invalid-value",
				},
			},
			false,
			true,
		},
	}

	// Create the test namespace
	osmNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-system",
		},
	}
	_, err := fakeClient.CoreV1().Namespaces().Create(context.TODO(), osmNamespace, metav1.CreateOptions{})
	assert.Nil(err)

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing testcase %d", i), func(t *testing.T) {
			// create the configMap
			_, err := fakeClient.CoreV1().ConfigMaps(osmNamespace.Name).Create(context.TODO(), &tc.configMap, metav1.CreateOptions{})
			assert.Nil(err)

			enabled, err := cmd.isPermissiveModeEnabled()
			assert.Equal(enabled, tc.enabled)
			assert.Equal(err != nil, tc.expectError)

			// delete the configMap for the next test case using the same ConfigMap
			err = fakeClient.CoreV1().ConfigMaps(osmNamespace.Name).Delete(context.TODO(), tc.configMap.Name, metav1.DeleteOptions{})
			assert.Nil(err)
		})
	}
}

func TestCheckTrafficPolicy(t *testing.T) {
	assert := tassert.New(t)
	fakeClient := fake.NewSimpleClientset()
	fakeAccessClient := fakeAccessClient.NewSimpleClientset()
	out := new(bytes.Buffer)

	cmd := trafficPolicyCheckCmd{
		clientSet:       fakeClient,
		smiAccessClient: fakeAccessClient,
		out:             out,
	}

	testCases := []struct {
		srcPod            corev1.Pod
		dstPod            corev1.Pod
		trafficTarget     smiAccess.TrafficTarget
		configMap         corev1.ConfigMap
		expectError       bool
		expectedOutSubstr string
	}{
		// first test case: source and destination are allowed by SMI TrafficTarget
		{
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "ns-1",
					Labels:    map[string]string{constants.EnvoyUniqueIDLabelName: "test"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa-1",
				},
			},
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-2",
					Namespace: "ns-2",
					Labels:    map[string]string{constants.EnvoyUniqueIDLabelName: "test"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa-2",
				},
			},
			smiAccess.TrafficTarget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "access.smi-spec.io/v1alpha3",
					Kind:       "TrafficTarget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: "ns-2",
				},
				Spec: smiAccess.TrafficTargetSpec{
					Destination: smiAccess.IdentityBindingSubject{
						Kind:      "ServiceAccount",
						Name:      "sa-2",
						Namespace: "ns-2",
					},
					Sources: []smiAccess.IdentityBindingSubject{{
						Kind:      "ServiceAccount",
						Name:      "sa-1",
						Namespace: "ns-1",
					}},
				},
			},
			corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "osm-system",
					Name:      osmConfigMapName,
				},
				Data: map[string]string{
					configurator.PermissiveTrafficPolicyModeKey: "false",
				},
			},
			false,
			"is allowed to communicate",
		},
		// second test case: source and destination are not allowed by SMI TrafficTarget
		{
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "ns-1",
					Labels:    map[string]string{constants.EnvoyUniqueIDLabelName: "test"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa-1",
				},
			},
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-3",
					Namespace: "ns-3",
					Labels:    map[string]string{constants.EnvoyUniqueIDLabelName: "test"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa-3",
				},
			},
			smiAccess.TrafficTarget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "access.smi-spec.io/v1alpha3",
					Kind:       "TrafficTarget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: "ns-2",
				},
				Spec: smiAccess.TrafficTargetSpec{
					Destination: smiAccess.IdentityBindingSubject{
						Kind:      "ServiceAccount",
						Name:      "sa-2",
						Namespace: "ns-2",
					},
					Sources: []smiAccess.IdentityBindingSubject{{
						Kind:      "ServiceAccount",
						Name:      "sa-1",
						Namespace: "ns-1",
					}},
				},
			},
			corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "osm-system",
					Name:      osmConfigMapName,
				},
				Data: map[string]string{
					configurator.PermissiveTrafficPolicyModeKey: "false",
				},
			},
			false,
			"is not allowed to communicate",
		},

		// third test case: source and destination are allowed due to permissive mode
		{
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "ns-1",
					Labels:    map[string]string{constants.EnvoyUniqueIDLabelName: "test"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa-1",
				},
			},
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-3",
					Namespace: "ns-3",
					Labels:    map[string]string{constants.EnvoyUniqueIDLabelName: "test"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa-3",
				},
			},
			smiAccess.TrafficTarget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "access.smi-spec.io/v1alpha3",
					Kind:       "TrafficTarget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: "ns-2",
				},
				Spec: smiAccess.TrafficTargetSpec{
					Destination: smiAccess.IdentityBindingSubject{
						Kind:      "ServiceAccount",
						Name:      "sa-2",
						Namespace: "ns-2",
					},
					Sources: []smiAccess.IdentityBindingSubject{{
						Kind:      "ServiceAccount",
						Name:      "sa-1",
						Namespace: "ns-1",
					}},
				},
			},
			corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "osm-system",
					Name:      osmConfigMapName,
				},
				Data: map[string]string{
					configurator.PermissiveTrafficPolicyModeKey: "true",
				},
			},
			false,
			"is allowed to communicate",
		},
	}

	// Create OSM namespace
	osmNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-system",
		},
	}
	_, err := fakeClient.CoreV1().Namespaces().Create(context.TODO(), osmNamespace, metav1.CreateOptions{})
	assert.Nil(err)

	{
		// Create test namespace ns-1
		ns1 := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-1",
			},
		}
		_, err := fakeClient.CoreV1().Namespaces().Create(context.TODO(), ns1, metav1.CreateOptions{})
		assert.Nil(err)
	}

	{
		// Create test namespace ns-2
		ns2 := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		}
		_, err := fakeClient.CoreV1().Namespaces().Create(context.TODO(), ns2, metav1.CreateOptions{})
		assert.Nil(err)
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing testcase %d", i), func(t *testing.T) {
			// create the configMap
			_, err := fakeClient.CoreV1().ConfigMaps(osmNamespace.Name).Create(context.TODO(), &tc.configMap, metav1.CreateOptions{})
			assert.Nil(err)

			// create the traffic target
			_, err = fakeAccessClient.AccessV1alpha3().TrafficTargets(tc.trafficTarget.Namespace).Create(context.TODO(), &tc.trafficTarget, metav1.CreateOptions{})
			assert.Nil(err)

			err = cmd.checkTrafficPolicy(&tc.srcPod, &tc.dstPod)
			assert.Equal(err != nil, tc.expectError)
			assert.Contains(out.String(), tc.expectedOutSubstr)

			// delete the ConfigMap for the next test case using the same ConfigMap
			err = fakeClient.CoreV1().ConfigMaps(osmNamespace.Name).Delete(context.TODO(), tc.configMap.Name, metav1.DeleteOptions{})
			assert.Nil(err)

			// delete the TrafficTarget for the next test case
			err = fakeAccessClient.AccessV1alpha3().TrafficTargets(tc.trafficTarget.Namespace).Delete(context.TODO(), tc.trafficTarget.Name, metav1.DeleteOptions{})
			assert.Nil(err)
		})
	}
}
