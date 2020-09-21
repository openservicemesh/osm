package main

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	testMesh = "osm"
)

func newNamespace(name string, annotations map[string]string) *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: testMesh},
		},
	}

	if annotations != nil {
		ns.Annotations = annotations
	}

	return ns
}

func createFakeController(fakeClient kubernetes.Interface) error {
	controllerNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-system",
		},
	}

	if _, err := fakeClient.CoreV1().Namespaces().Create(context.TODO(), controllerNs, metav1.CreateOptions{}); err != nil {
		return err
	}

	controllerDep := createDeployment("test-controller", testMesh, true)
	if _, err := fakeClient.AppsV1().Deployments(controllerNs.Name).Create(context.TODO(), controllerDep, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func TestRun_MetricsEnable(t *testing.T) {
	assert := assert.New(t)
	fakeClient := fake.NewSimpleClientset()

	err := createFakeController(fakeClient)
	assert.Nil(err)

	type test struct {
		cmd           *metricsEnableCmd
		nsAnnotations map[string]string
	}

	testScenarios := []test{
		{
			cmd: &metricsEnableCmd{
				out:        new(bytes.Buffer),
				namespaces: []string{"ns-1"},
				clientSet:  fakeClient,
			},
			nsAnnotations: nil,
		},
		{
			cmd: &metricsEnableCmd{
				out:        new(bytes.Buffer),
				namespaces: []string{"ns-2", "ns-3"},
				clientSet:  fakeClient,
			},
			nsAnnotations: map[string]string{constants.MetricsAnnotation: "enabled"},
		},
	}

	for _, scenario := range testScenarios {
		// Create test fixrures for test scenario
		for _, ns := range scenario.cmd.namespaces {
			newNs := newNamespace(ns, scenario.nsAnnotations)
			ns, _ := fakeClient.CoreV1().Namespaces().Create(context.TODO(), newNs, metav1.CreateOptions{})
			assert.NotNil(ns)
		}

		err := scenario.cmd.run()
		assert.Nil(err)

		// Test expectation for scenario
		for _, ns := range scenario.cmd.namespaces {
			nsWithMetrics, _ := fakeClient.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
			assert.NotNil(nsWithMetrics)
			assert.Equal(nsWithMetrics.Annotations[constants.MetricsAnnotation], "enabled")
		}
	}
}

func TestRun_MetricsDisable(t *testing.T) {
	assert := assert.New(t)
	fakeClient := fake.NewSimpleClientset()

	err := createFakeController(fakeClient)
	assert.Nil(err)

	type test struct {
		cmd           *metricsDisableCmd
		nsAnnotations map[string]string
	}

	testScenarios := []test{
		{
			cmd: &metricsDisableCmd{
				out:        new(bytes.Buffer),
				namespaces: []string{"ns-1"},
				clientSet:  fakeClient,
			},
			nsAnnotations: map[string]string{constants.MetricsAnnotation: "enabled"},
		},
		{
			cmd: &metricsDisableCmd{
				out:        new(bytes.Buffer),
				namespaces: []string{"ns-2", "ns-3"},
				clientSet:  fakeClient,
			},
			nsAnnotations: map[string]string{constants.MetricsAnnotation: "enabled"},
		},
	}

	for _, scenario := range testScenarios {
		// Create test fixrures for test scenario
		for _, ns := range scenario.cmd.namespaces {
			newNs := newNamespace(ns, scenario.nsAnnotations)
			ns, _ := fakeClient.CoreV1().Namespaces().Create(context.TODO(), newNs, metav1.CreateOptions{})
			assert.NotNil(ns)
		}

		err := scenario.cmd.run()
		assert.Nil(err)

		// Test expectation for scenario
		for _, ns := range scenario.cmd.namespaces {
			nsWithMetrics, _ := fakeClient.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
			assert.NotNil(nsWithMetrics)
			assert.NotContains(nsWithMetrics.Annotations, constants.MetricsAnnotation)
		}
	}
}

func TestIsMonitoredNamespace(t *testing.T) {
	assert := assert.New(t)

	meshList := mapset.NewSet(testMesh)

	nsMonitored := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "ns-1",
			Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: testMesh},
		},
	}

	nsUnmonitored := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns-3",
		},
	}

	nsInvalidMonitorLabel := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "ns-4",
			Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: ""},
		},
	}

	nsWrongMeshName := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "ns-2",
			Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: "some-mesh"},
		},
	}

	testCases := []struct {
		ns        corev1.Namespace
		exists    bool
		expectErr bool
	}{
		{nsMonitored, true, false},
		{nsUnmonitored, false, false},
		{nsInvalidMonitorLabel, false, true},
		{nsWrongMeshName, false, true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing if %s exists is monitored", tc.ns.Name), func(t *testing.T) {
			monitored, err := isMonitoredNamespace(tc.ns, meshList)
			assert.Equal(monitored, tc.exists)
			assert.Equal(err != nil, tc.expectErr)
		})
	}
}
