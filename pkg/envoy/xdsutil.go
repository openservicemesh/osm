package envoy

import (
	"fmt"
	"strings"

	xds_accesslog_filter "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_accesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	extensions_upstream_http_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/kubernetes"
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
func GetTLSParams() *xds_auth.TlsParameters {
	return &xds_auth.TlsParameters{
		TlsMinimumProtocolVersion: xds_auth.TlsParameters_TLSv1_2,
		TlsMaximumProtocolVersion: xds_auth.TlsParameters_TLSv1_3,
	}
}

// GetAccessLog creates an Envoy AccessLog struct.
func GetAccessLog() []*xds_accesslog_filter.AccessLog {
	accessLog, err := ptypes.MarshalAny(getStdoutAccessLog())
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling AccessLog object")
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
// 'peerValidationSDSCert' determines the SDS Secret configs used to validate the peer TLS certificate.
func getCommonTLSContext(tlsSDSCert, peerValidationSDSCert secrets.SDSCert) *xds_auth.CommonTlsContext {
	return &xds_auth.CommonTlsContext{
		TlsParams: GetTLSParams(),
		TlsCertificateSdsSecretConfigs: []*xds_auth.SdsSecretConfig{{
			// Example ==> Name: "service-cert:NameSpaceHere/ServiceNameHere"
			Name:      tlsSDSCert.String(),
			SdsConfig: GetADSConfigSource(),
		}},
		ValidationContextType: &xds_auth.CommonTlsContext_ValidationContextSdsSecretConfig{
			ValidationContextSdsSecretConfig: &xds_auth.SdsSecretConfig{
				// Example ==> Name: "root-cert<type>:NameSpaceHere/ServiceNameHere"
				Name:      peerValidationSDSCert.String(),
				SdsConfig: GetADSConfigSource(),
			},
		},
	}
}

// GetDownstreamTLSContext creates a downstream Envoy TLS Context to be configured on the upstream for the given upstream's identity
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func GetDownstreamTLSContext(upstreamIdentity identity.ServiceIdentity, mTLS bool) *xds_auth.DownstreamTlsContext {
	upstreamSDSCert := secrets.SDSCert{
		Name:     upstreamIdentity.GetSDSCSecretName(),
		CertType: secrets.ServiceCertType,
	}

	var downstreamPeerValidationCertType secrets.SDSCertType
	if mTLS {
		// Perform SAN validation for downstream client certificates
		downstreamPeerValidationCertType = secrets.RootCertTypeForMTLSInbound
	} else {
		// TLS based cert validation (used for ingress)
		downstreamPeerValidationCertType = secrets.RootCertTypeForHTTPS
	}
	// The downstream peer validation SDS cert points to a cert with the name 'upstreamIdentity' only
	// because we use a single DownstreamTlsContext for all inbound traffic to the given upstream with the identity 'upstreamIdentity'.
	// This single DownstreamTlsContext is used to validate all allowed inbound SANs. The
	// 'RootCertTypeForMTLSInbound' cert type used for in-mesh downstreams, while 'RootCertTypeForHTTPS'
	// cert type is used for non-mesh downstreams such as ingress.
	downstreamPeerValidationSDSCert := secrets.SDSCert{
		Name:     upstreamIdentity.GetSDSCSecretName(),
		CertType: downstreamPeerValidationCertType,
	}

	tlsConfig := &xds_auth.DownstreamTlsContext{
		CommonTlsContext: getCommonTLSContext(upstreamSDSCert, downstreamPeerValidationSDSCert),
		// When RequireClientCertificate is enabled trusted CA certs must be provided via ValidationContextType
		RequireClientCertificate: &wrappers.BoolValue{Value: mTLS},
	}
	return tlsConfig
}

// GetUpstreamTLSContext creates an upstream Envoy TLS Context for the given downstream identity and upstream service pair
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func GetUpstreamTLSContext(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService) *xds_auth.UpstreamTlsContext {
	downstreamSDSCert := secrets.SDSCert{
		Name:     downstreamIdentity.GetSDSCSecretName(),
		CertType: secrets.ServiceCertType,
	}
	upstreamPeerValidationSDSCert := secrets.SDSCert{
		Name:     upstreamSvc.NameWithoutCluster(),
		CertType: secrets.RootCertTypeForMTLSOutbound,
	}
	commonTLSContext := getCommonTLSContext(downstreamSDSCert, upstreamPeerValidationSDSCert)

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

// GetHTTP2ProtocolOptions creates an Envoy http configuration that matches the downstream protocol
func GetHTTP2ProtocolOptions() (map[string]*any.Any, error) {
	marshalledHTTPProtocolOptions, err := ptypes.MarshalAny(
		&extensions_upstream_http_v3.HttpProtocolOptions{
			UpstreamProtocolOptions: &extensions_upstream_http_v3.HttpProtocolOptions_UseDownstreamProtocolConfig{
				UseDownstreamProtocolConfig: &extensions_upstream_http_v3.HttpProtocolOptions_UseDownstreamHttpConfig{
					Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
				},
			},
		})
	if err != nil {
		return nil, err
	}

	return map[string]*any.Any{
		"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": marshalledHTTPProtocolOptions,
	}, nil
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

// GetLocalClusterNameForService returns the name of the local cluster for the given service.
// The local cluster refers to the cluster corresponding to the service the proxy is fronting, accessible over localhost by the proxy.
func GetLocalClusterNameForService(proxyService service.MeshService) string {
	return GetLocalClusterNameForServiceCluster(proxyService.NameWithoutCluster())
}

// GetLocalClusterNameForServiceCluster returns the name of the local cluster for the given service cluster.
// The local cluster refers to the cluster corresponding to the service the proxy is fronting, accessible over localhost by the proxy.
func GetLocalClusterNameForServiceCluster(clusterName string) string {
	return fmt.Sprintf("%s%s", clusterName, localClusterSuffix)
}

// certificateCommonNameMeta is the type that stores the metadata present in the CommonName field in a proxy's certificate
type certificateCommonNameMeta struct {
	ProxyUUID uuid.UUID
	ProxyKind ProxyKind
	// TODO(draychev): Change this to ServiceIdentity type (instead of string)
	ServiceAccount string
	Namespace      string
}

func getCertificateCommonNameMeta(cn certificate.CommonName) (*certificateCommonNameMeta, error) {
	chunks := strings.Split(cn.String(), constants.DomainDelimiter)
	if len(chunks) < 4 {
		return nil, ErrInvalidCertificateCN
	}
	proxyUUID, err := uuid.Parse(chunks[0])
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing %s into uuid.UUID", chunks[0])
		return nil, err
	}

	return &certificateCommonNameMeta{
		ProxyUUID: proxyUUID,
		ProxyKind: ProxyKind(chunks[1]),
		// TODO(draychev): Use ServiceIdentity vs ServiceAccount
		ServiceAccount: chunks[2],
		Namespace:      chunks[3],
	}, nil
}

// GetPodFromCertificate returns the Kubernetes Pod object for a given certificate.
func GetPodFromCertificate(cn certificate.CommonName, kubecontroller kubernetes.Controller) (*v1.Pod, error) {
	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return nil, err
	}

	log.Trace().Msgf("Looking for pod with label %q=%q", constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID)
	podList := kubecontroller.ListPods()
	var pods []v1.Pod
	for _, pod := range podList {
		if pod.Namespace != cnMeta.Namespace {
			continue
		}
		if uuid, labelFound := pod.Labels[constants.EnvoyUniqueIDLabelName]; labelFound && uuid == cnMeta.ProxyUUID.String() {
			pods = append(pods, *pod)
		}
	}

	if len(pods) == 0 {
		log.Error().Msgf("Did not find Pod with label %s = %s in namespace %s",
			constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID, cnMeta.Namespace)
		return nil, ErrDidNotFindPodForCertificate
	}

	// --- CONVENTION ---
	// By Open Service Mesh convention the number of services a pod can belong to is 1
	// This is a limitation we set in place in order to make the mesh easy to understand and reason about.
	// When a pod belongs to more than one service XDS will not program the Envoy proxy, leaving it out of the mesh.
	if len(pods) > 1 {
		log.Error().Msgf("Found more than one pod with label %s = %s in namespace %s. There can be only one!",
			constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID, cnMeta.Namespace)
		return nil, ErrMoreThanOnePodForCertificate
	}

	pod := pods[0]
	log.Trace().Msgf("Found Pod with UID=%s for proxyID %s", pod.ObjectMeta.UID, cnMeta.ProxyUUID)

	// Ensure the Namespace encoded in the certificate matches that of the Pod
	if pod.Namespace != cnMeta.Namespace {
		log.Warn().Msgf("Pod with UID=%s belongs to Namespace %s. The pod's xDS certificate was issued for Namespace %s",
			pod.ObjectMeta.UID, pod.Namespace, cnMeta.Namespace)
		return nil, ErrNamespaceDoesNotMatchCertificate
	}

	// Ensure the Name encoded in the certificate matches that of the Pod
	// TODO(draychev): check that the Kind matches too! [https://github.com/openservicemesh/osm/issues/3173]
	if pod.Spec.ServiceAccountName != cnMeta.ServiceAccount {
		// Since we search for the pod in the namespace we obtain from the certificate -- these namespaces will always match.
		log.Warn().Msgf("Pod with UID=%s belongs to ServiceAccount=%s. The pod's xDS certificate was issued for ServiceAccount=%s",
			pod.ObjectMeta.UID, pod.Spec.ServiceAccountName, cnMeta.ServiceAccount)
		return nil, ErrServiceAccountDoesNotMatchCertificate
	}

	return &pod, nil
}

// GetServiceAccountFromProxyCertificate returns the ServiceAccount information encoded in the certificate CN
func GetServiceAccountFromProxyCertificate(cn certificate.CommonName) (identity.K8sServiceAccount, error) {
	var svcAccount identity.K8sServiceAccount
	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return svcAccount, err
	}

	svcAccount.Name = cnMeta.ServiceAccount
	svcAccount.Namespace = cnMeta.Namespace

	return svcAccount, nil
}
