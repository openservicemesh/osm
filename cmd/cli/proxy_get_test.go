package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"os"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestIsMeshedPod(t *testing.T) {
	assert := tassert.New(t)

	type test struct {
		pod      corev1.Pod
		isMeshed bool
	}

	testCases := []test{
		{
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "pod-1",
					Labels: map[string]string{constants.EnvoyUniqueIDLabelName: "test"},
				},
			},
			isMeshed: true,
		},
		{
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-2",
				},
			},
			isMeshed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing if pod %s is meshed", tc.pod.Name), func(t *testing.T) {
			isMeshed := isMeshedPod(tc.pod)
			assert.Equal(isMeshed, tc.isMeshed)
		})
	}
}

func TestAnnotateErrMsgWithPodNamespaceMsg(t *testing.T) {
	assert := tassert.New(t)

	type test struct {
		errorMsg     string
		podName      string
		podNamespace string
		annotatedMsg string
	}

	podNamespaceActionableMsg := "Note: Use the flag --namespace to modify the intended pod namespace."

	testCases := []test{
		{
			"Proxy get command error for pod name [%s] in pod namespace [%s]",
			"test-pod-name",
			"test-namespace",
			"Proxy get command error for pod name [test-pod-name] in pod namespace [test-namespace]\n\n" + podNamespaceActionableMsg,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing annotated error message for pod name [%s] in pod namespace [%s]", tc.podName, tc.podNamespace), func(t *testing.T) {
			assert.Equal(
				tc.annotatedMsg,
				annotateErrMsgWithPodNamespaceMsg(tc.errorMsg, tc.podName, tc.podNamespace).Error())
		})
	}
}

func TestRunProxyGet(t *testing.T) {
	assert := tassert.New(t)
	fakeK8sClient := fake.NewSimpleClientset()
	fakeConfigClient := fakeConfig.NewSimpleClientset()
	out := new(bytes.Buffer)
	port := uint16(8080) // TODO: change this
	sigintChan := make(chan os.Signal, 1)
	query:= "config_dump"

	cmd := proxyGetCmd{
		out: out,
		clientSet: fakeK8sClient,
		query : query,
		localPort: port,
		sigintChan: sigintChan,
	}

	testCases := []struct {
		meshedNamespace string
		pod corev1.Pod
		createPod bool
		expectError bool
	}{
		// first test case: pod not in namespace
		{
			"mesh-ns-1",
			corev1.Pod{},
			false,
			true,
		},

		// second test case: pod exists in namespace but is not in mesh i.e., no Envoy UID Label
		{
			"mesh-ns-1",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-1",
					Namespace: "mesh-ns-1",
				},
			},
			true,
			true,
		},
		// third test case: run does not err and proxyGet returns nil
		{
			"mesh-ns-1",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mesh-pod-1",
					Namespace: "mesh-ns-1",
					Labels:    map[string]string{constants.EnvoyUniqueIDLabelName: "test"},
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

	// Create meshed namespace
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh-ns-1",
		},
	}

	_, err = fakeK8sClient.CoreV1().Namespaces().Create(context.TODO(), ns1, metav1.CreateOptions{})
	assert.Nil(err)

	// Create MeshConfig
	meshConfig := &v1alpha1.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "osm-system",
			Name:      defaultOsmMeshConfigName,
		},
	}

	_, err = fakeConfigClient.ConfigV1alpha1().MeshConfigs(osmNamespace.Name).Create(context.TODO(), meshConfig, metav1.CreateOptions{})
	assert.Nil(err)

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing testcase %d", i), func(t *testing.T) {
			cmd.namespace = tc.meshedNamespace

			// create pods
			if tc.createPod {
				_, err = fakeK8sClient.CoreV1().Pods(ns1.Name).Create(context.TODO(), &tc.pod, metav1.CreateOptions{})
				assert.Nil(err)
			}

			err = cmd.run()
			assert.Equal(tc.expectError, err != nil)

			// delete pods for next test case
			if tc.createPod {
				err = fakeK8sClient.CoreV1().Pods(ns1.Name).Delete(context.TODO(), tc.pod.Name, metav1.DeleteOptions{})
				assert.Nil(err)
			}
		})
	}

}
