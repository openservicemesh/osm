package injector

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestCreatePatch(t *testing.T) {
	// Setup all variables and constants needed for the tests
	ctx := context.Background()
	proxyUUID := uuid.New()
	const (
		namespace = "-namespace-"
		podName   = "-pod-name-"
	)

	testCases := []struct {
		name            string
		os              string
		namespace       *corev1.Namespace
		dryRun          bool
		expectedPatches []string
	}{
		{
			name: "creates a patch for a unix worker",
			os:   constants.OSLinux,
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
			name: "creates a patch for a windows worker",
			os:   constants.OSWindows,
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
				// Add Envoy Container
				`"path":"/spec/containers"`,
				`"command":["envoy"]`,
			},
		},
		{
			name: "metrics enabled",
			os:   constants.OSLinux,
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
		{
			name: "unix dry run",
			os:   constants.OSLinux,
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			},
			dryRun: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			client := fake.NewSimpleClientset()
			mockCtrl := gomock.NewController(t)
			mockNsController := k8s.NewMockController(mockCtrl)
			mockNsController.EXPECT().GetNamespace(namespace).Return(tc.namespace).AnyTimes()
			_, err := client.CoreV1().Namespaces().Create(context.TODO(), tc.namespace, metav1.CreateOptions{})
			assert.NoError(err)

			wh := &mutatingWebhook{
				kubeClient:          client,
				kubeController:      mockNsController,
				certManager:         tresorFake.NewFake(1 * time.Hour),
				nonInjectNamespaces: mapset.NewSet(),
			}

			mockNsController.EXPECT().GetMeshConfig().Return(v1alpha2.MeshConfig{
				Spec: v1alpha2.MeshConfigSpec{
					Sidecar: v1alpha2.SidecarSpec{
						EnvoyWindowsImage:  "envoy-linux-image",
						EnvoyImage:         "envoy-windows-image",
						InitContainerImage: "init-container-image",
						Resources:          corev1.ResourceRequirements{},
					},
				},
			}).AnyTimes()

			pod := tests.NewOsSpecificPodFixture(namespace, podName, tests.BookstoreServiceAccountName, nil, tc.os)

			raw, err := json.Marshal(pod)
			assert.NoError(err)

			req := &admissionv1.AdmissionRequest{
				Namespace: namespace,
				Object:    runtime.RawExtension{Raw: raw},
				DryRun:    &tc.dryRun,
			}
			rawPatches, err := wh.createPatch(pod, req, proxyUUID)
			assert.NoError(err)
			patches := string(rawPatches)

			for _, expectedPatch := range tc.expectedPatches {
				assert.Contains(patches, expectedPatch)
			}

			// Ensure the bootstrap config was created if not in dry run
			conf, err := client.CoreV1().Secrets(namespace).Get(ctx, "envoy-bootstrap-config-"+proxyUUID.String(), metav1.GetOptions{})
			if tc.dryRun {
				assert.Error(err)
				assert.Nil(conf)
			} else {
				assert.NoError(err)
				assert.NotNil(conf)
			}

			// Now we try to reinject, and ensure the only patch is the updated UUID. We also verify the config was
			// properly created.

			// Assert that the pod has been injected.
			assert.Contains(pod.Labels, constants.EnvoyUniqueIDLabelName)
			// We remove the object meta to mimic kubectl debug.
			pod.ObjectMeta = metav1.ObjectMeta{Name: "debug", Namespace: namespace}

			raw, err = json.Marshal(pod)
			assert.NoError(err)

			req = &admissionv1.AdmissionRequest{
				Namespace: namespace,
				Object:    runtime.RawExtension{Raw: raw},
				DryRun:    &tc.dryRun,
			}

			newUUID := uuid.New()
			rawPatches, err = wh.createPatch(pod, req, newUUID)
			assert.NoError(err)

			patches = string(rawPatches)

			expectedPatches := []string{
				fmt.Sprintf(`{"op":"add","path":"/metadata/labels","value":{"osm-proxy-uuid":"%s"}}`, newUUID.String()),
				fmt.Sprintf(`{"op":"replace","path":"/spec/volumes/0/secret/secretName","value":"envoy-bootstrap-config-%s"}`, newUUID.String()),
			}

			for _, expectedPatch := range expectedPatches {
				assert.Contains(patches, expectedPatch)
			}

			conf, err = client.CoreV1().Secrets(namespace).Get(ctx, "envoy-bootstrap-config-"+newUUID.String(), metav1.GetOptions{})
			if tc.dryRun {
				assert.Error(err)
				assert.Nil(conf)
			} else {
				assert.NoError(err)
				assert.NotNil(conf)
			}
		})
	}

	t.Run("error checking if metrics is enabled", func(t *testing.T) {
		assert := tassert.New(t)
		client := fake.NewSimpleClientset()
		mockCtrl := gomock.NewController(t)
		mockNsController := k8s.NewMockController(mockCtrl)

		wh := &mutatingWebhook{
			kubeClient:          client,
			kubeController:      mockNsController,
			certManager:         tresorFake.NewFake(1 * time.Hour),
			nonInjectNamespaces: mapset.NewSet(),
		}

		namespace := "not-" + namespace

		mockNsController.EXPECT().GetNamespace(namespace).Return(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}).AnyTimes()
		mockNsController.EXPECT().GetMeshConfig().Return(v1alpha2.MeshConfig{}).AnyTimes()

		pod := tests.NewOsSpecificPodFixture(namespace, podName, tests.BookstoreServiceAccountName, nil, constants.OSLinux)

		raw, err := json.Marshal(pod)
		assert.NoError(err)

		req := &admissionv1.AdmissionRequest{
			Namespace: namespace,
			Object:    runtime.RawExtension{Raw: raw},
		}
		_, err = wh.createPatch(pod, req, proxyUUID)
		assert.Error(err)
	})
}

func TestVerifyPrerequisites(t *testing.T) {
	testCases := []struct {
		name         string
		podOS        string
		linuxImage   string
		windowsImage string
		initImage    string
		expectErr    bool
	}{
		{
			name:       "prereqs met for linux pod",
			linuxImage: "envoy",
			initImage:  "init",
			expectErr:  false,
		},
		{
			name:       "prereqs not met for linux pod when init container image is missing",
			linuxImage: "envoy",
			expectErr:  true,
		},
		{
			name:      "prereqs not met for linux pod when envoy container image is missing",
			initImage: "init",
			expectErr: true,
		},
		{
			name:         "prereqs met for windows pod",
			podOS:        "windows",
			windowsImage: "windows",
			initImage:    "init",
			expectErr:    false,
		},
		{
			name:         "prereqs met for windows pod when init container image is missing",
			podOS:        "windows",
			windowsImage: "envoy",
			expectErr:    false,
		},
		{
			name:      "prereqs not met for windows pod when envoy container image is missing",
			initImage: "init",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			assert := tassert.New(t)
			mockK8s := k8s.NewMockController(mockCtrl)

			wh := &mutatingWebhook{
				kubeController: mockK8s,
			}

			mockK8s.EXPECT().GetMeshConfig().Return(v1alpha2.MeshConfig{
				Spec: v1alpha2.MeshConfigSpec{
					Sidecar: v1alpha2.SidecarSpec{
						EnvoyWindowsImage:  tc.windowsImage,
						EnvoyImage:         tc.linuxImage,
						InitContainerImage: tc.initImage,
					},
				},
			}).AnyTimes()

			err := wh.verifyPrerequisites(tc.podOS)
			assert.Equal(tc.expectErr, err != nil)
		})
	}
}
