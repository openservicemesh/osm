package envoy

import (
	envoy_api_v3_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_api_v3_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/protobuf/ptypes/wrappers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("Test Envoy tools", func() {
	Context("Test GetAddress()", func() {
		It("should return address", func() {
			addr := "blah"
			port := uint32(95346)
			actual := GetAddress(addr, port)
			expected := &envoy_api_v3_core.Address{
				Address: &envoy_api_v3_core.Address_SocketAddress{
					SocketAddress: &envoy_api_v3_core.SocketAddress{
						Protocol: envoy_api_v3_core.SocketAddress_TCP,
						Address:  addr,
						PortSpecifier: &envoy_api_v3_core.SocketAddress_PortValue{
							PortValue: port,
						},
					},
				},
			}

			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test CertName interface", func() {
		It("Interface marshals and unmarshals preserving the exact same data", func() {
			InitialObj := SDSCert{
				CertType: ServiceCertType,
				Service: service.NamespacedService{
					Namespace: "test-namespace",
					Service:   "test-service",
				},
			}

			// Marshal/stringify it
			marshaledStr := InitialObj.String()

			// Unmarshal it back from the string
			finalObj, _ := UnmarshalSDSCert(marshaledStr)

			// First and final object must be equal
			Expect(*finalObj).To(Equal(InitialObj))
		})
	})

	Context("Test getRequestedCertType()", func() {
		It("returns service cert", func() {
			actual, err := UnmarshalSDSCert("service-cert:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(ServiceCertType))
			Expect(actual.Service.Namespace).To(Equal("namespace-test"))
			Expect(actual.Service.Service).To(Equal("blahBlahBlahCert"))
		})
		It("returns root cert for mTLS", func() {
			actual, err := UnmarshalSDSCert("root-cert-for-mtls-outbound:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(RootCertTypeForMTLSOutbound))
			Expect(actual.Service.Namespace).To(Equal("namespace-test"))
			Expect(actual.Service.Service).To(Equal("blahBlahBlahCert"))
		})

		It("returns root cert for non-mTLS", func() {
			actual, err := UnmarshalSDSCert("root-cert-https:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(RootCertTypeForHTTPS))
			Expect(actual.Service.Namespace).To(Equal("namespace-test"))
			Expect(actual.Service.Service).To(Equal("blahBlahBlahCert"))
		})

		It("returns an error (invalid formatting)", func() {
			_, err := UnmarshalSDSCert("blahBlahBlahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (invalid formatting)", func() {
			_, err := UnmarshalSDSCert("blahBlahBlahCert:moreblabla/amazingservice:bla")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (missing cert type)", func() {
			_, err := UnmarshalSDSCert("blahBlahBlahCert/service")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (service is not namespaced)", func() {
			_, err := UnmarshalSDSCert("root-cert-https:blahBlahBlahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (invalid namespace formatting)", func() {
			_, err := UnmarshalSDSCert("root-cert-https:blah/BlahBl/ahCert")
			Expect(err).To(HaveOccurred())
		})
		It("returns an error (empty left-side namespace)", func() {
			_, err := UnmarshalSDSCert("root-cert-https:/ahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (empty cert type)", func() {
			_, err := UnmarshalSDSCert(":ns/svc")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (empty slice on right/wrong number of slices)", func() {
			_, err := UnmarshalSDSCert("root-cert-https:aaa/ahCert:")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (invalid serv type)", func() {
			_, err := UnmarshalSDSCert("revoked-cert:blah/BlahBlahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (invalid mtls cert type)", func() {
			_, err := UnmarshalSDSCert("oot-cert-for-mtls-diagonalstream:blah/BlahBlahCert")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test GetDownstreamTLSContext()", func() {
		It("should return TLS context", func() {
			tlsContext := GetDownstreamTLSContext(tests.BookstoreService, true)

			expectedTLSContext := &envoy_api_v3_auth.DownstreamTlsContext{
				CommonTlsContext: &envoy_api_v3_auth.CommonTlsContext{
					TlsParams: &envoy_api_v3_auth.TlsParameters{
						TlsMinimumProtocolVersion: 3,
						TlsMaximumProtocolVersion: 4,
					},
					TlsCertificates: nil,
					TlsCertificateSdsSecretConfigs: []*envoy_api_v3_auth.SdsSecretConfig{{
						Name: "service-cert:default/bookstore",
						SdsConfig: &envoy_api_v3_core.ConfigSource{
							ConfigSourceSpecifier: &envoy_api_v3_core.ConfigSource_Ads{
								Ads: &envoy_api_v3_core.AggregatedConfigSource{},
							},
						},
					}},
					ValidationContextType: &envoy_api_v3_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
						ValidationContextSdsSecretConfig: &envoy_api_v3_auth.SdsSecretConfig{
							Name: SDSCert{
								Service: service.NamespacedService{
									Namespace: "default",
									Service:   "bookstore",
								},
								CertType: RootCertTypeForMTLSInbound,
							}.String(),
							SdsConfig: &envoy_api_v3_core.ConfigSource{
								ConfigSourceSpecifier: &envoy_api_v3_core.ConfigSource_Ads{
									Ads: &envoy_api_v3_core.AggregatedConfigSource{},
								},
							},
						},
					},
					AlpnProtocols: nil,
				},
				RequireClientCertificate: &wrappers.BoolValue{Value: true},
			}
			Expect(tlsContext.CommonTlsContext.TlsParams).To(Equal(expectedTLSContext.CommonTlsContext.TlsParams))
			Expect(tlsContext.CommonTlsContext.TlsCertificates).To(Equal(expectedTLSContext.CommonTlsContext.TlsCertificates))
			Expect(tlsContext.CommonTlsContext.TlsCertificateSdsSecretConfigs).To(Equal(expectedTLSContext.CommonTlsContext.TlsCertificateSdsSecretConfigs))
			Expect(tlsContext.CommonTlsContext.ValidationContextType).To(Equal(expectedTLSContext.CommonTlsContext.ValidationContextType))
			Expect(tlsContext).To(Equal(expectedTLSContext))
		})
	})

	Context("Test GetDownstreamTLSContext() for mTLS", func() {
		It("should return TLS context with client certificate validation enabled", func() {
			tlsContext := GetDownstreamTLSContext(tests.BookstoreService, true)
			Expect(tlsContext.RequireClientCertificate).To(Equal(&wrappers.BoolValue{Value: true}))
		})
	})

	Context("Test GetDownstreamTLSContext() for TLS", func() {
		It("should return TLS context with client certificate validation disabled", func() {
			tlsContext := GetDownstreamTLSContext(tests.BookstoreService, false)
			Expect(tlsContext.RequireClientCertificate).To(Equal(&wrappers.BoolValue{Value: false}))
		})
	})

	Context("Test GetUpstreamTLSContext()", func() {
		It("should return TLS context", func() {
			sni := "bookstore.default.svc.cluster.local"
			tlsContext := GetUpstreamTLSContext(tests.BookstoreService, sni)

			expectedTLSContext := &envoy_api_v3_auth.UpstreamTlsContext{
				CommonTlsContext: &envoy_api_v3_auth.CommonTlsContext{
					TlsParams: &envoy_api_v3_auth.TlsParameters{
						TlsMinimumProtocolVersion: 3,
						TlsMaximumProtocolVersion: 4,
					},
					TlsCertificates: nil,
					TlsCertificateSdsSecretConfigs: []*envoy_api_v3_auth.SdsSecretConfig{{
						Name: "service-cert:default/bookstore",
						SdsConfig: &envoy_api_v3_core.ConfigSource{
							ConfigSourceSpecifier: &envoy_api_v3_core.ConfigSource_Ads{
								Ads: &envoy_api_v3_core.AggregatedConfigSource{},
							},
						},
					}},
					ValidationContextType: &envoy_api_v3_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
						ValidationContextSdsSecretConfig: &envoy_api_v3_auth.SdsSecretConfig{
							Name: SDSCert{
								Service: service.NamespacedService{
									Namespace: "default",
									Service:   "bookstore",
								},
								CertType: RootCertTypeForMTLSOutbound,
							}.String(),
							SdsConfig: &envoy_api_v3_core.ConfigSource{
								ConfigSourceSpecifier: &envoy_api_v3_core.ConfigSource_Ads{
									Ads: &envoy_api_v3_core.AggregatedConfigSource{},
								},
							},
						},
					},
					AlpnProtocols: ALPNInMesh,
				},
				Sni:                sni, // "bookstore.default.svc.cluster.local"
				AllowRenegotiation: false,
			}

			// Ensure the SNI is in the expected format!
			Expect(tlsContext.Sni).To(Equal(tests.BookstoreService.GetCommonName().String()))
			Expect(tlsContext.Sni).To(Equal("bookstore.default.svc.cluster.local"))

			Expect(tlsContext.CommonTlsContext.TlsParams).To(Equal(expectedTLSContext.CommonTlsContext.TlsParams))
			Expect(tlsContext.CommonTlsContext.TlsCertificates).To(Equal(expectedTLSContext.CommonTlsContext.TlsCertificates))
			Expect(tlsContext.CommonTlsContext.TlsCertificateSdsSecretConfigs).To(Equal(expectedTLSContext.CommonTlsContext.TlsCertificateSdsSecretConfigs))
			Expect(tlsContext.CommonTlsContext.ValidationContextType).To(Equal(expectedTLSContext.CommonTlsContext.ValidationContextType))
			Expect(tlsContext).To(Equal(expectedTLSContext))
		})
	})

	Context("Test GetUpstreamTLSContext()", func() {
		It("creates correct UpstreamTlsContext.Sni field", func() {
			sni := "test.default.svc.cluster.local"
			tlsContext := GetUpstreamTLSContext(tests.BookbuyerService, sni)
			// To show the actual string for human comprehension
			Expect(tlsContext.Sni).To(Equal(sni))
		})
	})

	Context("Test getCommonTLSContext()", func() {
		It("returns proper auth.CommonTlsContext for mTLS", func() {
			namespacedService := service.NamespacedService{
				Namespace: "-namespace-",
				Service:   "-service-",
			}
			actual := getCommonTLSContext(namespacedService, true /* mTLS */, Inbound)

			expectedServiceCertName := SDSCert{
				Service:  namespacedService,
				CertType: ServiceCertType,
			}.String()
			expectedRootCertName := SDSCert{
				Service:  namespacedService,
				CertType: RootCertTypeForMTLSInbound,
			}.String()

			expected := &envoy_api_v3_auth.CommonTlsContext{
				TlsParams: GetTLSParams(),
				TlsCertificateSdsSecretConfigs: []*envoy_api_v3_auth.SdsSecretConfig{{
					Name:      expectedServiceCertName,
					SdsConfig: GetADSConfigSource(),
				}},
				ValidationContextType: &envoy_api_v3_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
					ValidationContextSdsSecretConfig: &envoy_api_v3_auth.SdsSecretConfig{
						Name:      expectedRootCertName,
						SdsConfig: GetADSConfigSource(),
					},
				},
				AlpnProtocols: nil,
			}

			Expect(len(actual.TlsCertificateSdsSecretConfigs)).To(Equal(1))
			Expect(actual.TlsCertificateSdsSecretConfigs[0].Name).To(Equal(expectedServiceCertName))
			Expect(actual.GetValidationContextSdsSecretConfig().Name).To(Equal(expectedRootCertName))
			Expect(actual).To(Equal(expected))
		})

		It("returns proper auth.CommonTlsContext for non-mTLS", func() {
			namespacedService := service.NamespacedService{
				Namespace: "-namespace-",
				Service:   "-service-",
			}
			actual := getCommonTLSContext(namespacedService, false, false /* Ignored in case of non-tls */)

			expectedServiceCertName := SDSCert{
				Service:  namespacedService,
				CertType: ServiceCertType,
			}.String()
			expectedRootCertName := SDSCert{
				Service:  namespacedService,
				CertType: RootCertTypeForHTTPS,
			}.String()

			expected := &envoy_api_v3_auth.CommonTlsContext{
				TlsParams: GetTLSParams(),
				TlsCertificateSdsSecretConfigs: []*envoy_api_v3_auth.SdsSecretConfig{{
					Name:      expectedServiceCertName,
					SdsConfig: GetADSConfigSource(),
				}},
				ValidationContextType: &envoy_api_v3_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
					ValidationContextSdsSecretConfig: &envoy_api_v3_auth.SdsSecretConfig{
						Name:      expectedRootCertName,
						SdsConfig: GetADSConfigSource(),
					},
				},
				AlpnProtocols: nil,
			}

			Expect(len(actual.TlsCertificateSdsSecretConfigs)).To(Equal(1))
			Expect(actual.TlsCertificateSdsSecretConfigs[0].Name).To(Equal(expectedServiceCertName))
			Expect(actual.GetValidationContextSdsSecretConfig().Name).To(Equal(expectedRootCertName))
			Expect(actual).To(Equal(expected))
		})
	})
})
