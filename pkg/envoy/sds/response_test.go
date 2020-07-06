package sds

import (
	"context"
	"fmt"
	"time"

	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-service-mesh/osm/pkg/certresource"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("Test SDS response functions", func() {

	prep := func(resourceNames []string, namespace, svcName string) (certificate.Certificater, *envoy.Proxy, catalog.MeshCataloger) {
		serviceAccount := tests.BookstoreServiceAccountName
		proxyID := uuid.New().String()
		podName := uuid.New().String()

		newPod := tests.NewPodTestFixture(namespace, podName)
		newPod.Labels[constants.EnvoyUniqueIDLabelName] = proxyID
		newPod.Labels[tests.SelectorKey] = tests.SelectorValue
		kubeClient := testclient.NewSimpleClientset()
		_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &newPod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create the SERVICE
		selector := map[string]string{tests.SelectorKey: tests.SelectorValue}
		svc := tests.NewServiceFixture(svcName, namespace, selector)
		_, err = kubeClient.CoreV1().Services(namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		cn := certificate.CommonName(fmt.Sprintf("%s.%s.%s", proxyID, serviceAccount, namespace))

		proxy := envoy.NewProxy(cn, nil)
		cache := make(map[certificate.CommonName]certificate.Certificater)
		validityPeriod := 1 * time.Hour
		certManager := tresor.NewFakeCertManager(&cache, validityPeriod)

		cert, err := certManager.IssueCertificate(cn, nil)
		Expect(err).ToNot(HaveOccurred())

		mc := catalog.NewFakeMeshCatalog(kubeClient)

		return cert, proxy, mc
	}

	Context("Test getRootCert()", func() {
		It("returns a properly formatted struct", func() {
			cache := make(map[certificate.CommonName]certificate.Certificater)
			certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)

			cert, err := certManager.IssueCertificate("blah", nil)
			Expect(err).ToNot(HaveOccurred())

			svc := service.NamespacedService{
				Namespace: "ns",
				Service:   "svc",
			}

			sdsc := certresource.CertResource{
				Service:  svc,
				CertType: certresource.RootCertTypeForMTLSInbound,
			}

			resourceName := sdsc.String()
			mc := catalog.NewFakeMeshCatalog(testclient.NewSimpleClientset())
			actual, err := getRootCert(cert, sdsc, tests.BookstoreService, mc)
			Expect(err).ToNot(HaveOccurred())

			expected := &auth.Secret{
				// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name
				Name: resourceName,
				Type: &auth.Secret_ValidationContext{
					ValidationContext: &auth.CertificateValidationContext{
						TrustedCa: &core.DataSource{
							Specifier: &core.DataSource_InlineBytes{
								InlineBytes: cert.GetIssuingCA(),
							},
						},
						MatchSubjectAltNames: []*envoy_type_matcher.StringMatcher{{
							MatchPattern: &envoy_type_matcher.StringMatcher_Exact{
								// The Certificates Subject Common Name will look like this: "bookbuyer.default.svc.cluster.local"
								// BookbuyerService is an inbound service that is allowed.
								Exact: tests.BookbuyerService.GetCommonName().String(),
							}},
						},
					},
				},
			}

			Expect(actual.Name).To(Equal(expected.Name))

			Expect(actual.GetValidationContext().MatchSubjectAltNames[0].GetExact()).To(Equal("bookbuyer.default.svc.cluster.local"))
			Expect(actual.GetValidationContext().MatchSubjectAltNames).To(Equal(expected.GetValidationContext().MatchSubjectAltNames))

			Expect(actual.GetValidationContext()).To(Equal(expected.GetValidationContext()))
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test getEnvoySDSSecrets()", func() {
		It("returns a list of root certificate issuance tasks for a mTLS root cert", func() {
			namespace := uuid.New().String()
			serviceName := uuid.New().String()

			svc := service.NamespacedService{
				Namespace: namespace,
				Service:   serviceName,
			}

			sdsc := certresource.CertResource{
				Service:  svc,
				CertType: certresource.RootCertTypeForMTLSOutbound,
			}
			resourceNames := []string{sdsc.String()}
			cert, proxy, mc := prep(resourceNames, namespace, serviceName)

			actual := getEnvoySDSSecrets(cert, proxy, resourceNames, mc)

			Expect(len(actual)).To(Equal(1))
			Expect(actual[0].Name).To(Equal(sdsc.String()))
			// Expect(actual[0].GetTlsCertificate()).ToNot(BeNil())
			Expect(actual[0].GetTlsCertificate()).To(BeNil())
			Expect(actual[0].GetValidationContext().TrustedCa.GetInlineBytes()).ToNot(BeNil())
		})

		It("returns a list of root certificate issuance tasks for a HTTPS root cert", func() {
			namespace := uuid.New().String()
			serviceName := uuid.New().String()
			resourceNames := []string{fmt.Sprintf("root-cert-https:%s/%s", namespace, serviceName)}
			cert, proxy, mc := prep(resourceNames, namespace, serviceName)

			actual := getEnvoySDSSecrets(cert, proxy, resourceNames, mc)

			Expect(len(actual)).To(Equal(1))
			Expect(actual[0].Name).To(Equal(fmt.Sprintf("root-cert-https:%s/%s", namespace, serviceName)))
			// Expect(actual[0].GetTlsCertificate()).ToNot(BeNil())
			Expect(actual[0].GetTlsCertificate()).To(BeNil())
			Expect(actual[0].GetValidationContext().TrustedCa.GetInlineBytes()).ToNot(BeNil())
		})

		It("returns a list of service certificate tasks", func() {
			namespace := uuid.New().String()
			serviceName := uuid.New().String()
			resourceNames := []string{fmt.Sprintf("service-cert:%s/%s", namespace, serviceName)}
			cert, proxy, mc := prep(resourceNames, namespace, serviceName)

			actual := getEnvoySDSSecrets(cert, proxy, resourceNames, mc)

			Expect(len(actual)).To(Equal(1))
			Expect(actual[0].Name).To(Equal(fmt.Sprintf("service-cert:%s/%s", namespace, serviceName)))
			Expect(actual[0].GetTlsCertificate().PrivateKey.Specifier).ToNot(BeNil())
			Expect(actual[0].GetTlsCertificate().CertificateChain.Specifier).ToNot(BeNil())
			Expect(actual[0].GetValidationContext()).To(BeNil())
		})

		It("returns empty list - the proxy requested something that does not belong to that proxy", func() {
			namespace := uuid.New().String()
			serviceName := uuid.New().String()
			resourceNames := []string{"service-cert:SomeOtherNamespace/SomeOtherService"}
			cert, proxy, mc := prep(resourceNames, namespace, serviceName)

			actual := getEnvoySDSSecrets(cert, proxy, resourceNames, mc)

			Expect(len(actual)).To(Equal(0))
		})
	})

})
