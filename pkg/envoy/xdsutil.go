package envoy

import (
	"net"
	"strings"

	xds_accesslog_filter "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_accesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	v1 "k8s.io/api/core/v1"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	// ErrInvalidCertificateCN is an error for when a certificate has a CommonName, which does not match expected string format.
	ErrInvalidCertificateCN = errors.New("invalid cn")

	// ErrMoreThanOnePodForCertificate is an error for when OSM finds more than one pod for a given xDS certificate. There should always be exactly one Pod for a given xDS certificate.
	ErrMoreThanOnePodForCertificate = errors.New("found more than one pod for xDS certificate")

	// ErrDidNotFindPodForCertificate is an error for when OSM cannot not find a pod for the given xDS certificate.
	ErrDidNotFindPodForCertificate = errors.New("did not find pod for certificate")

	// ErrServiceAccountDoesNotMatchCertificate is an error for when the service account of a Pod does not match the xDS certificate.
	ErrServiceAccountDoesNotMatchCertificate = errors.New("service account does not match certificate")

	// ErrNamespaceDoesNotMatchCertificate is an error for when the namespace of the Pod does not match the xDS certificate.
	ErrNamespaceDoesNotMatchCertificate = errors.New("namespace does not match certificate")
)

const (
	// TransportProtocolTLS is the TLS transport protocol used in Envoy configurations
	TransportProtocolTLS = "tls"

	// OutboundPassthroughCluster is the outbound passthrough cluster name
	OutboundPassthroughCluster = "passthrough-outbound"

	// AccessLoggerName is name used for the envoy access loggers.
	AccessLoggerName = "envoy.access_loggers.stream"

	// MulticlusterGatewayCluster is the tls passthough cluster name for multicluster gateway
	MulticlusterGatewayCluster = "passthrough-multicluster-gateway"
)

// ALPNInMesh indicates that the proxy is connecting to an in-mesh destination.
// It is set as a part of configuring the UpstreamTLSContext.
var ALPNInMesh = []string{"osm"}

// GetAddress creates an Envoy Address struct.
func GetAddress(address string, port uint32) *xds_core.Address {
	return &xds_core.Address{
		Address: &xds_core.Address_SocketAddress{
			SocketAddress: &xds_core.SocketAddress{
				Protocol: xds_core.SocketAddress_TCP,
				Address:  address,
				PortSpecifier: &xds_core.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
}

// GetTLSParams creates Envoy TlsParameters struct.
func GetTLSParams(sidecarSpec configv1alpha3.SidecarSpec) *xds_auth.TlsParameters {
	minVersionInt := xds_auth.TlsParameters_TlsProtocol_value[sidecarSpec.TLSMinProtocolVersion]
	maxVersionInt := xds_auth.TlsParameters_TlsProtocol_value[sidecarSpec.TLSMaxProtocolVersion]
	tlsMinVersion := xds_auth.TlsParameters_TlsProtocol(minVersionInt)
	tlsMaxVersion := xds_auth.TlsParameters_TlsProtocol(maxVersionInt)

	tlsParams := &xds_auth.TlsParameters{
		TlsMinimumProtocolVersion: tlsMinVersion,
		TlsMaximumProtocolVersion: tlsMaxVersion,
	}
	if len(sidecarSpec.CipherSuites) > 0 {
		tlsParams.CipherSuites = sidecarSpec.CipherSuites
	}
	if len(sidecarSpec.ECDHCurves) > 0 {
		tlsParams.EcdhCurves = sidecarSpec.ECDHCurves
	}

	return tlsParams
}

// GetAccessLog creates an Envoy AccessLog struct.
func GetAccessLog() []*xds_accesslog_filter.AccessLog {
	accessLog, err := anypb.New(getStdoutAccessLog())
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling AccessLog object")
		return nil
	}
	return []*xds_accesslog_filter.AccessLog{{
		Name: AccessLoggerName,
		ConfigType: &xds_accesslog_filter.AccessLog_TypedConfig{
			TypedConfig: accessLog,
		}},
	}
}

func getStdoutAccessLog() *xds_accesslog.StdoutAccessLog {
	accessLogger := &xds_accesslog.StdoutAccessLog{
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
	return accessLogger
}

func pbStringValue(v string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StringValue{
			StringValue: v,
		},
	}
}

// getCommonTLSContext returns a CommonTlsContext type for a given 'tlsSDSCert' and 'peerValidationSDSCert' pair.
// 'tlsSDSCert' determines the SDS Secret config used to present the TLS certificate.
// 'peerValidationSDSCert' determines the SDS Secret configs used to validate the peer TLS certificate. A nil value
// is used to indicate peer certificate validation should be skipped, and is used when mTLS is disabled (ex. with TLS
// based ingress).
// 'sidecarSpec' is the sidecar section of MeshConfig.
func getCommonTLSContext(tlsSDSCert secrets.SDSCert, peerValidationSDSCert *secrets.SDSCert, sidecarSpec configv1alpha3.SidecarSpec) *xds_auth.CommonTlsContext {
	commonTLSContext := &xds_auth.CommonTlsContext{
		TlsParams: GetTLSParams(sidecarSpec),
		TlsCertificateSdsSecretConfigs: []*xds_auth.SdsSecretConfig{{
			// Example ==> Name: "service-cert:NameSpaceHere/ServiceNameHere"
			Name:      tlsSDSCert.String(),
			SdsConfig: GetADSConfigSource(),
		}},
	}

	// For TLS (non-mTLS) based validation, the client certificate should not be validated and the
	// 'peerValidationSDSCert' will be set to nil to indicate this.
	if peerValidationSDSCert != nil {
		commonTLSContext.ValidationContextType = &xds_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
			ValidationContextSdsSecretConfig: &xds_auth.SdsSecretConfig{
				// Example ==> Name: "root-cert<type>:NameSpaceHere/ServiceNameHere"
				Name:      peerValidationSDSCert.String(),
				SdsConfig: GetADSConfigSource(),
			},
		}
	}

	return commonTLSContext
}

// GetDownstreamTLSContext creates a downstream Envoy TLS Context to be configured on the upstream for the given upstream's identity
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func GetDownstreamTLSContext(upstreamIdentity identity.ServiceIdentity, mTLS bool, sidecarSpec configv1alpha3.SidecarSpec) *xds_auth.DownstreamTlsContext {
	upstreamSDSCert := secrets.SDSCert{
		Name:     secrets.GetSecretNameForIdentity(upstreamIdentity),
		CertType: secrets.ServiceCertType,
	}

	var downstreamPeerValidationSDSCert *secrets.SDSCert
	if mTLS {
		// The downstream peer validation SDS cert points to a cert with the name 'upstreamIdentity' only
		// because we use a single DownstreamTlsContext for all inbound traffic to the given upstream with the identity 'upstreamIdentity'.
		// This single DownstreamTlsContext is used to validate all allowed inbound SANs. The
		// 'RootCertTypeForMTLSInbound' cert type is used for in-mesh downstreams.
		downstreamPeerValidationSDSCert = &secrets.SDSCert{
			Name:     secrets.GetSecretNameForIdentity(upstreamIdentity),
			CertType: secrets.RootCertTypeForMTLSInbound,
		}
	} else {
		// When 'mTLS' is disabled, the upstream should not validate the certificate presented by the downstream.
		// This is used for HTTPS ingress with mTLS disabled.
		downstreamPeerValidationSDSCert = nil
	}

	tlsConfig := &xds_auth.DownstreamTlsContext{
		CommonTlsContext: getCommonTLSContext(upstreamSDSCert, downstreamPeerValidationSDSCert, sidecarSpec),
		// When RequireClientCertificate is enabled trusted CA certs must be provided via ValidationContextType
		RequireClientCertificate: &wrappers.BoolValue{Value: mTLS},
	}
	return tlsConfig
}

// GetUpstreamTLSContext creates an upstream Envoy TLS Context for the given downstream identity and upstream service pair
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func GetUpstreamTLSContext(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService, sidecarSpec configv1alpha3.SidecarSpec) *xds_auth.UpstreamTlsContext {
	downstreamSDSCert := secrets.SDSCert{
		Name:     secrets.GetSecretNameForIdentity(downstreamIdentity),
		CertType: secrets.ServiceCertType,
	}
	upstreamPeerValidationSDSCert := &secrets.SDSCert{
		Name:     upstreamSvc.String(),
		CertType: secrets.RootCertTypeForMTLSOutbound,
	}
	commonTLSContext := getCommonTLSContext(downstreamSDSCert, upstreamPeerValidationSDSCert, sidecarSpec)

	// Advertise in-mesh using UpstreamTlsContext.CommonTlsContext.AlpnProtocols
	commonTLSContext.AlpnProtocols = ALPNInMesh
	tlsConfig := &xds_auth.UpstreamTlsContext{
		CommonTlsContext: commonTLSContext,

		// The Sni field is going to be used to do FilterChainMatch in getInboundMeshHTTPFilterChain()
		// The "Sni" field below of an incoming request will be matched against a list of server names
		// in FilterChainMatch.ServerNames
		Sni: upstreamSvc.ServerName(),
	}
	return tlsConfig
}

// GetADSConfigSource creates an Envoy ConfigSource struct.
func GetADSConfigSource() *xds_core.ConfigSource {
	return &xds_core.ConfigSource{
		ConfigSourceSpecifier: &xds_core.ConfigSource_Ads{
			Ads: &xds_core.AggregatedConfigSource{},
		},
		ResourceApiVersion: xds_core.ApiVersion_V3,
	}
}

// GetEnvoyServiceNodeID creates the string for Envoy's "--service-node" CLI argument for the Kubernetes sidecar container Command/Args
func GetEnvoyServiceNodeID(nodeID, workloadKind, workloadName string) string {
	items := []string{
		"$(POD_UID)",
		"$(POD_NAMESPACE)",
		"$(POD_IP)",
		"$(SERVICE_ACCOUNT)",
		nodeID,
		"$(POD_NAME)",
		workloadKind,
		workloadName,
	}

	return strings.Join(items, constants.EnvoyServiceNodeSeparator)
}

// ParseEnvoyServiceNodeID parses the given Envoy service node ID and returns the encoded metadata
func ParseEnvoyServiceNodeID(serviceNodeID string) (*PodMetadata, error) {
	chunks := strings.Split(serviceNodeID, constants.EnvoyServiceNodeSeparator)

	if len(chunks) < 5 {
		return nil, errors.New("invalid envoy service node id format")
	}

	meta := &PodMetadata{
		UID:            chunks[0],
		Namespace:      chunks[1],
		IP:             chunks[2],
		ServiceAccount: identity.K8sServiceAccount{Name: chunks[3], Namespace: chunks[1]},
		EnvoyNodeID:    chunks[4],
	}

	if len(chunks) >= 8 {
		meta.Name = chunks[5]
		meta.WorkloadKind = chunks[6]
		meta.WorkloadName = chunks[7]
	}

	return meta, nil
}

// certificateCommonNameMeta is the type that stores the metadata present in the CommonName field in a proxy's certificate
type certificateCommonNameMeta struct {
	ProxyUUID uuid.UUID
	ProxyKind ProxyKind
	// TODO(draychev): Change this to ServiceIdentity type (instead of string)
	ServiceIdentity identity.ServiceIdentity
}

func getCertificateCommonNameMeta(cn certificate.CommonName) (*certificateCommonNameMeta, error) {
	// XDS cert CN is of the form <proxy-UUID>.<kind>.<proxy-identity>
	chunks := strings.SplitN(cn.String(), constants.DomainDelimiter, 3)
	if len(chunks) < 3 {
		return nil, ErrInvalidCertificateCN
	}
	proxyUUID, err := uuid.Parse(chunks[0])
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrParsingXDSCertCN)).
			Msgf("Error parsing %s into uuid.UUID", chunks[0])
		return nil, err
	}

	return &certificateCommonNameMeta{
		ProxyUUID:       proxyUUID,
		ProxyKind:       ProxyKind(chunks[1]),
		ServiceIdentity: identity.ServiceIdentity(chunks[2]),
	}, nil
}

// GetPodFromCertificate returns the Kubernetes Pod object for a given certificate.
func GetPodFromCertificate(cn certificate.CommonName, kubecontroller k8s.Controller) (*v1.Pod, error) {
	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return nil, err
	}

	log.Trace().Msgf("Looking for pod with label %q=%q", constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID)
	podList := kubecontroller.ListPods()
	var pods []v1.Pod
	for _, pod := range podList {
		if pod.Namespace != cnMeta.ServiceIdentity.ToK8sServiceAccount().Namespace {
			continue
		}
		if uuid, labelFound := pod.Labels[constants.EnvoyUniqueIDLabelName]; labelFound && uuid == cnMeta.ProxyUUID.String() {
			pods = append(pods, *pod)
		}
	}

	if len(pods) == 0 {
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingPodFromCert)).
			Msgf("Did not find Pod with label %s = %s in namespace %s",
				constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID, cnMeta.ServiceIdentity.ToK8sServiceAccount().Namespace)
		return nil, ErrDidNotFindPodForCertificate
	}

	// Each pod is assigned a unique UUID at the time of sidecar injection.
	// The certificate's CommonName encodes this UUID, and we lookup the pod
	// whose label matches this UUID.
	// Only 1 pod must match the UUID encoded in the given certificate. If multiple
	// pods match, it is an error.
	if len(pods) > 1 {
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrPodBelongsToMultipleServices)).
			Msgf("Found more than one pod with label %s = %s in namespace %s. There can be only one!",
				constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID, cnMeta.ServiceIdentity.ToK8sServiceAccount().Namespace)
		return nil, ErrMoreThanOnePodForCertificate
	}

	pod := pods[0]
	log.Trace().Msgf("Found Pod with UID=%s for proxyID %s", pod.ObjectMeta.UID, cnMeta.ProxyUUID)

	// Ensure the Namespace encoded in the certificate matches that of the Pod
	if pod.Namespace != cnMeta.ServiceIdentity.ToK8sServiceAccount().Namespace {
		log.Warn().Msgf("Pod with UID=%s belongs to Namespace %s. The pod's xDS certificate was issued for Namespace %s",
			pod.ObjectMeta.UID, pod.Namespace, cnMeta.ServiceIdentity.ToK8sServiceAccount().Namespace)
		return nil, ErrNamespaceDoesNotMatchCertificate
	}

	// Ensure the Name encoded in the certificate matches that of the Pod
	// TODO(draychev): check that the Kind matches too! [https://github.com/openservicemesh/osm/issues/3173]
	if pod.Spec.ServiceAccountName != cnMeta.ServiceIdentity.ToK8sServiceAccount().Name {
		// Since we search for the pod in the namespace we obtain from the certificate -- these namespaces will always match.
		log.Warn().Msgf("Pod with UID=%s belongs to ServiceAccount=%s. The pod's xDS certificate was issued for ServiceAccount=%s",
			pod.ObjectMeta.UID, pod.Spec.ServiceAccountName, cnMeta.ServiceIdentity.ToK8sServiceAccount())
		return nil, ErrServiceAccountDoesNotMatchCertificate
	}

	return &pod, nil
}

// GetServiceIdentityFromProxyCertificate returns the ServiceIdentity information encoded in the XDS certificate CN
func GetServiceIdentityFromProxyCertificate(cn certificate.CommonName) (identity.ServiceIdentity, error) {
	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return "", err
	}

	return cnMeta.ServiceIdentity, nil
}

// GetKindFromProxyCertificate returns the proxy kind, which is encoded in the Common Name of the XDS certificate.
func GetKindFromProxyCertificate(cn certificate.CommonName) (ProxyKind, error) {
	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return "", err
	}

	return cnMeta.ProxyKind, nil
}

// GetCIDRRangeFromStr converts the given CIDR as a string to an XDS CidrRange object
func GetCIDRRangeFromStr(cidr string) (*xds_core.CidrRange, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	prefixLen, _ := ipNet.Mask.Size()
	return &xds_core.CidrRange{
		AddressPrefix: ip.String(),
		PrefixLen: &wrapperspb.UInt32Value{
			Value: uint32(prefixLen),
		},
	}, nil
}
