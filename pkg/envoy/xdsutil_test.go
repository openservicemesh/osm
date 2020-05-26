package envoy

import (
	"fmt"

	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
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

	Context("Test GetDownstreamTLSContext()", func() {
		It("should return TLS context", func() {
			tlsContext := GetDownstreamTLSContext(tests.BookstoreService, true)

			expectedTLSContext := envoy_api_v2_auth.DownstreamTlsContext{
				CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
					TlsParams: &envoy_api_v2_auth.TlsParameters{
						TlsMinimumProtocolVersion: 3,
						TlsMaximumProtocolVersion: 4,
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
				RequireClientCertificate: &wrappers.BoolValue{Value: true},
			}
			Expect(tlsContext.CommonTlsContext.TlsParams).To(Equal(expectedTLSContext.CommonTlsContext.TlsParams))
			Expect(tlsContext.CommonTlsContext.TlsCertificates).To(Equal(expectedTLSContext.CommonTlsContext.TlsCertificates))
			Expect(tlsContext.CommonTlsContext.TlsCertificateSdsSecretConfigs).To(Equal(expectedTLSContext.CommonTlsContext.TlsCertificateSdsSecretConfigs))
			Expect(tlsContext.CommonTlsContext.ValidationContextType).To(Equal(expectedTLSContext.CommonTlsContext.ValidationContextType))
			Expect(*tlsContext).To(Equal(expectedTLSContext))
		})
	})

	Context("Test GetDownstreamTLSContext() for mTLS", func() {
		It("should return TLS context with client certificate validation enabled", func() {
			tlsContext := GetDownstreamTLSContext(tests.BookstoreService, true)
			Expect(*tlsContext.RequireClientCertificate).To(Equal(wrappers.BoolValue{Value: true}))
		})
	})

	Context("Test GetDownstreamTLSContext() for TLS", func() {
		It("should return TLS context with client certificate validation disabled", func() {
			tlsContext := GetDownstreamTLSContext(tests.BookstoreService, false)
			Expect(*tlsContext.RequireClientCertificate).To(Equal(wrappers.BoolValue{Value: false}))
		})
	})

	Context("Test GetUpstreamTLSContext()", func() {
		It("should return TLS context", func() {
			tlsContext := GetUpstreamTLSContext(tests.BookstoreService)

			expectedTLSContext := envoy_api_v2_auth.UpstreamTlsContext{
				CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
					TlsParams: &envoy_api_v2_auth.TlsParameters{
						TlsMinimumProtocolVersion: 3,
						TlsMaximumProtocolVersion: 4,
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
			Expect(*tlsContext).To(Equal(expectedTLSContext))
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
