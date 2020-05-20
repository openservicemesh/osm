package envoy

import (
	"fmt"

	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

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
			expected := &envoy_api_v2_core.Address{
				Address: &envoy_api_v2_core.Address_SocketAddress{
					SocketAddress: &envoy_api_v2_core.SocketAddress{
						Protocol: envoy_api_v2_core.SocketAddress_TCP,
						Address:  addr,
						PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
							PortValue: port,
						},
					},
				},
			}

			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test GetUpstreamTLSContext()", func() {
		It("should return TLS context", func() {
			actual := GetUpstreamTLSContext(tests.BookstoreService)
			expected := &any.Any{
				TypeUrl: string(TypeUpstreamTLSContext),
				Value:   []byte{10, 141, 1, 10, 66, 8, 3, 16, 4, 26, 29, 69, 67, 68, 72, 69, 45, 69, 67, 68, 83, 65, 45, 65, 69, 83, 49, 50, 56, 45, 71, 67, 77, 45, 83, 72, 65, 50, 53, 54, 26, 29, 69, 67, 68, 72, 69, 45, 69, 67, 68, 83, 65, 45, 67, 72, 65, 67, 72, 65, 50, 48, 45, 80, 79, 76, 89, 49, 51, 48, 53, 50, 36, 10, 30, 115, 101, 114, 118, 105, 99, 101, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 115, 116, 111, 114, 101, 18, 2, 26, 0, 58, 33, 10, 27, 114, 111, 111, 116, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 115, 116, 111, 114, 101, 18, 2, 26, 0, 18, 17, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 115, 116, 111, 114, 101},
			}
			Expect(actual).To(Equal(expected))

			tlsContext := envoy_api_v2_auth.UpstreamTlsContext{}
			err := ptypes.UnmarshalAny(actual, &tlsContext)
			Expect(err).ToNot(HaveOccurred())

			expectedTLSContext := envoy_api_v2_auth.UpstreamTlsContext{
				CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
					TlsParams: &envoy_api_v2_auth.TlsParameters{
						TlsMinimumProtocolVersion: 3,
						TlsMaximumProtocolVersion: 4,
						CipherSuites: []string{
							"ECDHE-ECDSA-AES128-GCM-SHA256",
							"ECDHE-ECDSA-CHACHA20-POLY1305",
						},
					},
					TlsCertificates: nil,
					TlsCertificateSdsSecretConfigs: []*envoy_api_v2_auth.SdsSecretConfig{{
						Name: "service-cert:default/bookstore",
						SdsConfig: &envoy_api_v2_core.ConfigSource{
							ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
								Ads: &envoy_api_v2_core.AggregatedConfigSource{},
							},
						},
					}},
					ValidationContextType: &envoy_api_v2_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
						ValidationContextSdsSecretConfig: &envoy_api_v2_auth.SdsSecretConfig{
							Name: fmt.Sprintf("%s%s%s", RootCertPrefix, Separator, "default/bookstore"),
							SdsConfig: &envoy_api_v2_core.ConfigSource{
								ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
									Ads: &envoy_api_v2_core.AggregatedConfigSource{},
								},
							},
						},
					},
					AlpnProtocols: nil,
				},
				Sni:                "default/bookstore",
				AllowRenegotiation: false,
			}
			Expect(tlsContext.CommonTlsContext.TlsParams).To(Equal(expectedTLSContext.CommonTlsContext.TlsParams))
			Expect(tlsContext.CommonTlsContext.TlsCertificates).To(Equal(expectedTLSContext.CommonTlsContext.TlsCertificates))
			Expect(tlsContext.CommonTlsContext.TlsCertificateSdsSecretConfigs).To(Equal(expectedTLSContext.CommonTlsContext.TlsCertificateSdsSecretConfigs))
			Expect(tlsContext.CommonTlsContext.ValidationContextType).To(Equal(expectedTLSContext.CommonTlsContext.ValidationContextType))
			Expect(tlsContext).To(Equal(expectedTLSContext))
		})
	})

	Context("Test getCommonTLSContext()", func() {
		It("returns proper auth.CommonTlsContext", func() {
			namespacedService := service.NamespacedService{
				Namespace: "-namespace-",
				Service:   "-service-",
			}
			actual := getCommonTLSContext(namespacedService)

			expectedServiceCertName := fmt.Sprintf("service-cert:%s/%s", namespacedService.Namespace, namespacedService.Service)
			expectedRootCertName := fmt.Sprintf("root-cert:%s/%s", namespacedService.Namespace, namespacedService.Service)
			expected := &envoy_api_v2_auth.CommonTlsContext{
				TlsParams: GetTLSParams(),
				TlsCertificateSdsSecretConfigs: []*envoy_api_v2_auth.SdsSecretConfig{{
					Name:      fmt.Sprintf("%s%s%s/%s", ServiceCertPrefix, Separator, namespacedService.Namespace, namespacedService.Service),
					SdsConfig: GetADSConfigSource(),
				}},
				ValidationContextType: &envoy_api_v2_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
					ValidationContextSdsSecretConfig: &envoy_api_v2_auth.SdsSecretConfig{
						Name:      fmt.Sprintf("%s%s%s/%s", RootCertPrefix, Separator, namespacedService.Namespace, namespacedService.Service),
						SdsConfig: GetADSConfigSource(),
					},
				},
			}

			Expect(len(actual.TlsCertificateSdsSecretConfigs)).To(Equal(1))
			Expect(actual.TlsCertificateSdsSecretConfigs[0].Name).To(Equal(expectedServiceCertName))
			Expect(actual.GetValidationContextSdsSecretConfig().Name).To(Equal(expectedRootCertName))
			Expect(actual).To(Equal(expected))
		})
	})
})
