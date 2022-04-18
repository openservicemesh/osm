package envoy

import (
	"fmt"
	"testing"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_accesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetAccessLog(t *testing.T) {
	assert := tassert.New(t)

	res := GetAccessLog()
	assert.NotNil(res)
}

func TestGetStdoutAccessLog(t *testing.T) {
	assert := tassert.New(t)

	expAccessLogger := &xds_accesslog.StdoutAccessLog{
		AccessLogFormat: &xds_accesslog.StdoutAccessLog_LogFormat{
			LogFormat: &xds_core.SubstitutionFormatString{
				Format: &xds_core.SubstitutionFormatString_JsonFormat{
					JsonFormat: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"start_time":            pbStringValue(`%START_TIME%`),
							"method":                pbStringValue(`%REQ(:METHOD)%`),
							"path":                  pbStringValue(`%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%`),
							"protocol":              pbStringValue(`%PROTOCOL%`),
							"response_code":         pbStringValue(`%RESPONSE_CODE%`),
							"response_code_details": pbStringValue(`%RESPONSE_CODE_DETAILS%`),
							"time_to_first_byte":    pbStringValue(`%RESPONSE_DURATION%`),
							"upstream_cluster":      pbStringValue(`%UPSTREAM_CLUSTER%`),
							"response_flags":        pbStringValue(`%RESPONSE_FLAGS%`),
							"bytes_received":        pbStringValue(`%BYTES_RECEIVED%`),
							"bytes_sent":            pbStringValue(`%BYTES_SENT%`),
							"duration":              pbStringValue(`%DURATION%`),
							"upstream_service_time": pbStringValue(`%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%`),
							"x_forwarded_for":       pbStringValue(`%REQ(X-FORWARDED-FOR)%`),
							"user_agent":            pbStringValue(`%REQ(USER-AGENT)%`),
							"request_id":            pbStringValue(`%REQ(X-REQUEST-ID)%`),
							"requested_server_name": pbStringValue("%REQUESTED_SERVER_NAME%"),
							"authority":             pbStringValue(`%REQ(:AUTHORITY)%`),
							"upstream_host":         pbStringValue(`%UPSTREAM_HOST%`),
						},
					},
				},
			},
		},
	}
	resAccessLogger := getStdoutAccessLog()

	assert.Equal(resAccessLogger, expAccessLogger)
}

var sidecarSpec = configv1alpha3.SidecarSpec{
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
							Name: secrets.SDSCert{
								Name:     "test/foo",
								CertType: secrets.RootCertTypeForMTLSInbound,
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

	Context("Test pbStringValue()", func() {
		It("returns structpb", func() {
			exp := &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: "apples",
				},
			}
			res := pbStringValue("apples")
			Expect(res).To(Equal(exp))
		})
	})

	Context("Test getCommonTLSContext()", func() {
		It("returns proper auth.CommonTlsContext for outbound mTLS", func() {
			tlsSDSCert := secrets.SDSCert{
				Name:     "default/bookbuyer",
				CertType: secrets.ServiceCertType,
			}
			peerValidationSDSCert := &secrets.SDSCert{
				Name:     "default/bookstore-v1",
				CertType: secrets.RootCertTypeForMTLSOutbound,
			}

			actual := getCommonTLSContext(tlsSDSCert, peerValidationSDSCert, sidecarSpec)

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
			tlsSDSCert := secrets.SDSCert{
				Name:     "default/bookstore-v1",
				CertType: secrets.ServiceCertType,
			}
			peerValidationSDSCert := &secrets.SDSCert{
				Name:     "default/bookstore-v1",
				CertType: secrets.RootCertTypeForMTLSInbound,
			}

			actual := getCommonTLSContext(tlsSDSCert, peerValidationSDSCert, sidecarSpec)

			expected := &auth.CommonTlsContext{
				TlsParams: GetTLSParams(sidecarSpec),
				TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
					Name:      "service-cert:default/bookstore-v1",
					SdsConfig: GetADSConfigSource(),
				}},
				ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
					ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
						Name:      "root-cert-for-mtls-inbound:default/bookstore-v1",
						SdsConfig: GetADSConfigSource(),
					},
				},
				AlpnProtocols: nil,
			}

			Expect(actual).To(Equal(expected))
		})

		It("returns proper auth.CommonTlsContext for TLS (non-mTLS)", func() {
			tlsSDSCert := secrets.SDSCert{
				Name:     "default/bookstore-v1",
				CertType: secrets.ServiceCertType,
			}

			actual := getCommonTLSContext(tlsSDSCert, nil /* no client cert validation */, sidecarSpec)

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

	Context("Test GetEnvoyServiceNodeID()", func() {
		It("", func() {
			actual := GetEnvoyServiceNodeID("-nodeID-", "-workload-kind-", "-workload-name-")
			expected := "$(POD_UID)/$(POD_NAMESPACE)/$(POD_IP)/$(SERVICE_ACCOUNT)/-nodeID-/$(POD_NAME)/-workload-kind-/-workload-name-"
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test ParseEnvoyServiceNodeID()", func() {
		It("", func() {
			serviceNodeID := GetEnvoyServiceNodeID("-nodeID-", "-workload-kind-", "-workload-name-")
			meta, err := ParseEnvoyServiceNodeID(serviceNodeID)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.UID).To(Equal("$(POD_UID)"))
			Expect(meta.Namespace).To(Equal("$(POD_NAMESPACE)"))
			Expect(meta.IP).To(Equal("$(POD_IP)"))
			Expect(meta.ServiceAccount.Name).To(Equal("$(SERVICE_ACCOUNT)"))
			Expect(meta.ServiceAccount.Namespace).To(Equal("$(POD_NAMESPACE)"))
			Expect(meta.EnvoyNodeID).To(Equal("-nodeID-"))
			Expect(meta.Name).To(Equal("$(POD_NAME)"))
			Expect(meta.WorkloadKind).To(Equal("-workload-kind-"))
			Expect(meta.WorkloadName).To(Equal("-workload-name-"))
		})

		It("handles when not all fields are defined", func() {
			serviceNodeID := "$(POD_UID)/$(POD_NAMESPACE)/$(POD_IP)/$(SERVICE_ACCOUNT)/-nodeID-"
			meta, err := ParseEnvoyServiceNodeID(serviceNodeID)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.UID).To(Equal("$(POD_UID)"))
			Expect(meta.Namespace).To(Equal("$(POD_NAMESPACE)"))
			Expect(meta.IP).To(Equal("$(POD_IP)"))
			Expect(meta.ServiceAccount.Name).To(Equal("$(SERVICE_ACCOUNT)"))
			Expect(meta.ServiceAccount.Namespace).To(Equal("$(POD_NAMESPACE)"))
			Expect(meta.EnvoyNodeID).To(Equal("-nodeID-"))
			Expect(meta.Name).To(Equal(""))
			Expect(meta.WorkloadKind).To(Equal(""))
			Expect(meta.WorkloadName).To(Equal(""))
		})

		It("should error when there are less than 5 chunks in the serviceNodeID string", func() {
			// this 'serviceNodeID' will yield 2 chunks
			serviceNodeID := "$(POD_UID)/$(POD_NAMESPACE)"
			_, err := ParseEnvoyServiceNodeID(serviceNodeID)
			Expect(err).To(HaveOccurred())
		})
	})
})

func TestGetKindFromProxyCertificate(t *testing.T) {
	assert := tassert.New(t)
	cn := certificate.CommonName("fcbd7396-2e8c-49dc-91ff-7267d81287ba.gateway.2.3.4.5.6.7.8")
	actualProxyKind, err := GetKindFromProxyCertificate(cn)
	assert.Nil(err, fmt.Sprintf("Expected err to be nil; Actually it was %+v", err))
	expectedProxyKind := KindGateway
	assert.Equal(expectedProxyKind, actualProxyKind)
}

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
