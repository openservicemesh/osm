package injector

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test all patch operations", func() {

	// Setup all constants and variables needed for the tests
	envoyUID := uuid.New().String()
	const (
		namespace    = "-namespace-"
		podName      = "-pod-name-"
		volumeOne    = "-volume-one-"
		volumeTwo    = "-volume-two-"
		containerOne = "-container-one-"
		containerTwo = "-container-two-"
		basePath     = "/base/path"
	)

	Context("Test updateLabels() function", func() {
		It("adds", func() {
			pod := tests.NewPodTestFixture(namespace, podName)
			pod.Labels = nil
			actual := updateLabels(&pod, envoyUID)
			expected := &JSONPatchOperation{
				Op:   addOperation,
				Path: "/metadata/labels",
				Value: map[string]string{
					"osm-envoy-uid": envoyUID,
				},
			}
			Expect(actual).To(Equal(expected))
		})

		It("replaces", func() {
			pod := tests.NewPodTestFixture(namespace, podName)
			actual := updateLabels(&pod, envoyUID)
			replace := &JSONPatchOperation{
				Op:    replaceOperation,
				Path:  "/metadata/labels/osm-envoy-uid",
				Value: envoyUID,
			}
			Expect(actual).To(Equal(replace))
		})
	})

	Context("test addVolume() function", func() {
		It("adds volume", func() {
			target := []corev1.Volume{{
				Name: volumeOne,
			}}
			add := []corev1.Volume{{
				Name: volumeTwo,
			}}

			actualPatch := addVolume(target, add, basePath)
			addVolume := JSONPatchOperation{
				Op:   addOperation,
				Path: "/base/path/-",
				Value: corev1.Volume{
					Name: volumeTwo,
				},
			}
			Expect(len(actualPatch)).To(Equal(1))
			Expect(actualPatch).To(ContainElement(addVolume))
		})
	})

	Context("test addContainer() function", func() {
		It("adds container", func() {
			target := []corev1.Container{{
				Name: containerOne,
			}}

			add := []corev1.Container{{
				Name: containerTwo,
			}}

			actualPatches := addContainer(target, add, basePath)

			expectedAddContainer := JSONPatchOperation{
				Op:   addOperation,
				Path: "/base/path/-",
				Value: corev1.Container{
					Name: containerTwo,
				},
			}
			Expect(len(actualPatches)).To(Equal(1))
			Expect(actualPatches).To(ContainElement(expectedAddContainer))
		})
	})

	Context("test updateAnnotation() function", func() {
		It("creates a list of patches", func() {
			target := map[string]string{
				"one": "1",
				"two": "2",
			}

			add := map[string]string{
				"two":   "2",
				"three": "3",
			}

			actual := updateAnnotation(target, add, basePath)

			expectedReplaceTwo := JSONPatchOperation{
				Op:    replaceOperation,
				Path:  "/base/path/two",
				Value: "2",
			}
			Expect(actual).To(ContainElement(expectedReplaceTwo))

			expectedAddThree := JSONPatchOperation{
				Op:    addOperation,
				Path:  "/base/path/three",
				Value: "3",
			}
			Expect(actual).To(ContainElement(expectedAddThree))
		})

		It("creates a list of patches", func() {
			annotationsToAdd := map[string]string{
				"three": "3",
				"two":   "2",
			}

			// Target here is NIL -- this means we will be CREATING
			actual := updateAnnotation(nil, annotationsToAdd, basePath)

			// The first operation is "three" ("three" comes before "two" alphabetically)
			// This is a CREATE operation since target is NIL
			expectedCreateThree := JSONPatchOperation{
				Op:   addOperation,
				Path: "/base/path",
				Value: map[string]string{
					"three": "3",
				},
			}
			Expect(actual).To(ContainElement(expectedCreateThree))

			expectedAddTwo := JSONPatchOperation{
				Op:    addOperation,
				Path:  "/base/path/two",
				Value: "2",
			}
			Expect(actual).To(ContainElement(expectedAddTwo))
		})
	})

	Context("test createPatch() function", func() {
		It("creates a patch", func() {
			client := fake.NewSimpleClientset()
			mockCtrl := gomock.NewController(GinkgoT())
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			mockNsController := k8s.NewMockController(mockCtrl)
			mockNsController.EXPECT().GetNamespace(namespace).Return(&corev1.Namespace{})
			testNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			}
			_, err := client.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			cache := make(map[certificate.CommonName]certificate.Certificater)

			wh := &webhook{
				kubeClient:     client,
				kubeController: mockNsController,
				certManager:    tresor.NewFakeCertManager(&cache, mockConfigurator),
				meshCatalog:    catalog.NewFakeMeshCatalog(client),
				configurator:   mockConfigurator,
			}

			pod := tests.NewPodTestFixture(namespace, podName)
			mockConfigurator.EXPECT().GetEnvoyLogLevel().Return("").Times(1)

			jsonPatches, err := wh.createPatch(&pod, namespace, envoyUID)

			Expect(err).ToNot(HaveOccurred())

			expectedJSONPatches := `[` +
				// Add Volumes
				`{"op":addOperation,` +
				`"path":"/spec/volumes",` +
				`"value":[{"name":"envoy-bootstrap-config-volume","secret":{"secretName":"envoy-bootstrap-config-proxy-uuid"}}]},` +

				// Add Init Container
				`{"op":addOperation,` +
				`"path":"/spec/initContainers",` +
				`"value":[{"name":"osm-init","env":[{"name":"OSM_PROXY_UID","value":"1337"},` +
				`{"name":"OSM_ENVOY_INBOUND_PORT","value":"15003"},{"name":"OSM_ENVOY_OUTBOUND_PORT","value":"15001"}],` +
				`"resources":{},"securityContext":{"capabilities":{addOperation:["NET_ADMIN"]}}}]},` +

				// Add Envoy Container
				`{"op":addOperation,"path":"/spec/containers",` +
				`"value":[{"name":"envoy","command":["envoy"],` +
				`"args":["--log-level","","--config-path","/etc/envoy/bootstrap.yaml","--service-node","bookstore","--service-cluster","bookstore.-namespace-","--bootstrap-version 3"],` +
				`"ports":[{"name":"proxy-admin","containerPort":15000},{"name":"proxy-inbound","containerPort":15003},{"name":"proxy-metrics","containerPort":15010}],` +
				`"resources":{},"volumeMounts":[{"name":"envoy-bootstrap-config-volume","readOnly":true,"mountPath":"/etc/envoy"}],` +
				`"imagePullPolicy":"Always",` +
				`"securityContext":{"runAsUser":1337}}]},{"op":addOperation,"path":"/metadata/annotations",` +
				`"value":{"prometheus.io/scrape":"true"}},` +

				// Add Prometheus Port Annotation
				`{"op":addOperation,"path":"/metadata/annotations/prometheus.io~1port","value":"15010"},` +

				// Add Prometheus Path Annotation
				`{"op":addOperation,"path":"/metadata/annotations/prometheus.io~1path","value":"/stats/prometheus"},` +

				// Add Envoy UID Label
				`{"op":replaceOperation,"path":"/metadata/labels/osm-envoy-uid","value":"proxy-uuid"}` +

				`]`

			Expect(string(jsonPatches)).ToNot(Equal(expectedJSONPatches),
				fmt.Sprintf("Actual: %s", jsonPatches))
		})
	})
})
