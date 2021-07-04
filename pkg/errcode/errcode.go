// Package errcode defines the error codes for error messages and an explanation
// of what the error signifies.
package errcode

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// ErrCode defines the type to represent error codes
type ErrCode int

const (
	// Kind defines the kind for the error code constants
	Kind = "error_code"
)

// Range 1000-1050 is reserved for errors related to application startup or bootstrapping
const (
	// ErrInvalidCLIArgument indicates an invalid CLI argument
	ErrInvalidCLIArgument ErrCode = iota + 1000

	// ErrSettingLogLevel indicates the specified log level could not be set
	ErrSettingLogLevel

	// ErrParsingMeshConfig indicates the MeshConfig resource could not be parsed
	ErrParsingMeshConfig

	// ErrFetchingControllerPod indicates the osm-controller pod resource could not be fetched
	ErrFetchingControllerPod

	// ErrFetchingInjectorPod indicates the osm-injector pod resource could not be fetched
	ErrFetchingInjectorPod

	// ErrStartingReconcileManager indicates the controller-runtime Manager failed to start
	ErrStartingReconcileManager
)

// Range 2000-2500 is reserved for errors related to traffic policies
const (
	// ErrDedupEgressTrafficMatches indicates an error related to deduplicating egress traffic matches
	ErrDedupEgressTrafficMatches ErrCode = iota + 2000

	// ErrDedupEgressClusterConfigs indicates an error related to deduplicating egress cluster configs
	ErrDedupEgressClusterConfigs

	// ErrInvalidEgressIPRange indicates the IP address range specified in an egress policy is invalid
	ErrInvalidEgressIPRange

	// ErrInvalidEgressMatches indicates the matches specified in an egress policy is invalid
	ErrInvalidEgressMatches

	// ErrEgressSMIHTTPRouteGroupNotFound indicates the SMI HTTPRouteGroup specified in the egress policy was not found
	ErrEgressSMIHTTPRouteGroupNotFound

	// ErrFetchingSMIHTTPRouteGroupForTrafficTarget indicates the SMI HTTPRouteGroup specified as a match in an SMI
	// TrafficTarget resource was not able to be retrieved
	ErrFetchingSMIHTTPRouteGroupForTrafficTarget

	// ErrSMIHTTPRouteGroupNoMatch indicates the SMI HTTPRouteGroup resource has no matches specified
	ErrSMIHTTPRouteGroupNoMatch

	// ErrMultipleSMISplitPerServiceUnsupported indicates multiple SMI split policies per service exists and is not supported
	ErrMultipleSMISplitPerServiceUnsupported

	// ErrAddingRouteToOutboundTrafficPolicy indicates there was an error adding a route to an outbound traffic policy
	ErrAddingRouteToOutboundTrafficPolicy

	// ErrFetchingServiceForTrafficTargetDestination indicates an error retrieving services associated with a TrafficTarget destination
	ErrFetchingServiceForTrafficTargetDestination
)

// Range 3000-3500 is reserved for errors related to k8s constructs (service accounts, namespaces, etc.)
const (
	// ErrServiceHostnames indicates the hostnames associated with a service could not be computed
	ErrServiceHostnames ErrCode = iota + 3000

	// ErrNoMatchingServiceForServiceAccount indicates there are no services associated with the service account
	ErrNoMatchingServiceForServiceAccount
)

// Range 4000-4100 reserved for errors related to certificate providers
const (
	// ErrFetchingCertSecret indicates a secret containing a certificate could not be fetched
	ErrFetchingCertSecret ErrCode = iota + 4000

	// ErrObtainingCertFrom indicates a certificate could not be obtained from a secret
	ErrObtainingCertFromSecret

	// ErrObtainingPrivateKeyFromSecret indicates the certificate's private key could not be obtained from a secret
	ErrObtainingPrivateKeyFromSecret

	// ErrObtainingCertExpirationFromSecret indicates the certificate's expiration could not be obtained from a secret
	ErrObtainingCertExpirationFromSecret

	// ErrParsingCertExpiration indicates the certificate's expiration could not be parsed
	ErrParsingCertExpiration

	// ErrCreatingCertSecret indicates the secret to containing a certificate could not be created
	ErrCreatingCertSecret

	// ErrGeneratingPrivateKey indicates a private key could not be generated
	ErrGeneratingPrivateKey

	// ErrEncodingKeyDERtoPEM indicates a private key could not be encoded from DER to PEM
	ErrEncodingKeyDERtoPEM

	// ErrCreatingCertReq indicates a certificate request could not be created
	ErrCreatingCertReq

	// ErrDeletingcertReq indicates a certificate request could not be deleted
	ErrDeletingCertReq

	// ErrdeletingCertReq inicates that the issue certificate request could not be deleted
	ErrCreatingRootCert

	// ErrEncodingCertDERtoPEM indicates a certificate could not be encoded from DER to PEM
	ErrEncodingCertDERtoPEM

	// ErrDecodingPEMCert indicates a PEM certificate could not be decoded
	ErrDecodingPEMCert

	// ErrDecodingPEMPrivateKey indicates a PEM private key for a certificate could not be decoded
	ErrDecodingPEMPrivateKey

	// ErrIssuingCert indicates a nonspecific failure to issue a certificate
	ErrIssuingCert

	// ErrCreatingCert indicates certificate creation failed when issuing a certificate
	ErrCreatingCert

	// ErrInvalidCA indicates an invalid certificate authority was provided when attempting to issue a certificate
	ErrInvalidCA

	// ErrRotatingCert indicates a certificate could not be rotated
	ErrRotatingCert
)

// Range 4100-4150 reserved for PubSub system
const (
	// ErrPubSubMessageFormat indicates error when parsing an object to a pubsub message
	ErrPubSubMessageFormat ErrCode = iota + 4100
)

// Range 4150-4200 reserved for MeshConfig related errors
const (
	// ErrMeshConfigInformerInitCache indicates failed to init cache sync for MeshConfig informer
	ErrMeshConfigInformerInitCache ErrCode = iota + 4150

	// ErrMeshConfigStructParsing indicates failed to cast object to MeshConfig
	ErrMeshConfigStructCasting

	// ErrMeshConfigFetchFromCache indicates failed to fetch MeshConfig from cache with specific key
	ErrMeshConfigFetchFromCache

	// ErrMeshConfigMarshaling indicates failed to marshal MeshConfig into other format like JSON
	ErrMeshConfigMarshaling
)

// Range 5000-5500 reserved for errors related to Envoy XDS control plane
const (
	// ErrMarshallingXDSResource indicates an XDS resource could not be marshalled
	ErrMarshallingXDSResource ErrCode = iota + 5000

	// ErrParsingXDSCertCN indicates the configured XDS certificate common name could not be parsed
	ErrParsingXDSCertCN

	// ErrFetchingPodFromCert indicates the proxy UUID obtained from a certificate's common name metadata was not
	// found as a osm-proxy-uuid label value for any pod
	ErrFetchingPodFromCert

	// ErrPodBelongsToMultipleServices indicates a pod in the mesh belongs to more than one service
	ErrPodBelongsToMultipleServices

	// ErrGettingProxyFromPod indicates the proxy data structure could not be obtained from the osm-proxy-uuid
	// label value on a pods
	ErrGettingProxyFromPod

	// ErrGRPCConnectionFailed indicates discovery requests cannot be received by ADS due to a GRPC connection failure
	ErrGRPCConnectionFailed

	// ErrSendingDiscoveryResponse indicates the configured discovery response could not be sent
	ErrSendingDiscoveryResponse

	// ErrGeneratingReqResource indicates the resources for the discovery response could not be generated
	ErrGeneratingReqResource

	// ErrRecordingSnapshot indicates the aggregated resources generate for a discovery response could not be created
	ErrRecordingSnapshot

	// ErrGettingServiceIdentity indicates the ServiceIdentity name encoded in the XDS certificate CN could not be
	// obtained
	ErrGettingServiceIdentity

	// ErrStartingADSServer indicates the gPRC service failed to start
	ErrStartingADSServer

	// ERRInitializingProxy indicates an instance of the Envoy proxy that connected to the XDS server could not be
	// initialized
	ErrInitializingProxy

	// ErrMismatchedServiceAccount inicates the ServiceAccount referenced in the NodeID does not match the
	// ServiceAccount specified in the proxy certificate
	ErrMismatchedServiceAccount

	// ErrGRPCStreamClosedByProxy indicates the gRPC stream was closed by the proxy
	ErrGRPCStreamClosedByProxy

	// ErrUnexpectedXDSRequest indicates that a proxy has not completed its init phase and is not ready to
	// receive updates
	ErrUnexpectedXDSRequest

	// ErrInvalidXDSTypeURI indicates the TypeURL of the discovery request is invalid
	ErrInvalidXDSTypeURI

	// ErrParsingDiscoveryReqVersion indicates the discovery request response version could not be parsed
	ErrParsingDiscoveryReqVersion

	// ErrGettingServicePorts indicates the mapping of a service's ports to their corresponding application protocol could not be obtained
	ErrGettingServicePorts

	// ErrGettingOrgDstEgressCluster indicates that an Envoy egress cluster that routes traffic to its original destination could not be configured
	ErrGettingOrgDstEgressCluster

	// ErrGettingDNSEgressCluster indicates that an Envoy egress cluster that routes traffic based on the specified Host resolved using DNS could not be configured
	ErrGettingDNSEgressCluster

	// ErrObtainingUpstreamServiceCluster indicates an Envoy cluster corresponding to an upstream service could not be configured
	ErrObtainingUpstreamServiceCluster

	// ErrFetchingServiceList indicates the services corresponding to a specified proxy could not be listed
	ErrFetchingServiceList

	// ErrGettingLocalServiceCluster indicates that an Envoy cluster for a local service behind the Envoy proxy could not be configured
	ErrGettingLocalServiceCluster

	// ErrDuplicateluster indicates Envoy clusters with the same name were found
	ErrDuplicateClusters
)

// String returns the error code as a string, ex. E1000
func (e ErrCode) String() string {
	return fmt.Sprintf("E%d", e)
}

// FromStr returns the ErrCode representation for the given error code string
// Ex. E1000 is converted to ErrInvalidCLIArgument
func FromStr(e string) (ErrCode, error) {
	errStr := strings.TrimLeft(e, "E")
	errInt, err := strconv.Atoi(errStr)
	if err != nil {
		return ErrCode(0), errors.Errorf("Error code '%s' is not a valid error code format. Should be of the form Exxxx, ex. E1000.", e)
	}
	return ErrCode(errInt), nil
}

// ErrCodeMap defines the mapping of error codes to their description.
// Note: error code description mappings must be defined in the same order
// as they appear in the error code definitions - from lowest to highest
// ranges in the order they appear within the range.
var ErrCodeMap = map[ErrCode]string{
	//
	// Range 1000-1050
	//
	ErrInvalidCLIArgument: `
An invalid command line argument was passed to the application.
`,

	ErrSettingLogLevel: `
The specified log level could not be set in the system.
`,

	ErrParsingMeshConfig: `
The 'osm-mesh-config' MeshConfig custom resource could not be parsed.
`,

	ErrFetchingControllerPod: `
The osm-controller k8s pod resource was not able to be retrieved by the system.
`,

	ErrFetchingInjectorPod: `
The osm-injector k8s pod resource was not able to be retrieved by the system.
`,

	ErrStartingReconcileManager: `
The controller-runtime manager to manage the controller used to reconcile the
sidecar injector's MutatingWebhookConfiguration resource failed to start.
`,

	//
	// Range 2000-2500
	//
	ErrDedupEgressTrafficMatches: `
An error was encountered while attempting to deduplicate traffic matching attributes
(destination port, protocol, IP address etc.) used for matching egress traffic.
The applied egress policies could be conflicting with each other, and the system
was unable to process affected egress policies.
`,

	ErrDedupEgressClusterConfigs: `
An error was encountered while attempting to deduplicate upstream clusters associated
with the egress destination.
The applied egress policies could be conflicting with each other, and the system
was unable to process affected egress policies.
`,

	ErrInvalidEgressIPRange: `
An invalid IP address range was specified in the egress policy. The IP address range
must be specified as as a CIDR notation IP address and prefix length, like "192.0.2.0/24",
as defined in RFC 4632.
The invalid IP address range was ignored by the system.
`,

	ErrInvalidEgressMatches: `
An invalid match was specified in the egress policy.
The specified match was ignored by the system while applying the egress policy.
`,

	ErrEgressSMIHTTPRouteGroupNotFound: `
The SMI HTTPRouteGroup resource specified as a match in an egress policy was not found.
Please verify that the specified SMI HTTPRouteGroup resource exists in the same namespace
as the egress policy referencing it as a match.
`,

	ErrFetchingSMIHTTPRouteGroupForTrafficTarget: `
The SMI HTTPRouteGroup resources specified as a match in an SMI TrafficTarget policy was
unable to be retrieved by the system.
The associated SMI TrafficTarget policy was ignored by the system. Please verify that the
matches specified for the Traffictarget resource exist in the same namespace as the
TrafficTarget policy referencing the match.
`,

	ErrSMIHTTPRouteGroupNoMatch: `
The SMI HTTPRouteGroup resource is invalid as it does not have any matches specified.
The SMI HTTPRouteGroup policy was ignored by the system.
`,

	ErrMultipleSMISplitPerServiceUnsupported: `
There are multiple SMI traffic split policies associated with the same apex(root)
service specified in the policies. The system does not support this scenario so
onlt the first encountered policy is processed by the system, subsequent policies
referring the same apex service are ignored.
`,

	ErrAddingRouteToOutboundTrafficPolicy: `
There was an error adding a route match to an outbound traffic policy representation
within the system.
The associated route was ignored by the system.
`,

	ErrFetchingServiceForTrafficTargetDestination: `
The system was unable to lookup the services associated with the destination specified
in the SMI TrafficTarget policy.
The associated SMI TrafficTarget policy was ignored by the system.
`,

	//
	// Range 3000-3500
	//
	ErrServiceHostnames: `
The hostnames (FQDN) used to access the k8s service could not be retrieved by the system.
Any HTTP traffic policy associated with this service was ignored by the system.
`,

	ErrNoMatchingServiceForServiceAccount: `
The system expected k8s services to be associated with a service account, but no such
service was found. A service matches a service account if the pods backing the service
belong to the service account.
`,

	//
	// Range 4000-4100
	//
	ErrFetchingCertSecret: `
The Kubernetes secret containing the certificate could not be retrieved by the system.
`,

	ErrObtainingCertFromSecret: `
The certificate specified by name could not be obtained by key from the secret's data.
`,

	ErrObtainingPrivateKeyFromSecret: `
The private key specified by name could not be obtained by key from the secret's data.
`,

	ErrObtainingCertExpirationFromSecret: `
The certificate expiration specified by name could not be obtained by key from the secret's
data.
`,

	ErrParsingCertExpiration: `
The certificate expiration obtained from the secret's data by name could not be parsed.
`,

	ErrCreatingCertSecret: `
The secret containing a certificate could not be created by the system.
`,

	ErrGeneratingPrivateKey: `
A private key failed to be generated.
`,

	ErrEncodingKeyDERtoPEM: `
The specified private key could be be could not be converted from a DER encoded key to a
PEM encoded key.
`,

	ErrCreatingCertReq: `
The certificate request fails to be created when attempting to issue a certificate.
`,

	ErrDeletingCertReq: `
The certificate request could not be deleted.
`,

	ErrCreatingRootCert: `
When creating a new certificate authority, the root certificate could not be obtained by
the system.
`,

	ErrEncodingCertDERtoPEM: `
The specified certificate could not be converted from a DER encoded certificate to a PEM
encoded certificate.
`,

	ErrDecodingPEMCert: `
The specified PEM encoded certificate could not be decoded.
`,

	ErrDecodingPEMPrivateKey: `
The specified PEM privateKey for the certificate authority's root certificate could not
be decoded.
`,

	ErrIssuingCert: `
An unspecified error occurred when issuing a certificate from the certificate manager.
`,

	ErrCreatingCert: `
An error occurred when creating a certificate to issue from the certificate manager.
`,

	ErrInvalidCA: `
The certificate authority privided when issuing a certificate was invalid.
`,

	ErrRotatingCert: `
The specified certificate could not be rotated.
`,

	//
	// Range 4100-4150
	//
	ErrPubSubMessageFormat: `
Failed parsing object into PubSub message.
`,

	//
	// Range 4150-4200
	//
	ErrMeshConfigInformerInitCache: `
Failed initial cache sync for MeshConfig informer.
`,
	ErrMeshConfigStructCasting: `
Failed to cast object to MeshConfig.
`,
	ErrMeshConfigFetchFromCache: `
Failed to fetch MeshConfig from cache with specific key.
`,
	ErrMeshConfigMarshaling: `
Failed to marshal MeshConfig into other format.
`,

	//
	// Range 5000-5500
	//
	ErrMarshallingXDSResource: `
A XDS resource could not be marshalled.
`,

	ErrParsingXDSCertCN: `
The XDS certificate common name could not be parsed. The CN should be of the form 
<proxy-UUID>.<kind>.<proxy-identity>.
`,

	ErrFetchingPodFromCert: `
The proxy UUID obtained from parsing the XDS certificate's common name did not match
the osm-proxy-uuid label value for any pod. The pod associated with the specified Envoy
proxy could not be found.
`,

	ErrPodBelongsToMultipleServices: `
A pod in the mesh belongs to more than one service. By Open Service Mesh convention
the number of services a pod can belong to is 1. This is a limitation we set in place
in order to make the mesh easy to understand and reason about. When a pod belongs to
more than one service XDS will not program the Envoy proxy, leaving it out of the mesh.
`,

	ErrGettingProxyFromPod: `
The Envoy proxy data structure created by ADS to reference an Envoy proxy sidecar from
a pod's osm-proxy-uuid label could not be configured. 
`,

	ErrGRPCConnectionFailed: `
A GRPC connection failure occurred and the ADS is no longer able to receive
DiscoveryRequests.
`,

	ErrSendingDiscoveryResponse: `
The DiscoveryResponse configured by ADS failed to send to the Envoy proxy.
`,

	ErrGeneratingReqResource: `
The resources to be included in the DiscoveryResponse could not be generated.
`,

	ErrRecordingSnapshot: `
The aggregated resources generated for a DiscoveryResponse failed to be configured as
a new snapshot in the Envoy xDS Aggregate Discovery Services cache.
`,

	ErrGettingServiceIdentity: `
The ServiceIdentity specified in the XDS certificate CN could not be obtained when
creating SDS DiscoveryRequests corresponding to all types of secrets associated with
the proxy.
`,

	ErrStartingADSServer: `
The Aggregate Discovery Server (ADS) created by the OSM controller failed to start.
`,

	ErrInitializingProxy: `
An Envoy proxy data structure representing a newly connected envoy proxy to the XDS
server could not be initialized.
`,

	ErrMismatchedServiceAccount: `
The ServiceAccount referenced in the NodeID does not match the ServiceAccount
specified in the proxy certificate. In this case, the proxy is not allowed to be a
part of the mesh.
`,

	ErrGRPCStreamClosedByProxy: `
The gRPC stream is closed by the proxy and no DiscoveryRequests can be received. 
The Stream Agreggated Resource server is terminated for the specified proxy
`,

	ErrUnexpectedXDSRequest: `
The envoy proxy has not completed the initialization phase and it is not ready
to receive broadcast updates from control plane related changes. New versions
should not be pushed if the first request has not be received.
The broadcast update is ignored for that proxy.
`,

	ErrInvalidXDSTypeURI: `
The TypeURL of the resource being requested in the DiscoveryRequest is invalid.
`,

	ErrParsingDiscoveryReqVersion: `
The version of the DiscoveryRequest could not be parsed by ADS.
`,

	ErrGettingServicePorts: `
The mapping of ports the application is exposing a service on to their corresponding
application protocol could not be obtained for a specified service.
`,

	ErrGettingOrgDstEgressCluster: `
An Envoy egress cluster which routes traffic to its original destination could not
be configured. When a Host is not specified in the cluster config, the original 
destination is used.
`,

	ErrGettingDNSEgressCluster: `
An Envoy egress cluster that routes traffic based on the specified Host resolved
using DNS could not be configured.
`,

	ErrObtainingUpstreamServiceCluster: `
An Envoy cluster that corresponds to a specified upstream service could not be
configured. 
`,

	ErrFetchingServiceList: `
The meshed services corresponding a specified Envoy proxy could not be listed.
`,

	ErrGettingLocalServiceCluster: `
An Envoy cluster for a local service behind an Envoy proxy could not be configured.
`,

	ErrDuplicateClusters: `
Multiple Envoy clusters with the same name were configured. The duplicate clusters
will not be sent to the Envoy proxy in a ClusterDiscovery response.
`,
}
