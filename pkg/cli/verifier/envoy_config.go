package verifier

import (
	"context"
	"fmt"
	"io"
	"strings"

	mapset "github.com/deckarep/golang-set"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/lds"
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
	if t.trafficAttr.DstPod != nil {
		fmt.Fprintf(&s, "\tdestination pod: %s\n", t.trafficAttr.DstPod)
	}
	if t.trafficAttr.DstService != nil {
		fmt.Fprintf(&s, "\tdestination service: %s\n", t.trafficAttr.DstService)
	}
	if t.trafficAttr.DstHost != "" {
		fmt.Fprintf(&s, "\tdestination host: %s\n", t.trafficAttr.DstHost)
	}
	if t.trafficAttr.DstPort != 0 {
		fmt.Fprintf(&s, "\tdestination port: %d\n", t.trafficAttr.DstPort)
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
		Context: fmt.Sprintf("Verify Envoy config on source for traffic: %s", v.configAttr),
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

	// Next, if the destination service is known, verify it has a matching filter chain
	if v.configAttr.trafficAttr.DstService != nil {
		dst := v.configAttr.trafficAttr.DstService
		svc, err := v.kubeClient.CoreV1().Services(dst.Namespace).Get(context.Background(), dst.Name, metav1.GetOptions{})
		if err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Destination service %q not found: %s", dst, err)
			return result
		}
		if err := v.findOutboundFilterChainForService(svc, outboundListener.FilterChains); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find matching outbound filter chain for service %q: %s", dst, err)
			return result
		}
	}

	result.Status = Success
	return result
}

func (v *EnvoyConfigVerifier) findOutboundFilterChainForService(svc *corev1.Service, filterChains []*xds_listener.FilterChain) error {
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
			return errors.Errorf("Endpoints not found for service %q", svcNamespacedName)
		}
		for _, sub := range endpoints.Subsets {
			for _, ip := range sub.Addresses {
				dstIPRanges.Add(ip.IP)
			}
		}
	} else {
		dstIPRanges.Add(svc.Spec.ClusterIP)
	}

	for _, port := range svc.Spec.Ports {
		meshSvc := service.MeshService{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Protocol:  v.configAttr.trafficAttr.AppProtocol,
			Port:      uint16(port.Port),
		}
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
		return errors.Errorf("filter chain match %s not found", meshSvc.OutboundTrafficMatchName())
	}

	// Verify the filter chain match
	if filterChain.FilterChainMatch.DestinationPort.GetValue() != uint32(meshSvc.Port) {
		return errors.Errorf("filter chain match not found for port %d", meshSvc.Port)
	}
	configuredIPSet := mapset.NewSet()
	for _, prefix := range filterChain.FilterChainMatch.PrefixRanges {
		configuredIPSet.Add(prefix.GetAddressPrefix())
	}
	if !dstIPRanges.Equal(configuredIPSet) {
		return errors.Errorf("filter chain match not found for IP ranges %s, found %s", dstIPRanges, configuredIPSet)
	}

	// Verify the app protocol filter is present
	filterName := getFilterForProtocol(meshSvc.Protocol)
	if filterName == "" {
		return errors.Errorf("unsupported protocol %s", meshSvc.Protocol)
	}
	filterFound := false
	for _, filter := range filterChain.Filters {
		if filter.Name == filterName {
			filterFound = true
			break
		}
	}
	if !filterFound {
		return errors.Errorf("filter %s not found", filterName)
	}

	return nil
}

func getFilterForProtocol(protocol string) string {
	switch protocol {
	case constants.ProtocolHTTP:
		return wellknown.HTTPConnectionManager

	case constants.ProtocolTCP:
		return wellknown.TCPProxy

	default:
		return ""
	}
}

func (v *EnvoyConfigVerifier) verifyDestination() Result {
	result := Result{
		Context: fmt.Sprintf("Verify Envoy config on destination for traffic: %s", v.configAttr),
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

	// Next, if the destination service is known, verify it has a matching filter chain
	if v.configAttr.trafficAttr.DstService != nil {
		dst := v.configAttr.trafficAttr.DstService
		svc, err := v.kubeClient.CoreV1().Services(dst.Namespace).Get(context.Background(), dst.Name, metav1.GetOptions{})
		if err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Destination service %q not found: %s", dst, err)
			return result
		}
		if err := v.findInboundFilterChainForService(svc, inboundListener.FilterChains); err != nil {
			result.Status = Failure
			result.Reason = fmt.Sprintf("Did not find matching inbound filter chain for service %q: %s", dst, err)
			return result
		}
	}

	result.Status = Success
	return result
}

func (v *EnvoyConfigVerifier) findInboundFilterChainForService(svc *corev1.Service, filterChains []*xds_listener.FilterChain) error {
	if svc == nil {
		return nil
	}

	for _, port := range svc.Spec.Ports {
		meshSvc := service.MeshService{
			Name:       svc.Name,
			Namespace:  svc.Namespace,
			Protocol:   v.configAttr.trafficAttr.AppProtocol,
			TargetPort: uint16(port.TargetPort.IntVal),
		}
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
		return errors.Errorf("filter chain match %s not found", meshSvc.InboundTrafficMatchName())
	}

	// Verify the filter chain match
	if filterChain.FilterChainMatch.DestinationPort.GetValue() != uint32(meshSvc.TargetPort) {
		return errors.Errorf("filter chain match not found for port %d", meshSvc.TargetPort)
	}

	// Verify the app protocol filter is present
	filterName := getFilterForProtocol(meshSvc.Protocol)
	if filterName == "" {
		return errors.Errorf("unsupported protocol %s", meshSvc.Protocol)
	}
	filterFound := false
	for _, filter := range filterChain.Filters {
		if filter.Name == filterName {
			filterFound = true
			break
		}
	}
	if !filterFound {
		return errors.Errorf("filter %s not found", filterName)
	}

	return nil
}
