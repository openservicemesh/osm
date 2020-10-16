package catalog

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/utils"
)

// GetServicesFromEnvoyCertificate returns a list of services the given Envoy is a member of based on the certificate provided, which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
func (mc *MeshCatalog) GetServicesFromEnvoyCertificate(cn certificate.CommonName) ([]service.MeshService, error) {
	var serviceList []service.MeshService
	pod, err := GetPodFromCertificate(cn, mc.kubeController)
	if err != nil {
		return nil, err
	}

	services, err := listServicesForPod(pod, mc.kubeController)
	if err != nil {
		return nil, err
	}

	// Remove services that have been split into other services.
	// Filters out services referenced in TrafficSplit.spec.service
	services = mc.filterTrafficSplitServices(services)

	if len(services) == 0 {
		log.Error().Msgf("No services found for connected proxy ID %s", cn)
		return nil, errNoServicesFoundForCertificate
	}

	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return nil, err
	}

	for _, svc := range services {
		meshService := service.MeshService{
			Namespace: cnMeta.Namespace,
			Name:      svc.Name,
		}
		serviceList = append(serviceList, meshService)
	}
	return serviceList, nil
}

// filterTrafficSplitServices takes a list of services and removes from it the ones
// that have been split via an SMI TrafficSplit.
func (mc *MeshCatalog) filterTrafficSplitServices(services []v1.Service) []v1.Service {
	excludeTheseServices := make(map[service.MeshService]interface{})
	for _, trafficSplit := range mc.meshSpec.ListTrafficSplits() {
		svc := service.MeshService{
			Namespace: trafficSplit.Namespace,
			Name:      trafficSplit.Spec.Service,
		}
		excludeTheseServices[svc] = nil
	}

	log.Debug().Msgf("Filtered out apex services (no pods can belong to these): %+v", excludeTheseServices)

	// These are the services except ones that are a root of a TrafficSplit policy
	var filteredServices []v1.Service

	for i, svc := range services {
		nsSvc := utils.K8sSvcToMeshSvc(&services[i])
		if _, shouldSkip := excludeTheseServices[nsSvc]; shouldSkip {
			continue
		}
		filteredServices = append(filteredServices, svc)
	}

	return filteredServices
}

// GetPodFromCertificate returns the Kubernetes Pod object for a given certificate.
func GetPodFromCertificate(cn certificate.CommonName, kubecontroller k8s.Controller) (*v1.Pod, error) {
	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return nil, err
	}

	log.Trace().Msgf("Looking for pod with label %q=%q", constants.EnvoyUniqueIDLabelName, cnMeta.ProxyID)
	podList := kubecontroller.ListPods()
	var pods []v1.Pod
	for _, pod := range podList {
		if pod.Namespace != cnMeta.Namespace {
			continue
		}
		for labelKey, labelValue := range pod.Labels {
			if labelKey == constants.EnvoyUniqueIDLabelName && labelValue == cnMeta.ProxyID {
				pods = append(pods, *pod)
			}
		}
	}

	if len(pods) == 0 {
		log.Error().Msgf("Did not find pod with label %s = %s in namespace %s", constants.EnvoyUniqueIDLabelName, cnMeta.ProxyID, cnMeta.Namespace)
		return nil, errDidNotFindPodForCertificate
	}

	// --- CONVENTION ---
	// By Open Service Mesh convention the number of services a pod can belong to is 1
	// This is a limitation we set in place in order to make the mesh easy to understand and reason about.
	// When a pod belongs to more than one service XDS will not program the Envoy proxy, leaving it out of the mesh.
	if len(pods) > 1 {
		log.Error().Msgf("Found more than one pod with label %s = %s in namespace %s; There should be only one!", constants.EnvoyUniqueIDLabelName, cnMeta.ProxyID, cnMeta.Namespace)
		return nil, errMoreThanOnePodForCertificate
	}

	pod := pods[0]
	log.Trace().Msgf("Found pod %s for proxyID %s", pod.Name, cnMeta.ProxyID)

	// Ensure the Namespace encoded in the certificate matches that of the Pod
	if pod.Namespace != cnMeta.Namespace {
		log.Warn().Msgf("Pod %s belongs to Namespace %s while the pod's cert was issued for Namespace %s", pod.Name, pod.Namespace, cnMeta.Namespace)
		return nil, errNamespaceDoesNotMatchCertificate
	}

	// Ensure the Name encoded in the certificate matches that of the Pod
	if pod.Spec.ServiceAccountName != cnMeta.ServiceAccount {
		// Since we search for the pod in the namespace we obtain from the certificate -- these namespaces will always matech.
		log.Warn().Msgf("Pod %s/%s belongs to Name %q while the pod's cert was issued for Name %q", pod.Namespace, pod.Name, pod.Spec.ServiceAccountName, cnMeta.ServiceAccount)
		return nil, errServiceAccountDoesNotMatchCertificate
	}

	return &pod, nil
}

// listServicesForPod lists Kubernetes services whose selectors match pod labels
func listServicesForPod(pod *v1.Pod, kubeController k8s.Controller) ([]v1.Service, error) {
	var serviceList []v1.Service
	svcList := kubeController.ListServices()

	for _, svc := range svcList {
		if svc.Namespace != pod.Namespace {
			continue
		}
		svcRawSelector := svc.Spec.Selector
		selector := labels.Set(svcRawSelector).AsSelector()
		if selector.Matches(labels.Set(pod.Labels)) {
			serviceList = append(serviceList, *svc)
		}
	}

	return serviceList, nil
}

func getCertificateCommonNameMeta(cn certificate.CommonName) (*certificateCommonNameMeta, error) {
	chunks := strings.Split(cn.String(), constants.DomainDelimiter)
	if len(chunks) < 3 {
		return nil, errInvalidCertificateCN
	}
	return &certificateCommonNameMeta{
		ProxyID:        chunks[0],
		ServiceAccount: chunks[1],
		Namespace:      chunks[2],
	}, nil
}

// NewCertCommonNameWithProxyID returns a newly generated CommonName for a certificate of the form: <ProxyID>.<serviceAccount>.<namespace>
func NewCertCommonNameWithProxyID(proxyUUID, serviceAccount, namespace string) certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{proxyUUID, serviceAccount, namespace}, constants.DomainDelimiter))
}
