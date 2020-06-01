package sds

import (
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("Test SDS response functions", func() {
	Context("Test getResourceKindFromRequest()", func() {
		It("returns service cert", func() {
			actual, err := getResourceKindFromRequest("service-cert:blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(envoy.ServiceCertPrefix))
		})

		It("returns root cert", func() {
			actual, err := getResourceKindFromRequest("root-cert:blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(envoy.RootCertPrefix))
		})

		It("returns an error", func() {
			_, err := getResourceKindFromRequest("blahBlahBlahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error", func() {
			_, err := getResourceKindFromRequest("service-cert:blah:BlahBlahCert")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test getServiceFromServiceCertificateRequest()", func() {
		It("returns a properly formatted NamespacedService", func() {
			actual, err := getServiceFromServiceCertificateRequest("service-cert:foo/bar")
			Expect(err).ToNot(HaveOccurred())
			expected := service.NamespacedService{
				Namespace: "foo",
				Service:   "bar",
			}
			Expect(actual).To(Equal(expected))
		})

		It("returns an error", func() {
			actual, err := getServiceFromServiceCertificateRequest("service-cert:guh")
			Expect(err).To(HaveOccurred())
			expected := service.NamespacedService{}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test getServiceFromRootCertificateRequest()", func() {
		It("returns a properly formatted NamespacedService", func() {
			actual, err := getServiceFromRootCertificateRequest("root-cert:foo/bar")
			Expect(err).ToNot(HaveOccurred())
			expected := service.NamespacedService{
				Namespace: "foo",
				Service:   "bar",
			}
			Expect(actual).To(Equal(expected))
		})

		It("returns an error", func() {
			actual, err := getServiceFromRootCertificateRequest("root-cert:guh")
			Expect(err).To(HaveOccurred())
			expected := service.NamespacedService{}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test getRootCert()", func() {
		It("returns a properly formatted struct", func() {
			cache := make(map[certificate.CommonName]certificate.Certificater)
			certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)

			cert, err := certManager.IssueCertificate("blah", nil)
			Expect(err).ToNot(HaveOccurred())

			resourceName := "root-cert:blah"
			mc := catalog.NewFakeMeshCatalog(testclient.NewSimpleClientset())
			actual, err := getRootCert(cert, resourceName, tests.BookstoreService, mc)
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

	Context("Test getRootCertTask()", func() {
		It("returns a properly formatted task", func() {
			resourceName := "root-cert:ns/svc"
			serviceForProxy := service.NamespacedService{Namespace: "ns", Service: "svc"}
			proxyCN := certificate.CommonName("blah")

			actualTask, err := getRootCertTask(resourceName, serviceForProxy, proxyCN)
			Expect(err).ToNot(HaveOccurred())
			Expect(actualTask.resourceName).To(Equal("root-cert:ns/svc"))
		})

		It("returns an error", func() {
			resourceName := "root-cert:ns"
			serviceForProxy := service.NamespacedService{Namespace: "ns", Service: "svc"}
			proxyCN := certificate.CommonName("blah")

			_, err := getRootCertTask(resourceName, serviceForProxy, proxyCN)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test getTasks()", func() {
		It("returns a list of root certificate issuance tasks", func() {
			proxy := envoy.NewProxy("cn", nil)
			req := envoy_api_v2.DiscoveryRequest{
				ResourceNames: []string{"root-cert:ns/svc"},
				TypeUrl:       "",
			}
			serviceForProxy := service.NamespacedService{
				Namespace: "ns",
				Service:   "svc",
			}
			actual := getTasks(proxy, &req, serviceForProxy)
			Expect(len(actual)).To(Equal(1))
			Expect(actual[0].resourceName).To(Equal("root-cert:ns/svc"))
		})

		It("returns a list of service certificate tasks", func() {
			proxy := envoy.NewProxy("cn", nil)
			req := envoy_api_v2.DiscoveryRequest{
				ResourceNames: []string{"service-cert:ns/svc"},
				TypeUrl:       "",
			}
			serviceForProxy := service.NamespacedService{
				Namespace: "ns",
				Service:   "svc",
			}
			actual := getTasks(proxy, &req, serviceForProxy)
			Expect(len(actual)).To(Equal(1))
			Expect(actual[0].resourceName).To(Equal("service-cert:ns/svc"))
		})

		It("returns empty list - the proxy requested something that does not belong to that proxy", func() {
			proxy := envoy.NewProxy("cn", nil)
			req := envoy_api_v2.DiscoveryRequest{
				ResourceNames: []string{"service-cert:ns/svc"},
				TypeUrl:       "",
			}
			serviceForProxy := service.NamespacedService{
				Namespace: "nsXXX",
				Service:   "svcXXX",
			}
			actual := getTasks(proxy, &req, serviceForProxy)
			Expect(len(actual)).To(Equal(0))
		})
	})
})
