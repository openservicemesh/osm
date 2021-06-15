package injector

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestCreatePatch(t *testing.T) {
	assert := tassert.New(t)

	// Setup all variables and constants needed for the tests
	proxyUUID := uuid.New()
	const (
		namespace = "-namespace-"
		podName   = "-pod-name-"
	)

	testCases := []struct {
		name            string
		namespace       *corev1.Namespace
		expectedPatches []string
	}{
		{
			name: "creates a patch",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			},
			expectedPatches: []string{
				// Add Envoy UID Label
				`"path":"/metadata/labels"`,
				fmt.Sprintf(`"value":{"osm-proxy-uuid":"%v"`, proxyUUID),
				// Add Volumes
				`"path":"/spec/volumes"`,
				fmt.Sprintf(`"value":[{"name":"envoy-bootstrap-config-volume","secret":{"secretName":"envoy-bootstrap-config-%v"}}]}`, proxyUUID),
				// Add Init Container
				`"path":"/spec/initContainers"`,
				`"command":["/bin/sh"]`,
				// Add Envoy Container
				`"path":"/spec/containers"`,
				`"command":["envoy"]`,
			},
		},
		{
			name: "metrics enabled",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        namespace,
					Annotations: map[string]string{constants.MetricsAnnotation: "enabled"},
				},
			},
			expectedPatches: []string{
				// Add Envoy UID Label
				`"path":"/metadata/labels"`,
				fmt.Sprintf(`"value":{"osm-proxy-uuid":"%v"`, proxyUUID),
				// Add metrics Annotations
				`"path":"/metadata/annotations"`,
				`"value":{"prometheus.io/path":"/stats/prometheus","prometheus.io/port":"15010","prometheus.io/scrape":"true"}`,
				// Add Volumes
				`"path":"/spec/volumes"`,
				fmt.Sprintf(`"value":[{"name":"envoy-bootstrap-config-volume","secret":{"secretName":"envoy-bootstrap-config-%v"}}]}`, proxyUUID),
				// Add Init Container
				`"path":"/spec/initContainers"`,
				`"command":["/bin/sh"]`,
				// Add Envoy Container
				`"path":"/spec/containers"`,
				`"command":["envoy"]`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			mockCtrl := gomock.NewController(t)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			mockNsController := k8s.NewMockController(mockCtrl)
			mockNsController.EXPECT().GetNamespace(namespace).Return(tc.namespace)
			_, err := client.CoreV1().Namespaces().Create(context.TODO(), tc.namespace, metav1.CreateOptions{})
			assert.NoError(err)

			wh := &mutatingWebhook{
				kubeClient:          client,
				kubeController:      mockNsController,
				certManager:         tresor.NewFakeCertManager(mockConfigurator),
				configurator:        mockConfigurator,
				nonInjectNamespaces: mapset.NewSet(),
			}

			mockConfigurator.EXPECT().GetEnvoyLogLevel().Return("").Times(1)
			mockConfigurator.EXPECT().GetEnvoyImage().Return("").Times(1)
			mockConfigurator.EXPECT().GetInitContainerImage().Return("").Times(1)
			mockConfigurator.EXPECT().IsPrivilegedInitContainer().Return(false).Times(1)
			mockConfigurator.EXPECT().GetOutboundIPRangeExclusionList().Return(nil).Times(1)
			mockConfigurator.EXPECT().GetOutboundPortExclusionList().Return(nil).Times(1)
			mockConfigurator.EXPECT().GetInboundPortExclusionList().Return(nil).Times(1)
			mockConfigurator.EXPECT().GetProxyResources().Return(corev1.ResourceRequirements{}).Times(1)

			pod := tests.NewPodFixture(namespace, podName, tests.BookstoreServiceAccountName, nil)

			raw, err := json.Marshal(pod)
			assert.NoError(err)

			req := &admissionv1.AdmissionRequest{Namespace: namespace, Object: runtime.RawExtension{Raw: raw}}
			rawPatches, err := wh.createPatch(&pod, req, proxyUUID)

			assert.NoError(err)

			patches := string(rawPatches)

			for _, expectedPatch := range tc.expectedPatches {
				assert.Contains(patches, expectedPatch)
			}
		})
	}
}

func TestMergePortExclusionLists(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                              string
		podOutboundPortExclusionList      []int
		globalOutboundPortExclusionList   []int
		expectedOutboundPortExclusionList []int
	}{
		{
			name:                              "overlap in global and pod outbound exclusion list",
			podOutboundPortExclusionList:      []int{6060, 7070},
			globalOutboundPortExclusionList:   []int{6060, 8080},
			expectedOutboundPortExclusionList: []int{6060, 7070, 8080},
		},
		{
			name:                              "no overlap in global and pod outbound exclusion list",
			podOutboundPortExclusionList:      []int{6060, 7070},
			globalOutboundPortExclusionList:   []int{8080},
			expectedOutboundPortExclusionList: []int{6060, 7070, 8080},
		},
		{
			name:                              "pod outbound exclusion list is nil",
			podOutboundPortExclusionList:      nil,
			globalOutboundPortExclusionList:   []int{8080},
			expectedOutboundPortExclusionList: []int{8080},
		},
		{
			name:                              "global outbound exclusion list is nil",
			podOutboundPortExclusionList:      []int{6060, 7070},
			globalOutboundPortExclusionList:   nil,
			expectedOutboundPortExclusionList: []int{6060, 7070},
		},
		{
			name:                              "no global or pod level outbound exclusion list",
			podOutboundPortExclusionList:      nil,
			globalOutboundPortExclusionList:   nil,
			expectedOutboundPortExclusionList: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := mergePortExclusionLists(tc.podOutboundPortExclusionList, tc.globalOutboundPortExclusionList)
			assert.ElementsMatch(tc.expectedOutboundPortExclusionList, actual)
		})
	}
}
