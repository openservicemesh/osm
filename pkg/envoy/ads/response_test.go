package ads

import (
	"context"
	"fmt"
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("Test ADS response functions", func() {

	// --- setup
	kubeClient := testclient.NewSimpleClientset()
	namespace := tests.Namespace
	envoyUID := tests.EnvoyUID
	serviceName := tests.BookstoreServiceName
	serviceAccountName := tests.BookstoreServiceAccountName

	labels := map[string]string{constants.EnvoyUniqueIDLabelName: tests.EnvoyUID}
	mc := catalog.NewFakeMeshCatalog(kubeClient)

	// Create a Pod
	pod := tests.NewPodTestFixture(namespace, fmt.Sprintf("pod-0-%s", uuid.New()))
	pod.Labels[constants.EnvoyUniqueIDLabelName] = envoyUID
	_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
	It("should have created a pod", func() {
		Expect(err).ToNot(HaveOccurred())
	})

	service := tests.NewServiceFixture(serviceName, namespace, labels)
	_, err = kubeClient.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	It("should have created a service", func() {
		Expect(err).ToNot(HaveOccurred())
	})
	cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", envoyUID, serviceAccountName, namespace))
	proxy := envoy.NewProxy(cn, nil)

	expectedSecretOneName := fmt.Sprintf("service-cert:default/%s", serviceName)
	expectedSecretTwoName := fmt.Sprintf("root-cert-for-mtls:default/%s", serviceName)
	expectedSecretThreeName := fmt.Sprintf("root-cert-https:default/%s", serviceName)

	Context("Test getRequestedCertType()", func() {
		It("returns service cert", func() {

			actual := makeRequestForAllSecrets(proxy, mc)
			expected := envoy_api_v2.DiscoveryRequest{
				TypeUrl: string(envoy.TypeSDS),
				ResourceNames: []string{
					expectedSecretOneName,
					expectedSecretTwoName,
					expectedSecretThreeName,
				},
			}
			Expect(actual).ToNot(BeNil())
			Expect(*actual).To(Equal(expected))
		})
	})

	Context("Test sendAllResponses()", func() {

		cache := make(map[certificate.CommonName]certificate.Certificater)
		certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
		cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", uuid.New(), serviceAccountName, tests.Namespace))
		certPEM, _ := certManager.IssueCertificate(cn, nil)
		cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())
		server, actualResponses := tests.NewFakeXDSServer(cert, nil, nil)
		config := &configurator.Config{
			OSMNamespace: "-test-namespace-",
		}

		It("returns Aggregated Discovery Service response", func() {
			s := Server{
				ctx:         context.TODO(),
				catalog:     mc,
				meshSpec:    smi.NewFakeMeshSpecClient(),
				xdsHandlers: getHandlers(),
			}

			s.sendAllResponses(proxy, &server, config)

			Expect(actualResponses).ToNot(BeNil())
			Expect(len(*actualResponses)).To(Equal(5))

			Expect((*actualResponses)[0].VersionInfo).To(Equal("1"))
			Expect((*actualResponses)[0].TypeUrl).To(Equal(string(envoy.TypeCDS)))

			Expect((*actualResponses)[1].VersionInfo).To(Equal("1"))
			Expect((*actualResponses)[1].TypeUrl).To(Equal(string(envoy.TypeEDS)))

			Expect((*actualResponses)[2].VersionInfo).To(Equal("1"))
			Expect((*actualResponses)[2].TypeUrl).To(Equal(string(envoy.TypeLDS)))

			Expect((*actualResponses)[3].VersionInfo).To(Equal("1"))
			Expect((*actualResponses)[3].TypeUrl).To(Equal(string(envoy.TypeRDS)))

			Expect((*actualResponses)[4].VersionInfo).To(Equal("1"))
			Expect((*actualResponses)[4].TypeUrl).To(Equal(string(envoy.TypeSDS)))
			Expect(len((*actualResponses)[4].Resources)).To(Equal(3))

			secretOne := envoy_api_v2_auth.Secret{}
			firstSecret := (*actualResponses)[4].Resources[0]
			err = ptypes.UnmarshalAny(firstSecret, &secretOne)
			Expect(secretOne.Name).To(Equal(expectedSecretOneName))

			secretTwo := envoy_api_v2_auth.Secret{}
			secondSecret := (*actualResponses)[4].Resources[1]
			err = ptypes.UnmarshalAny(secondSecret, &secretTwo)
			Expect(secretTwo.Name).To(Equal(expectedSecretTwoName))

			secretThree := envoy_api_v2_auth.Secret{}
			thirdSecret := (*actualResponses)[4].Resources[2]
			err = ptypes.UnmarshalAny(thirdSecret, &secretThree)
			Expect(secretThree.Name).To(Equal(expectedSecretThreeName))

		})
	})
})
