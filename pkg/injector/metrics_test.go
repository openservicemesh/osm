package injector

import (
	"context"
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
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

func TestIsMetricsEnabled(t *testing.T) {
	assert := tassert.New(t)
	fakeClient := fake.NewSimpleClientset()

	// create namespace with metrics enabled
	nsWithMetrics := newNamespace("ns-1", map[string]string{constants.MetricsAnnotation: "enabled"})
	ns, _ := fakeClient.CoreV1().Namespaces().Create(context.TODO(), nsWithMetrics, metav1.CreateOptions{})
	assert.NotNil(ns)

	// create namespace with metrics disabled
	nsWithMetricsDisabled := newNamespace("ns-2", map[string]string{constants.MetricsAnnotation: "disabled"})
	ns, _ = fakeClient.CoreV1().Namespaces().Create(context.TODO(), nsWithMetricsDisabled, metav1.CreateOptions{})
	assert.NotNil(ns)

	// create namespace without metrics annotation
	nsWithoutMetricsAnnotation := newNamespace("ns-3", nil)
	ns, _ = fakeClient.CoreV1().Namespaces().Create(context.TODO(), nsWithoutMetricsAnnotation, metav1.CreateOptions{})
	assert.NotNil(ns)

	// create namespace with invalid annotation value
	nsWithInvalidAnnotation := newNamespace("ns-4", map[string]string{constants.MetricsAnnotation: "invalid"})
	ns, _ = fakeClient.CoreV1().Namespaces().Create(context.TODO(), nsWithInvalidAnnotation, metav1.CreateOptions{})
	assert.NotNil(ns)

	// Test different scenarios using table driven testing
	testCases := []struct {
		namespace                string
		expectMetricsToBeEnabled bool // set to true if metrics is expected to be enabled
		expectedErr              bool // set to true if error is expected
	}{
		{nsWithMetrics.Name, true, false},
		{nsWithMetricsDisabled.Name, false, false},
		{nsWithoutMetricsAnnotation.Name, false, false},
		{nsWithInvalidAnnotation.Name, false, true},
	}

	mockController := k8s.NewMockController(gomock.NewController(t))
	wh := &mutatingWebhook{
		kubeClient:          fakeClient,
		kubeController:      mockController,
		nonInjectNamespaces: mapset.NewSet(),
	}

	mockController.EXPECT().GetNamespace("ns-1").Return(nsWithMetrics)
	mockController.EXPECT().GetNamespace("ns-2").Return(nsWithMetricsDisabled)
	mockController.EXPECT().GetNamespace("ns-3").Return(nsWithoutMetricsAnnotation)
	mockController.EXPECT().GetNamespace("ns-4").Return(nsWithInvalidAnnotation)

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Namespace %s", tc.namespace), func(t *testing.T) {
			enabled, err := wh.isMetricsEnabled(tc.namespace)
			assert.Equal(enabled, tc.expectMetricsToBeEnabled)
			assert.Equal(err != nil, tc.expectedErr)
		})
	}
}
