package envoy

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/protobuf/ptypes/wrappers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test Envoy tools", func() {
	Context("Test GetAddress()", func() {
		It("should return address", func() {
			addr := "blah"
			port := uint32(95346)
			actual := GetAddress(addr, port)
			expected := &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.SocketAddress_TCP,
						Address:  addr,
						PortSpecifier: &core.SocketAddress_PortValue{
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
				MeshService: service.MeshService{
					Namespace: "test-namespace",
					Name:      "test-service",
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
			Expect(actual.MeshService.Namespace).To(Equal("namespace-test"))
			Expect(actual.MeshService.Name).To(Equal("blahBlahBlahCert"))
		})
		It("returns root cert for mTLS", func() {
			actual, err := UnmarshalSDSCert("root-cert-for-mtls-outbound:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(RootCertTypeForMTLSOutbound))
			Expect(actual.MeshService.Namespace).To(Equal("namespace-test"))
			Expect(actual.MeshService.Name).To(Equal("blahBlahBlahCert"))
		})

		It("returns root cert for non-mTLS", func() {
			actual, err := UnmarshalSDSCert("root-cert-https:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(RootCertTypeForHTTPS))
			Expect(actual.MeshService.Namespace).To(Equal("namespace-test"))
			Expect(actual.MeshService.Name).To(Equal("blahBlahBlahCert"))
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
			tlsContext := GetDownstreamTLSContext(tests.BookstoreV1Service, true)

			expectedTLSContext := &auth.DownstreamTlsContext{
				CommonTlsContext: &auth.CommonTlsContext{
					TlsParams: &auth.TlsParameters{
						TlsMinimumProtocolVersion: 3,
						TlsMaximumProtocolVersion: 4,
					},
					TlsCertificates: nil,
					TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
						Name: "service-cert:default/bookstore-v1",
						SdsConfig: &core.ConfigSource{
							ConfigSourceSpecifier: &core.ConfigSource_Ads{
								Ads: &core.AggregatedConfigSource{},
							},
							ResourceApiVersion: core.ApiVersion_V3,
						},
					}},
					ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
						ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
							Name: SDSCert{
								MeshService: service.MeshService{
									Namespace: "default",
									Name:      "bookstore-v1",
								},
								CertType: RootCertTypeForMTLSInbound,
							}.String(),
							SdsConfig: &core.ConfigSource{
								ConfigSourceSpecifier: &core.ConfigSource_Ads{
									Ads: &core.AggregatedConfigSource{},
								},
								ResourceApiVersion: core.ApiVersion_V3,
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
			tlsContext := GetDownstreamTLSContext(tests.BookstoreV1Service, true)
			Expect(tlsContext.RequireClientCertificate).To(Equal(&wrappers.BoolValue{Value: true}))
		})
	})

	Context("Test GetDownstreamTLSContext() for TLS", func() {
		It("should return TLS context with client certificate validation disabled", func() {
			tlsContext := GetDownstreamTLSContext(tests.BookstoreV1Service, false)
			Expect(tlsContext.RequireClientCertificate).To(Equal(&wrappers.BoolValue{Value: false}))
		})
	})

	Context("Test GetUpstreamTLSContext()", func() {
		It("should return TLS context", func() {
			sni := "bookstore-v1.default.svc.cluster.local"
			tlsContext := GetUpstreamTLSContext(tests.BookstoreV1Service, sni)

			expectedTLSContext := &auth.UpstreamTlsContext{
				CommonTlsContext: &auth.CommonTlsContext{
					TlsParams: &auth.TlsParameters{
						TlsMinimumProtocolVersion: 3,
						TlsMaximumProtocolVersion: 4,
					},
					TlsCertificates: nil,
					TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
						Name: "service-cert:default/bookstore-v1",
						SdsConfig: &core.ConfigSource{
							ConfigSourceSpecifier: &core.ConfigSource_Ads{
								Ads: &core.AggregatedConfigSource{},
							},
							ResourceApiVersion: core.ApiVersion_V3,
						},
					}},
					ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
						ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
							Name: SDSCert{
								MeshService: service.MeshService{
									Namespace: "default",
									Name:      "bookstore-v1",
								},
								CertType: RootCertTypeForMTLSOutbound,
							}.String(),
							SdsConfig: &core.ConfigSource{
								ConfigSourceSpecifier: &core.ConfigSource_Ads{
									Ads: &core.AggregatedConfigSource{},
								},
								ResourceApiVersion: core.ApiVersion_V3,
							},
						},
					},
					AlpnProtocols: ALPNInMesh,
				},
				Sni:                sni, // "bookstore.default.svc.cluster.local"
				AllowRenegotiation: false,
			}

			// Ensure the SNI is in the expected format!
			Expect(tlsContext.Sni).To(Equal(tests.BookstoreV1Service.GetCommonName().String()))
			Expect(tlsContext.Sni).To(Equal("bookstore-v1.default.svc.cluster.local"))

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
			namespacedService := service.MeshService{
				Namespace: "-namespace-",
				Name:      "-service-",
			}
			actual := getCommonTLSContext(namespacedService, true /* mTLS */, Inbound)

			expectedServiceCertName := SDSCert{
				MeshService: namespacedService,
				CertType:    ServiceCertType,
			}.String()
			expectedRootCertName := SDSCert{
				MeshService: namespacedService,
				CertType:    RootCertTypeForMTLSInbound,
			}.String()

			expected := &auth.CommonTlsContext{
				TlsParams: GetTLSParams(),
				TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
					Name:      expectedServiceCertName,
					SdsConfig: GetADSConfigSource(),
				}},
				ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
					ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
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
			namespacedService := service.MeshService{
				Namespace: "-namespace-",
				Name:      "-service-",
			}
			actual := getCommonTLSContext(namespacedService, false, false /* Ignored in case of non-tls */)

			expectedServiceCertName := SDSCert{
				MeshService: namespacedService,
				CertType:    ServiceCertType,
			}.String()
			expectedRootCertName := SDSCert{
				MeshService: namespacedService,
				CertType:    RootCertTypeForHTTPS,
			}.String()

			expected := &auth.CommonTlsContext{
				TlsParams: GetTLSParams(),
				TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
					Name:      expectedServiceCertName,
					SdsConfig: GetADSConfigSource(),
				}},
				ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
					ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
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
