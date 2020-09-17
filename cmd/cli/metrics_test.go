package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

func newNamespace(name string, annotations map[string]string) *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	if annotations != nil {
		ns.Annotations = annotations
	}

	return ns
}

func TestRun_MetricsEnable(t *testing.T) {
	assert := assert.New(t)
	fakeClient := fake.NewSimpleClientset()

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
