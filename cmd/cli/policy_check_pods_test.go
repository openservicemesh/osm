package main

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	fakeAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned/fake"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestUnmarshalNamespacedPod(t *testing.T) {
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
			assert := tassert.New(t)

			actualNamespace, actualPodName, err := unmarshalNamespacedPod(tc.namespacedPod)
			assert.Equal(actualNamespace, tc.expectedNamespace)
			assert.Equal(actualPodName, tc.expectedPodName)
			assert.Equal(err != nil, tc.expectError)
		})
	}
}

func TestIsPermissiveModeEnabled(t *testing.T) {
	assert := tassert.New(t)
	fakeK8sClient := fake.NewSimpleClientset()
	fakeConfigClient := fakeConfig.NewSimpleClientset()
	out := new(bytes.Buffer)

	// Create the test namespace
	osmNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-system",
		},
	}
	_, err := fakeK8sClient.CoreV1().Namespaces().Create(context.TODO(), osmNamespace, metav1.CreateOptions{})
	assert.Nil(err)

	cmd := trafficPolicyCheckCmd{
		clientSet:        fakeK8sClient,
		meshConfigClient: fakeConfigClient,
		out:              out,
	}

	testCases := []struct {
		meshConfig  configv1alpha3.MeshConfig
		enabled     bool
		expectError bool
	}{
		{
			configv1alpha3.MeshConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace.Name,
					Name:      defaultOsmMeshConfigName,
				},
				Spec: configv1alpha3.MeshConfigSpec{
					Traffic: configv1alpha3.TrafficSpec{
						EnablePermissiveTrafficPolicyMode: true,
					},
				},
			},
			true,
			false,
		},
		{
			configv1alpha3.MeshConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: osmNamespace.Name,
					Name:      defaultOsmMeshConfigName,
				},
				Spec: configv1alpha3.MeshConfigSpec{
					Traffic: configv1alpha3.TrafficSpec{
						EnablePermissiveTrafficPolicyMode: false,
					},
				},
			},
			false,
			false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing testcase %d", i), func(t *testing.T) {
			assert := tassert.New(t)

			// create the MeshConfig
			_, err := fakeConfigClient.ConfigV1alpha3().MeshConfigs(osmNamespace.Name).Create(context.TODO(), &tc.meshConfig, metav1.CreateOptions{})
			assert.Nil(err)

			enabled, err := cmd.isPermissiveModeEnabled()
			assert.Equal(err != nil, tc.expectError)
			assert.Equal(enabled, tc.enabled)

			// delete the MeshConfig for the next test case using the same MeshConfig
			err = fakeConfigClient.ConfigV1alpha2().MeshConfigs(osmNamespace.Name).Delete(context.TODO(), tc.meshConfig.Name, metav1.DeleteOptions{})
			assert.Nil(err)
		})
	}
}

func TestRun(t *testing.T) {
	assert := tassert.New(t)

	fakeK8sClient := fake.NewSimpleClientset()
	fakeAccessClient := fakeAccess.NewSimpleClientset()
	fakeConfigClient := fakeConfig.NewSimpleClientset()
	out := new(bytes.Buffer)

	cmd := trafficPolicyCheckCmd{
		clientSet:        fakeK8sClient,
		smiAccessClient:  fakeAccessClient,
		meshConfigClient: fakeConfigClient,
		out:              out,
	}

	testCases := []struct {
		srcNamespacedPod string
		dstNamespacedPod string
		srcPod           corev1.Pod
		dstPod           corev1.Pod
		createPods       bool
		expectError      bool
	}{
		// first test case: invalid namespaced pod name for src
		{
			"ns-1/pod-1/foo",
			"ns-2/pod-2",
			corev1.Pod{},
			corev1.Pod{},
			false,
			true,
		},
		// second test case: invalid namespaced pod name for dst
		{
			"ns-1/pod-1",
			"ns-2/pod-2/foo",
			corev1.Pod{},
			corev1.Pod{},
			false,
			true,
		},
		// third test case: pods do not exist
		{
			"ns-1/pod-1",
			"ns-2/pod-2",
			corev1.Pod{},
			corev1.Pod{},
			false,
			true,
		},
		// fourth test case: pods are not a part of the mesh
		{
			"ns-1/pod-1",
			"ns-2/pod-2",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "ns-1",
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa-1",
				},
			},
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-2",
					Namespace: "ns-2",
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa-2",
				},
			},
			true,
			true,
		},
		// fifth test case: run does not err and checkTrafficPolicy returns nil
		{
			"ns-1/pod-1",
			"ns-2/pod-2",
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
			true,
			false,
		},
	}

	// Create OSM namespace
	osmNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-system",
		},
	}
	_, err := fakeK8sClient.CoreV1().Namespaces().Create(context.TODO(), osmNamespace, metav1.CreateOptions{})
	assert.Nil(err)

	// Create src and dst namespaces
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns-1",
		},
	}
	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns-2",
		},
	}
	_, err = fakeK8sClient.CoreV1().Namespaces().Create(context.TODO(), ns1, metav1.CreateOptions{})
	assert.Nil(err)

	_, err = fakeK8sClient.CoreV1().Namespaces().Create(context.TODO(), ns2, metav1.CreateOptions{})
	assert.Nil(err)

	// Create service accounts
	sa1 := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sa-1",
		},
	}
	sa2 := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sa-2",
		},
	}
	_, err = fakeK8sClient.CoreV1().ServiceAccounts(ns1.Name).Create(context.TODO(), sa1, metav1.CreateOptions{})
	assert.Nil(err)
	_, err = fakeK8sClient.CoreV1().ServiceAccounts(ns2.Name).Create(context.TODO(), sa2, metav1.CreateOptions{})
	assert.Nil(err)

	// Create MeshConfig with permissive mode enabled
	meshConfig := &configv1alpha3.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "osm-system",
			Name:      defaultOsmMeshConfigName,
		},
		Spec: configv1alpha3.MeshConfigSpec{
			Traffic: configv1alpha3.TrafficSpec{
				EnablePermissiveTrafficPolicyMode: true,
			},
		},
	}

	_, err = fakeConfigClient.ConfigV1alpha3().MeshConfigs(osmNamespace.Name).Create(context.TODO(), meshConfig, metav1.CreateOptions{})
	assert.Nil(err)

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing testcase %d", i), func(t *testing.T) {
			assert := tassert.New(t)

			cmd.sourcePod = tc.srcNamespacedPod
			cmd.destinationPod = tc.dstNamespacedPod

			// create pods
			if tc.createPods {
				_, err = fakeK8sClient.CoreV1().Pods(ns1.Name).Create(context.TODO(), &tc.srcPod, metav1.CreateOptions{})
				assert.Nil(err)
				_, err = fakeK8sClient.CoreV1().Pods(ns2.Name).Create(context.TODO(), &tc.dstPod, metav1.CreateOptions{})
				assert.Nil(err)
			}

			err = cmd.run()
			assert.Equal(tc.expectError, err != nil)

			// delete pods for next test case
			if tc.createPods {
				err = fakeK8sClient.CoreV1().Pods(ns1.Name).Delete(context.TODO(), tc.srcPod.Name, metav1.DeleteOptions{})
				assert.Nil(err)
				err = fakeK8sClient.CoreV1().Pods(ns2.Name).Delete(context.TODO(), tc.dstPod.Name, metav1.DeleteOptions{})
				assert.Nil(err)
			}
		})
	}
}

func TestCheckTrafficPolicy(t *testing.T) {
	assert := tassert.New(t)
	fakeK8sClient := fake.NewSimpleClientset()
	fakeAccessClient := fakeAccess.NewSimpleClientset()
	fakeConfigClient := fakeConfig.NewSimpleClientset()

	out := new(bytes.Buffer)

	cmd := trafficPolicyCheckCmd{
		clientSet:        fakeK8sClient,
		smiAccessClient:  fakeAccessClient,
		meshConfigClient: fakeConfigClient,
		out:              out,
	}

	testCases := []struct {
		srcPod            corev1.Pod
		dstPod            corev1.Pod
		trafficTarget     smiAccess.TrafficTarget
		meshConfig        configv1alpha3.MeshConfig
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
			configv1alpha3.MeshConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "osm-system",
					Name:      defaultOsmMeshConfigName,
				},
				Spec: configv1alpha3.MeshConfigSpec{
					Traffic: configv1alpha3.TrafficSpec{
						EnablePermissiveTrafficPolicyMode: false,
					},
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
			configv1alpha3.MeshConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "osm-system",
					Name:      defaultOsmMeshConfigName,
				},
				Spec: configv1alpha3.MeshConfigSpec{
					Traffic: configv1alpha3.TrafficSpec{
						EnablePermissiveTrafficPolicyMode: false,
					},
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
			configv1alpha3.MeshConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "osm-system",
					Name:      defaultOsmMeshConfigName,
				},
				Spec: configv1alpha3.MeshConfigSpec{
					Traffic: configv1alpha3.TrafficSpec{
						EnablePermissiveTrafficPolicyMode: true,
					},
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
	_, err := fakeK8sClient.CoreV1().Namespaces().Create(context.TODO(), osmNamespace, metav1.CreateOptions{})
	assert.Nil(err)

	{
		// Create test namespace ns-1
		ns1 := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-1",
			},
		}
		_, err := fakeK8sClient.CoreV1().Namespaces().Create(context.TODO(), ns1, metav1.CreateOptions{})
		assert.Nil(err)
	}

	{
		// Create test namespace ns-2
		ns2 := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		}
		_, err := fakeK8sClient.CoreV1().Namespaces().Create(context.TODO(), ns2, metav1.CreateOptions{})
		assert.Nil(err)
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing testcase %d", i), func(t *testing.T) {
			assert := tassert.New(t)

			// create MeshConfig
			_, err := fakeConfigClient.ConfigV1alpha3().MeshConfigs(osmNamespace.Name).Create(context.TODO(), &tc.meshConfig, metav1.CreateOptions{})
			assert.Nil(err)

			// create the traffic target
			_, err = fakeAccessClient.AccessV1alpha3().TrafficTargets(tc.trafficTarget.Namespace).Create(context.TODO(), &tc.trafficTarget, metav1.CreateOptions{})
			assert.Nil(err)

			err = cmd.checkTrafficPolicy(&tc.srcPod, &tc.dstPod)
			assert.Equal(err != nil, tc.expectError)
			assert.Contains(out.String(), tc.expectedOutSubstr)

			// delete the MeshConfig for the next test case using the same MeshConfig
			err = fakeConfigClient.ConfigV1alpha3().MeshConfigs(osmNamespace.Name).Delete(context.TODO(), tc.meshConfig.Name, metav1.DeleteOptions{})
			assert.Nil(err)

			// delete the TrafficTarget for the next test case
			err = fakeAccessClient.AccessV1alpha3().TrafficTargets(tc.trafficTarget.Namespace).Delete(context.TODO(), tc.trafficTarget.Name, metav1.DeleteOptions{})
			assert.Nil(err)
		})
	}
}
