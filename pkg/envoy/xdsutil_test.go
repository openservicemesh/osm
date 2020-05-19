package envoy

import (
	"fmt"

	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
				Value:   []byte{10, 79, 10, 4, 8, 3, 16, 4, 50, 36, 10, 30, 115, 101, 114, 118, 105, 99, 101, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 115, 116, 111, 114, 101, 18, 2, 26, 0, 58, 33, 10, 27, 114, 111, 111, 116, 45, 99, 101, 114, 116, 58, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 115, 116, 111, 114, 101, 18, 2, 26, 0, 18, 17, 100, 101, 102, 97, 117, 108, 116, 47, 98, 111, 111, 107, 115, 116, 111, 114, 101},
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
})
