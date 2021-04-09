package envoy

import (
	"testing"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_accesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	tassert "github.com/stretchr/testify/assert"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetLocalClusterNameForService(t *testing.T) {
	assert := tassert.New(t)

	actual := GetLocalClusterNameForService(tests.BookbuyerService)
	assert.Equal(actual, "default/bookbuyer-local")
}

func TestGetAccessLog(t *testing.T) {
	assert := tassert.New(t)

	res := GetAccessLog()
	assert.NotNil(res)
}

func TestGetFileAccessLog(t *testing.T) {
	assert := tassert.New(t)

	expAccessLogger := &xds_accesslog.FileAccessLog{
		Path: accessLogPath,
		AccessLogFormat: &xds_accesslog.FileAccessLog_LogFormat{
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
	resAccessLogger := getFileAccessLog()

	assert.Equal(resAccessLogger, expAccessLogger)
}

var _ = Describe("Test Envoy tools", func() {
	Context("Test GetLocalClusterNameForServiceCluster", func() {
		It("", func() {
			clusterName := "-cluster-name-"
			actual := GetLocalClusterNameForServiceCluster(clusterName)
			expected := "-cluster-name--local"
			Expect(actual).To(Equal(expected))
		})
	})

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

	Context("Test UnmarshalSDSCert()", func() {
		It("Interface marshals and unmarshals preserving the exact same data", func() {
			InitialObj := SDSCert{
				CertType: ServiceCertType,
				Name:     "test-namespace/test-service",
			}

			// Marshal/stringify it
			marshaledStr := InitialObj.String()

			// Unmarshal it back from the string
			finalObj, _ := UnmarshalSDSCert(marshaledStr)

			// First and final object must be equal
			Expect(*finalObj).To(Equal(InitialObj))
		})

		It("returns service cert", func() {
			actual, err := UnmarshalSDSCert("service-cert:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(ServiceCertType))
			Expect(actual.Name).To(Equal("namespace-test/blahBlahBlahCert"))
		})
		It("returns root cert for mTLS", func() {
			actual, err := UnmarshalSDSCert("root-cert-for-mtls-outbound:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(RootCertTypeForMTLSOutbound))
			Expect(actual.Name).To(Equal("namespace-test/blahBlahBlahCert"))

		})

		It("returns root cert for non-mTLS", func() {
			actual, err := UnmarshalSDSCert("root-cert-https:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(RootCertTypeForHTTPS))
			Expect(actual.Name).To(Equal("namespace-test/blahBlahBlahCert"))
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

		It("returns an error (invalid serv type)", func() {
			_, err := UnmarshalSDSCert("revoked-cert:blah/BlahBlahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (invalid mtls cert type)", func() {
			_, err := UnmarshalSDSCert("oot-cert-for-mtls-diagonalstream:blah/BlahBlahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (empty slice)", func() {
			_, err := UnmarshalSDSCert(":")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test GetDownstreamTLSContext()", func() {
		It("should return TLS context", func() {
			tlsContext := GetDownstreamTLSContext(service.K8sServiceAccount{Name: "foo", Namespace: "test"}, true)

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
							Name: SDSCert{
								Name:     "test/foo",
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
			tlsContext := GetDownstreamTLSContext(tests.BookstoreServiceAccount, true)
			Expect(tlsContext.RequireClientCertificate).To(Equal(&wrappers.BoolValue{Value: true}))
		})
	})

	Context("Test GetDownstreamTLSContext() for TLS", func() {
		It("should return TLS context with client certificate validation disabled", func() {
			tlsContext := GetDownstreamTLSContext(tests.BookstoreServiceAccount, false)
			Expect(tlsContext.RequireClientCertificate).To(Equal(&wrappers.BoolValue{Value: false}))
		})
	})

	Context("Test GetUpstreamTLSContext()", func() {
		It("should return TLS context", func() {
			sni := "bookstore-v1.default.svc.cluster.local"
			tlsContext := GetUpstreamTLSContext(tests.BookbuyerServiceAccount, tests.BookstoreV1Service)

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
			tlsContext := GetUpstreamTLSContext(tests.BookbuyerServiceAccount, tests.BookstoreV1Service)
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
			tlsSDSCert := SDSCert{
				Name:     "default/bookbuyer",
				CertType: ServiceCertType,
			}
			peerValidationSDSCert := SDSCert{
				Name:     "default/bookstore-v1",
				CertType: RootCertTypeForMTLSOutbound,
			}

			actual := getCommonTLSContext(tlsSDSCert, peerValidationSDSCert)

			expected := &auth.CommonTlsContext{
				TlsParams: GetTLSParams(),
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
			tlsSDSCert := SDSCert{
				Name:     "default/bookstore-v1",
				CertType: ServiceCertType,
			}
			peerValidationSDSCert := SDSCert{
				Name:     "default/bookstore-v1",
				CertType: RootCertTypeForMTLSInbound,
			}

			actual := getCommonTLSContext(tlsSDSCert, peerValidationSDSCert)

			expected := &auth.CommonTlsContext{
				TlsParams: GetTLSParams(),
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

		It("returns proper auth.CommonTlsContext for non-mTLS (HTTPS)", func() {
			tlsSDSCert := SDSCert{
				Name:     "default/bookstore-v1",
				CertType: ServiceCertType,
			}
			peerValidationSDSCert := SDSCert{
				Name:     "default/bookstore-v1",
				CertType: RootCertTypeForHTTPS,
			}

			actual := getCommonTLSContext(tlsSDSCert, peerValidationSDSCert)

			expected := &auth.CommonTlsContext{
				TlsParams: GetTLSParams(),
				TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
					Name:      "service-cert:default/bookstore-v1",
					SdsConfig: GetADSConfigSource(),
				}},
				ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
					ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
						Name:      "root-cert-https:default/bookstore-v1",
						SdsConfig: GetADSConfigSource(),
					},
				},
				AlpnProtocols: nil,
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
