package verifier

import (
	"context"
	"fmt"
	"io"
	"strings"

	mapset "github.com/deckarep/golang-set"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_secret "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/lds"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	envoySecrets "github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
)

// configAttribute describes the attributes of the traffic
type configAttribute struct {
	trafficAttr     TrafficAttribute
	srcConfigGetter ConfigGetter
	dstConfigGetter ConfigGetter
}

func (t configAttribute) String() string {
	var s strings.Builder
	fmt.Fprintf(&s, "\n")
	if t.trafficAttr.SrcPod != nil {
		fmt.Fprintf(&s, "\tsource pod: %s\n", t.trafficAttr.SrcPod)
	}
	if t.trafficAttr.DstService != nil {
		fmt.Fprintf(&s, "\tsource service: %s\n", t.trafficAttr.SrcService)
	}
	if t.trafficAttr.DstPod != nil {
		fmt.Fprintf(&s, "\tdestination pod: %s\n", t.trafficAttr.DstPod)
	}
	if t.trafficAttr.DstService != nil {
		fmt.Fprintf(&s, "\tdestination service: %s\n", t.trafficAttr.DstService)
	}
	if t.trafficAttr.ExternalHost != "" {
		fmt.Fprintf(&s, "\texternal host: %s\n", t.trafficAttr.ExternalHost)
	}
	if t.trafficAttr.ExternalPort != 0 {
		fmt.Fprintf(&s, "\texternal port: %d\n", t.trafficAttr.ExternalPort)
	}
	fmt.Fprintf(&s, "\tdestination protocol: %s\n", t.trafficAttr.AppProtocol)

	return strings.TrimSuffix(s.String(), "\n")
}

// EnvoyConfigVerifier implements the Verifier interface for Envoy configs
type EnvoyConfigVerifier struct {
	stdout     io.Writer
	stderr     io.Writer
	kubeClient kubernetes.Interface
	meshConfig *configv1alpha2.MeshConfig
	configAttr configAttribute
}

// NewEnvoyConfigVerifier returns a Verifier for Envoy config verification
func NewEnvoyConfigVerifier(stdout io.Writer, stderr io.Writer, kubeClient kubernetes.Interface,
	meshConfig *configv1alpha2.MeshConfig, configAttr configAttribute) Verifier {
	return &EnvoyConfigVerifier{
		stdout:     stdout,
		stderr:     stderr,
		kubeClient: kubeClient,
		meshConfig: meshConfig,
		configAttr: configAttr,
	}
}

// Run executes the Envoy config verifier
func (v *EnvoyConfigVerifier) Run() Result {
	result := Result{
		Context: fmt.Sprintf("Verify Envoy config for traffic: %s", v.configAttr),
	}

	if v.configAttr.trafficAttr.SrcPod != nil {
		// verify source
		res := v.verifySource()
		result.NestedResults = append(result.NestedResults, &res)
		if res.Status != Success {
			result.Status = Failure
			return result
		}
	}
	if v.configAttr.trafficAttr.DstPod != nil {
		// verify destination
		res := v.verifyDestination()
		result.NestedResults = append(result.NestedResults, &res)
		if res.Status != Success {
			result.Status = Failure
			return result
		}
	}

	result.Status = Success
	return result
}

func (v *EnvoyConfigVerifier) verifySource() Result {
	result := Result{
		Context: "Verify Envoy config on source",
	}

	config, err := v.configAttr.srcConfigGetter.Get()
	if err != nil || config == nil {
		result.Status = Unknown
		result.Reason = fmt.Sprintf("Error retrieving Envoy config for pod %q, err: %q", v.configAttr.trafficAttr.SrcPod, err)
		return result
	}

	//
	// Verify the config

	// Check for outbound listener
	listeners := config.Listeners.GetDynamicListeners()
	var outboundListener xds_listener.Listener
	for _, l := range listeners {
		if l.Name != lds.OutboundListenerName {
			continue
		}
		active := l.GetActiveState()
		if active == nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Outbound listener %q on source pod %q is not active", lds.OutboundListenerName, v.configAttr.trafficAttr.SrcPod)
			return result
		}
		//nolint: errcheck
		//#nosec G104: Errors unhandled
		active.Listener.UnmarshalTo(&outboundListener)
		break
	}

	if outboundListener.Name != lds.OutboundListenerName {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Outbound listener %q not found on source pod %q", lds.OutboundListenerName, v.configAttr.trafficAttr.SrcPod)
		return result
	}

	// Retrieve route configs
	var routeConfigs []*xds_route.RouteConfiguration
	configs := config.Routes.GetDynamicRouteConfigs()
	for _, r := range configs {
		routeConfig := &xds_route.RouteConfiguration{}
		//nolint: errcheck
		//#nosec G104: Errors unhandled
		r.GetRouteConfig().UnmarshalTo(routeConfig)
		routeConfigs = append(routeConfigs, routeConfig)
	}

	// Retrieve clusters
	var clusters []*xds_cluster.Cluster
	clustersConfigs := config.Clusters.GetDynamicActiveClusters()
	for _, c := range clustersConfigs {
		cluster := &xds_cluster.Cluster{}
		//nolint: errcheck
		//#nosec G104: Errors unhandled
		c.GetCluster().UnmarshalTo(cluster)
		clusters = append(clusters, cluster)
	}

	// Retrieve secrets
	var secrets []*xds_secret.Secret
	secretConfigs := config.Secrets.GetDynamicActiveSecrets()
	for _, s := range secretConfigs {
		secret := &xds_secret.Secret{}
		//nolint: errcheck
		//#nosec G104: Errors unhandled
		s.GetSecret().UnmarshalTo(secret)
		secrets = append(secrets, secret)
	}

	// If the destination service is known, verify it has a matching filter chain and route
	if v.configAttr.trafficAttr.DstService != nil {
		dst := v.configAttr.trafficAttr.DstService
		dstPod := v.configAttr.trafficAttr.DstPod
		svc, err := v.kubeClient.CoreV1().Services(dst.Namespace).Get(context.Background(), dst.Name, metav1.GetOptions{})
		if err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Destination service %q not found: %s", dst, err)
			return result
		}
		if err := v.findOutboundFilterChainForService(svc, outboundListener.FilterChains, dstPod.Name); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find matching outbound filter chain for service %q: %s", dst, err)
			return result
		}

		if err := v.findHTTPRouteForService(svc, routeConfigs, true, dstPod.Name); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find matching outbound route configuration for service %q: %s", dst, err)
			return result
		}

		if err := v.findClusterForService(svc, clusters, true, dstPod.Name); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find matching outbound cluster for service %q: %s", dst, err)
			return result
		}

		// Verify client TLS secret
		if err := v.findTLSSecretsOnSource(secrets); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Client TLS secret not found for pod %q: %s", v.configAttr.trafficAttr.SrcPod, err)
			return result
		}
	}

	if v.configAttr.trafficAttr.ExternalPort != 0 {
		if err := v.findEgressFilterChain(outboundListener.FilterChains); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find matching filter chain: %s", err)
			return result
		}

		if err := v.findEgressHTTPRoute(routeConfigs); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find outbound route configuration: %s", err)
			return result
		}

		if err := v.findEgressCluster(clusters); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find cluster for external port %d: %s", v.configAttr.trafficAttr.ExternalPort, err)
			return result
		}
	}

	result.Status = Success
	return result
}

func (v *EnvoyConfigVerifier) findOutboundFilterChainForService(svc *corev1.Service, filterChains []*xds_listener.FilterChain, podName string) error {
	if svc == nil {
		return nil
	}

	svcNamespacedName := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)

	// There should be a filter chain for each port on the service.
	// Each of those filter chains should match the clusterIP if set, otherwise
	// to a list of pod IP ranges backing the service
	dstIPRanges := mapset.NewSet()
	if len(svc.Spec.ClusterIP) == 0 || svc.Spec.ClusterIP == corev1.ClusterIPNone {
		endpoints, err := v.kubeClient.CoreV1().Endpoints(svc.Namespace).Get(context.Background(), svc.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Endpoints not found for service %q", svcNamespacedName)
		}
		for _, sub := range endpoints.Subsets {
			for _, ip := range sub.Addresses {
				dstIPRanges.Add(ip.IP)
			}
		}
	} else {
		dstIPRanges.Add(svc.Spec.ClusterIP)
	}

	meshServices, err := v.getDstMeshServicesForSvcPod(*svc, podName)
	if len(meshServices) == 0 || err != nil {
		return fmt.Errorf("endpoints not found for service %s/%s, err: %w", svc.Namespace, svc.Name, err)
	}

	for _, meshSvc := range meshServices {
		if err := findOutboundFilterChainForServicePort(meshSvc, dstIPRanges, filterChains); err != nil {
			return err
		}
	}

	return nil
}

func findOutboundFilterChainForServicePort(meshSvc service.MeshService, dstIPRanges mapset.Set, filterChains []*xds_listener.FilterChain) error {
	var filterChain *xds_listener.FilterChain
	for _, fc := range filterChains {
		if fc.Name == meshSvc.OutboundTrafficMatchName() {
			filterChain = fc
			break
		}
	}

	if filterChain == nil {
		return fmt.Errorf("filter chain match %s not found", meshSvc.OutboundTrafficMatchName())
	}

	// Verify the filter chain match
	if filterChain.FilterChainMatch.DestinationPort.GetValue() != uint32(meshSvc.Port) {
		return fmt.Errorf("filter chain match not found for port %d", meshSvc.Port)
	}
	configuredIPSet := mapset.NewSet()
	for _, prefix := range filterChain.FilterChainMatch.PrefixRanges {
		configuredIPSet.Add(prefix.GetAddressPrefix())
	}
	if !dstIPRanges.Equal(configuredIPSet) {
		return fmt.Errorf("filter chain match not found for IP ranges %s, found %s", dstIPRanges, configuredIPSet)
	}

	// Verify the app protocol filter is present
	filterName := getFilterForProtocol(meshSvc.Protocol)
	if filterName == "" {
		return fmt.Errorf("unsupported protocol %s", meshSvc.Protocol)
	}
	filterFound := false
	for _, filter := range filterChain.Filters {
		if filter.Name == filterName {
			filterFound = true
			break
		}
	}
	if !filterFound {
		return fmt.Errorf("filter %s not found", filterName)
	}

	return nil
}

func getFilterForProtocol(protocol string) string {
	switch protocol {
	case constants.ProtocolHTTP:
		return envoy.HTTPConnectionManagerFilterName

	case constants.ProtocolTCP, constants.ProtocolHTTPS:
		return envoy.TCPProxyFilterName

	default:
		return ""
	}
}

func (v *EnvoyConfigVerifier) verifyDestination() Result {
	result := Result{
		Context: "Verify Envoy config on destination",
	}

	config, err := v.configAttr.dstConfigGetter.Get()
	if err != nil || config == nil {
		result.Status = Unknown
		result.Reason = fmt.Sprintf("Error retrieving Envoy config for pod %q, err: %q", v.configAttr.trafficAttr.SrcPod, err)
		return result
	}

	//
	// Verify the config

	// Check for inbound listener
	listeners := config.Listeners.GetDynamicListeners()
	var inboundListener xds_listener.Listener
	for _, l := range listeners {
		if l.Name != lds.InboundListenerName {
			continue
		}
		active := l.GetActiveState()
		if active == nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Inbound listener %q on destination pod %q is not active", lds.InboundListenerName, v.configAttr.trafficAttr.DstPod)
			return result
		}
		//nolint: errcheck
		//#nosec G104: Errors unhandled
		active.Listener.UnmarshalTo(&inboundListener)
		break
	}

	if inboundListener.Name != lds.InboundListenerName {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Inbound listener %q not found on destination pod %q", lds.InboundListenerName, v.configAttr.trafficAttr.DstPod)
		return result
	}

	// Retrieve route configs
	var routeConfigs []*xds_route.RouteConfiguration
	configs := config.Routes.GetDynamicRouteConfigs()
	for _, r := range configs {
		routeConfig := &xds_route.RouteConfiguration{}
		//nolint: errcheck
		//#nosec G104: Errors unhandled
		r.GetRouteConfig().UnmarshalTo(routeConfig)
		routeConfigs = append(routeConfigs, routeConfig)
	}

	// Retrieve clusters
	var clusters []*xds_cluster.Cluster
	clustersConfigs := config.Clusters.GetDynamicActiveClusters()
	for _, c := range clustersConfigs {
		cluster := &xds_cluster.Cluster{}
		//nolint: errcheck
		//#nosec G104: Errors unhandled
		c.GetCluster().UnmarshalTo(cluster)
		clusters = append(clusters, cluster)
	}

	// Retrieve secrets
	var secrets []*xds_secret.Secret
	secretConfigs := config.Secrets.GetDynamicActiveSecrets()
	for _, s := range secretConfigs {
		secret := &xds_secret.Secret{}
		//nolint: errcheck
		//#nosec G104: Errors unhandled
		s.GetSecret().UnmarshalTo(secret)
		secrets = append(secrets, secret)
	}

	// Next, if the destination service is known, verify it has a matching filter chain
	if v.configAttr.trafficAttr.DstService != nil {
		dst := v.configAttr.trafficAttr.DstService
		dstPod := v.configAttr.trafficAttr.DstPod
		svc, err := v.kubeClient.CoreV1().Services(dst.Namespace).Get(context.Background(), dst.Name, metav1.GetOptions{})
		if err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Destination service %q not found: %s", dst, err)
			return result
		}
		if err := v.findInboundFilterChainForService(svc, inboundListener.FilterChains, dstPod.Name); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find matching inbound filter chain for service %q: %s", dst, err)
			return result
		}
		if err := v.findHTTPRouteForService(svc, routeConfigs, false, dstPod.Name); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find matching inbound route configuration for service %q: %s", dst, err)
			return result
		}
		if err := v.findClusterForService(svc, clusters, false, dstPod.Name); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find matching inbound cluster for service %q: %s", dst, err)
			return result
		}

		// Verify server TLS secret
		if err := v.findTLSSecretsOnDestination(secrets); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Server TLS secret not found for pod %q: %s", v.configAttr.trafficAttr.DstPod, err)
			return result
		}

		// Next, if ingress is enabled, check for ingress filters
		if v.configAttr.trafficAttr.IsIngress {
			if err := v.findIngressFilterChainForService(svc, inboundListener.FilterChains); err != nil {
				result.Status = Failure
				result.Reason = fmt.Sprintf("Did not find matching inbound filter chain for ingress %q: %s", dst, err)
				return result
			}
		}
	}

	result.Status = Success
	return result
}

func (v *EnvoyConfigVerifier) getDstMeshServicesForSvcPod(svc corev1.Service, podName string) ([]service.MeshService, error) {
	endpoints, err := v.kubeClient.CoreV1().Endpoints(svc.Namespace).Get(context.Background(), svc.Name, metav1.GetOptions{})
	if err != nil || endpoints == nil {
		return nil, err
	}

	var meshServices []service.MeshService
	for _, portSpec := range svc.Spec.Ports {
		meshSvc := service.MeshService{
			Namespace: svc.Namespace,
			Name:      svc.Name,
			Port:      uint16(portSpec.Port),
			Protocol:  pointer.StringDeref(portSpec.AppProtocol, constants.ProtocolHTTP),
		}

		// The endpoints for the kubernetes service carry information that allows
		// us to retrieve the TargetPort for the MeshService.
		meshSvc.TargetPort = k8s.GetTargetPortFromEndpoints(portSpec.Name, *endpoints)

		if !k8s.IsHeadlessService(svc) {
			meshServices = append(meshServices, meshSvc)
			continue
		}

		for _, subset := range endpoints.Subsets {
			for _, address := range subset.Addresses {
				if address.Hostname == "" {
					continue
				}
				mSvc := service.MeshService{
					Namespace:  svc.Namespace,
					Name:       fmt.Sprintf("%s.%s", address.Hostname, svc.Name),
					Port:       meshSvc.Port,
					TargetPort: meshSvc.TargetPort,
					Protocol:   meshSvc.Protocol,
				}
				meshServices = append(meshServices, mSvc)
			}
		}
	}

	return meshServices, nil
}

func (v *EnvoyConfigVerifier) findIngressFilterChainForService(svc *corev1.Service, filterChains []*xds_listener.FilterChain) error {
	if svc == nil {
		return nil
	}
	src := v.configAttr.trafficAttr.SrcService
	// grab endpoints for ingress service
	endpoints, err := v.kubeClient.CoreV1().Endpoints(src.Namespace).Get(context.Background(), src.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("ingress source service %q not found: %w", src, err)
	}

	sourceIPs := map[string]bool{}
	fakeMs := service.MeshService{
		Name:       svc.GetName(),
		Namespace:  svc.GetNamespace(),
		TargetPort: v.configAttr.trafficAttr.DstPort,
		Protocol:   v.configAttr.trafficAttr.AppProtocol,
	}

	filterChainName := fakeMs.IngressTrafficMatchName()
	var chain *xds_listener.FilterChain
	for _, c := range filterChains {
		if c.Name == filterChainName {
			chain = c
			break
		}
	}

	if chain == nil {
		return fmt.Errorf("ingress filter chain %s not found", filterChainName)
	}

	for _, ip := range chain.FilterChainMatch.SourcePrefixRanges {
		if ip.PrefixLen.GetValue() == 32 {
			sourceIPs[ip.AddressPrefix] = false
		}
	}

	for _, sub := range endpoints.Subsets {
		for _, ip := range sub.Addresses {
			matched, ok := sourceIPs[ip.IP]
			// Filter chain is missing a service endpoint
			if !ok {
				return fmt.Errorf("service endpoint %s was not found in the ingress inbound filter chain", ip.IP)
			}
			if ok && !matched {
				// ingress endpoint found in filter chain match
				sourceIPs[ip.IP] = true
			}
		}
	}

	return nil
}

func (v *EnvoyConfigVerifier) findInboundFilterChainForService(svc *corev1.Service, filterChains []*xds_listener.FilterChain, podName string) error {
	if svc == nil {
		return nil
	}

	meshServices, err := v.getDstMeshServicesForSvcPod(*svc, podName)
	if len(meshServices) == 0 || err != nil {
		return fmt.Errorf("endpoints not found for service %s/%s, err: %w", svc.Namespace, svc.Name, err)
	}

	for _, meshSvc := range meshServices {
		if err := findInboundFilterChainForServicePort(meshSvc, filterChains); err != nil {
			return err
		}
	}

	return nil
}

func findInboundFilterChainForServicePort(meshSvc service.MeshService, filterChains []*xds_listener.FilterChain) error {
	var filterChain *xds_listener.FilterChain
	for _, fc := range filterChains {
		if fc.Name == meshSvc.InboundTrafficMatchName() {
			filterChain = fc
			break
		}
	}

	if filterChain == nil {
		return fmt.Errorf("filter chain match %s not found", meshSvc.InboundTrafficMatchName())
	}

	// Verify the filter chain match
	if filterChain.FilterChainMatch.DestinationPort.GetValue() != uint32(meshSvc.TargetPort) {
		return fmt.Errorf("filter chain match not found for port %d", meshSvc.TargetPort)
	}

	// Verify the app protocol filter is present
	filterName := getFilterForProtocol(meshSvc.Protocol)
	if filterName == "" {
		return fmt.Errorf("unsupported protocol %s", meshSvc.Protocol)
	}
	filterFound := false
	for _, filter := range filterChain.Filters {
		if filter.Name == filterName {
			filterFound = true
			break
		}
	}
	if !filterFound {
		return fmt.Errorf("filter %s not found", filterName)
	}

	return nil
}

func (v *EnvoyConfigVerifier) findHTTPRouteForService(svc *corev1.Service, routeConfigs []*xds_route.RouteConfiguration, isOutbound bool, podName string) error {
	if svc == nil {
		return nil
	}

	meshServices, err := v.getDstMeshServicesForSvcPod(*svc, podName)
	if len(meshServices) == 0 || err != nil {
		return fmt.Errorf("endpoints not found for service %s/%s, err: %s", svc.Namespace, svc.Name, err)
	}

	for _, meshSvc := range meshServices {
		if meshSvc.Protocol != constants.ProtocolHTTP {
			continue
		}

		var desiredConfigName string
		if isOutbound {
			desiredConfigName = route.GetOutboundMeshRouteConfigNameForPort(int(meshSvc.Port))
		} else {
			desiredConfigName = route.GetInboundMeshRouteConfigNameForPort(int(meshSvc.TargetPort))
		}

		if err := findHTTPRouteConfig(routeConfigs, desiredConfigName, meshSvc.FQDN()); err != nil {
			return err
		}
	}

	return nil
}

func (v *EnvoyConfigVerifier) findClusterForService(svc *corev1.Service, clusters []*xds_cluster.Cluster, isOutbound bool, podName string) error {
	if svc == nil {
		return nil
	}

	meshServices, err := v.getDstMeshServicesForSvcPod(*svc, podName)
	if len(meshServices) == 0 || err != nil {
		return fmt.Errorf("endpoints not found for service %s/%s, err: %w", svc.Namespace, svc.Name, err)
	}

	for _, meshSvc := range meshServices {
		var desiredClusterName string
		if isOutbound {
			desiredClusterName = meshSvc.EnvoyClusterName()
		} else {
			desiredClusterName = meshSvc.EnvoyLocalClusterName()
		}

		if err := findCluster(clusters, desiredClusterName); err != nil {
			return err
		}
	}

	return nil
}

func findCluster(clusters []*xds_cluster.Cluster, name string) error {
	for _, c := range clusters {
		if c.Name == name {
			return nil
		}
	}

	return fmt.Errorf("cluster %s not found", name)
}

func findHTTPRouteConfig(routeConfigs []*xds_route.RouteConfiguration, desireConfigName string, desiredDomain string) error {
	var config *xds_route.RouteConfiguration

	for _, c := range routeConfigs {
		if c.Name == desireConfigName {
			config = c
			break
		}
	}

	if config == nil {
		return fmt.Errorf("route configuration %s not found", desireConfigName)
	}

	// Look for the FQDN in the virtual hosts
	var virtualHost *xds_route.VirtualHost
	for _, vh := range config.VirtualHosts {
		for _, domain := range vh.Domains {
			if domain == desiredDomain {
				virtualHost = vh
				break
			}
		}
	}

	if virtualHost == nil {
		return fmt.Errorf("virtual host for domain %s not found", desiredDomain)
	}

	return nil
}

func (v *EnvoyConfigVerifier) findTLSSecretsOnSource(secrets []*xds_secret.Secret) error {
	// Look for 2 secrets:
	// 1. Client TLS secret (based on client's ServiceAccount)
	// 2. Upstream peer validation secret (based on upstream service)
	srcPod := v.configAttr.trafficAttr.SrcPod
	pod, err := v.kubeClient.CoreV1().Pods(srcPod.Namespace).Get(context.Background(), srcPod.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("pod %s not found", srcPod)
	}
	downstreamIdentity := identity.K8sServiceAccount{Namespace: pod.Namespace, Name: pod.Spec.ServiceAccountName}.ToServiceIdentity()
	downstreamSecretName := envoySecrets.SDSCert{
		Name:     envoySecrets.GetSecretNameForIdentity(downstreamIdentity),
		CertType: envoySecrets.ServiceCertType,
	}.String()
	upstreamPeerValidationSecretName := envoySecrets.SDSCert{
		Name:     v.configAttr.trafficAttr.DstService.String(),
		CertType: envoySecrets.RootCertTypeForMTLSOutbound,
	}.String()

	expectedSecrets := mapset.NewSetWith(downstreamSecretName, upstreamPeerValidationSecretName)
	actualSecrets := mapset.NewSet()
	for _, secret := range secrets {
		actualSecrets.Add(secret.Name)
	}
	if !expectedSecrets.IsSubset(actualSecrets) {
		diff := expectedSecrets.Difference(actualSecrets)
		return fmt.Errorf("expected secrets %s not found", diff.String())
	}

	return nil
}

func (v *EnvoyConfigVerifier) findTLSSecretsOnDestination(secrets []*xds_secret.Secret) error {
	// Look for 2 secrets:
	// 1. Server TLS secret (based on upstream ServiceAccount)
	// 2. Downstream peer validation secret (based on upstream ServiceAccount)
	dstPod := v.configAttr.trafficAttr.DstPod
	pod, err := v.kubeClient.CoreV1().Pods(dstPod.Namespace).Get(context.Background(), dstPod.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("pod %s not found", dstPod)
	}
	upstreamIdentity := identity.K8sServiceAccount{Namespace: pod.Namespace, Name: pod.Spec.ServiceAccountName}.ToServiceIdentity()
	upstreamSecretName := envoySecrets.SDSCert{
		Name:     envoySecrets.GetSecretNameForIdentity(upstreamIdentity),
		CertType: envoySecrets.ServiceCertType,
	}.String()
	downstreamPeerValidationSecretName := envoySecrets.SDSCert{
		Name:     envoySecrets.GetSecretNameForIdentity(upstreamIdentity),
		CertType: envoySecrets.RootCertTypeForMTLSInbound,
	}.String()

	expectedSecrets := mapset.NewSetWith(upstreamSecretName, downstreamPeerValidationSecretName)
	actualSecrets := mapset.NewSet()
	for _, secret := range secrets {
		actualSecrets.Add(secret.Name)
	}
	if !expectedSecrets.IsSubset(actualSecrets) {
		diff := expectedSecrets.Difference(actualSecrets)
		return fmt.Errorf("expected secrets %s not found", diff.String())
	}

	return nil
}

func (v *EnvoyConfigVerifier) findEgressFilterChain(filterChains []*xds_listener.FilterChain) error {
	port := int(v.configAttr.trafficAttr.ExternalPort)
	protocol := v.configAttr.trafficAttr.AppProtocol
	matchName := trafficpolicy.GetEgressTrafficMatchName(port, protocol)

	var filterChain *xds_listener.FilterChain
	for _, fc := range filterChains {
		if fc.Name == matchName {
			filterChain = fc
			break
		}
	}
	if filterChain == nil {
		return fmt.Errorf("filter chain not found for for port=%d, protocol=%s", port, protocol)
	}

	// Verify the app protocol filter is present
	filterName := getFilterForProtocol(protocol)
	if filterName == "" {
		return fmt.Errorf("unsupported protocol %s", protocol)
	}
	filterFound := false
	for _, filter := range filterChain.Filters {
		if filter.Name == filterName {
			filterFound = true
			break
		}
	}
	if !filterFound {
		return fmt.Errorf("filter %s not found", filterName)
	}

	return nil
}

func (v *EnvoyConfigVerifier) findEgressHTTPRoute(routeConfigs []*xds_route.RouteConfiguration) error {
	protocol := v.configAttr.trafficAttr.AppProtocol
	if protocol != constants.ProtocolHTTP {
		return nil
	}

	port := int(v.configAttr.trafficAttr.ExternalPort)
	desiredRouteConfigName := route.GetEgressRouteConfigNameForPort(port)

	var config *xds_route.RouteConfiguration
	for _, c := range routeConfigs {
		if c.Name == desiredRouteConfigName {
			config = c
			break
		}
	}
	if config == nil {
		return fmt.Errorf("route configuration %s not found", desiredRouteConfigName)
	}

	dstHost := v.configAttr.trafficAttr.ExternalHost
	if dstHost == "" {
		return nil
	}

	var virtualHost *xds_route.VirtualHost
	for _, vh := range config.VirtualHosts {
		for _, domain := range vh.Domains {
			if domain == dstHost {
				virtualHost = vh
				break
			}
		}
	}

	if virtualHost == nil {
		return fmt.Errorf("virtual host for domain %s not found", dstHost)
	}

	return nil
}

func (v *EnvoyConfigVerifier) findEgressCluster(clusters []*xds_cluster.Cluster) error {
	protocol := v.configAttr.trafficAttr.AppProtocol
	port := v.configAttr.trafficAttr.ExternalPort
	host := v.configAttr.trafficAttr.ExternalHost

	var clusterName string
	switch protocol {
	case constants.ProtocolHTTP:
		clusterName = fmt.Sprintf("%s:%d", host, port)

	default:
		clusterName = fmt.Sprintf("%d", port)
	}

	for _, c := range clusters {
		if c.Name == clusterName {
			return nil
		}
	}

	return fmt.Errorf("cluster %s not found", clusterName)
}
