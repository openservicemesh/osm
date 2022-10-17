package envoy

import (
	"testing"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
)

var sidecarSpec = configv1alpha2.SidecarSpec{
	TLSMinProtocolVersion: "TLSv1_2",
	TLSMaxProtocolVersion: "TLSv1_3",
}

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

	Context("Test GetDownstreamTLSContext()", func() {
		It("should return TLS context", func() {
			svcAccount := identity.K8sServiceAccount{Name: "foo", Namespace: "test"}
			tlsContext := GetDownstreamTLSContext(svcAccount.ToServiceIdentity(), true, sidecarSpec)

			expectedTLSContext := &auth.DownstreamTlsContext{
				CommonTlsContext: &auth.CommonTlsContext{
					TlsParams: &auth.TlsParameters{
						TlsMinimumProtocolVersion: 3,
						TlsMaximumProtocolVersion: 4,
					},
					TlsCertificates: nil,
					TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
						Name: "service-cert:test/foo",
						SdsConfig: &core.ConfigSource{
							ConfigSourceSpecifier: &core.ConfigSource_Ads{
								Ads: &core.AggregatedConfigSource{},
							},
							ResourceApiVersion: core.ApiVersion_V3,
						},
					}},
					ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
						ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
							Name: secrets.NameForMTLSInbound,
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
			tlsContext := GetDownstreamTLSContext(tests.BookstoreServiceIdentity, true, sidecarSpec)
			Expect(tlsContext.RequireClientCertificate).To(Equal(&wrappers.BoolValue{Value: true}))
		})
	})

	Context("Test GetDownstreamTLSContext() for TLS", func() {
		It("should return TLS context with client certificate validation disabled", func() {
			tlsContext := GetDownstreamTLSContext(tests.BookstoreServiceIdentity, false, sidecarSpec)
			Expect(tlsContext.RequireClientCertificate).To(Equal(&wrappers.BoolValue{Value: false}))
		})
	})

	Context("Test GetUpstreamTLSContext()", func() {
		It("should return TLS context", func() {
			sni := "bookstore-v1.default.svc.cluster.local"
			tlsContext := GetUpstreamTLSContext(tests.BookbuyerServiceIdentity, tests.BookstoreV1Service, sidecarSpec)

			expectedTLSContext := &auth.UpstreamTlsContext{
				CommonTlsContext: &auth.CommonTlsContext{
					TlsParams: &auth.TlsParameters{
						TlsMinimumProtocolVersion: 3,
						TlsMaximumProtocolVersion: 4,
					},
					TlsCertificates: nil,
					TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
						Name: "service-cert:default/bookbuyer",
						SdsConfig: &core.ConfigSource{
							ConfigSourceSpecifier: &core.ConfigSource_Ads{
								Ads: &core.AggregatedConfigSource{},
							},
							ResourceApiVersion: core.ApiVersion_V3,
						},
					}},
					ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
						ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
							Name: "root-cert-for-mtls-outbound:default/bookstore-v1",
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
				Sni:                sni, // "bookstore-v1.default.svc.cluster.local"
				AllowRenegotiation: false,
			}

			// Ensure the SNI is in the expected format!
			Expect(tlsContext.Sni).To(Equal(tests.BookstoreV1Service.ServerName()))
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
			tlsContext := GetUpstreamTLSContext(tests.BookbuyerServiceIdentity, tests.BookstoreV1Service, sidecarSpec)
			// To show the actual string for human comprehension
			Expect(tlsContext.Sni).To(Equal(tests.BookstoreV1Service.ServerName()))
		})
	})

	Context("Test getCommonTLSContext()", func() {
		It("returns proper auth.CommonTlsContext for outbound mTLS", func() {
			actual := getCommonTLSContext(secrets.NameForIdentity(identity.New("bookbuyer", "default")),
				secrets.NameForUpstreamService("bookstore-v1", "default"), sidecarSpec)

			expected := &auth.CommonTlsContext{
				TlsParams: GetTLSParams(sidecarSpec),
				TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
					Name:      "service-cert:default/bookbuyer",
					SdsConfig: GetADSConfigSource(),
				}},
				ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
					ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
						Name:      "root-cert-for-mtls-outbound:default/bookstore-v1",
						SdsConfig: GetADSConfigSource(),
					},
				},
				AlpnProtocols: nil,
			}

			Expect(actual).To(Equal(expected))
		})

		It("returns proper auth.CommonTlsContext for inbound mTLS", func() {
			actual := getCommonTLSContext(secrets.NameForIdentity(identity.New("bookstore-v1", "default")),
				secrets.NameForMTLSInbound, sidecarSpec)

			expected := &auth.CommonTlsContext{
				TlsParams: GetTLSParams(sidecarSpec),
				TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
					Name:      "service-cert:default/bookstore-v1",
					SdsConfig: GetADSConfigSource(),
				}},
				ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
					ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
						Name:      "root-cert-for-mtls-inbound",
						SdsConfig: GetADSConfigSource(),
					},
				},
				AlpnProtocols: nil,
			}

			Expect(actual).To(Equal(expected))
		})

		It("returns proper auth.CommonTlsContext for TLS (non-mTLS)", func() {
			actual := getCommonTLSContext(secrets.NameForIdentity(identity.New("bookstore-v1", "default")), "" /* no client cert validation */, sidecarSpec)

			expected := &auth.CommonTlsContext{
				TlsParams: GetTLSParams(sidecarSpec),
				TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
					Name:      "service-cert:default/bookstore-v1",
					SdsConfig: GetADSConfigSource(),
				}},
				ValidationContextType: nil, // TLS cert type should not validate the client certificate
			}

			Expect(actual).To(Equal(expected))
		})
	})
})

func TestGetCIDRRangeFromStr(t *testing.T) {
	testCases := []struct {
		name              string
		cidr              string
		expectedCIDRRange *xds_core.CidrRange
		expectErr         bool
	}{
		{
			name: "valid CIDR range",
			cidr: "10.0.0.0/10",
			expectedCIDRRange: &xds_core.CidrRange{
				AddressPrefix: "10.0.0.0",
				PrefixLen: &wrapperspb.UInt32Value{
					Value: 10,
				},
			},
			expectErr: false,
		},
		{
			name:              "invalid CIDR range",
			cidr:              "10.0.0.1",
			expectedCIDRRange: nil,
			expectErr:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual, err := GetCIDRRangeFromStr(tc.cidr)
			assert.Equal(tc.expectedCIDRRange, actual)
			assert.Equal(tc.expectErr, err != nil)
		})
	}
}
