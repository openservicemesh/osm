package catalog

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/utils"
)

// GetServicesFromEnvoyCertificate returns a list of services the given Envoy is a member of based
// on the certificate provided, which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
func (mc *MeshCatalog) GetServicesFromEnvoyCertificate(cn certificate.CommonName) ([]service.MeshService, error) {
	pod, err := certificate.GetPodFromCertificate(cn, mc.kubeController)
	if err != nil {
		return nil, err
	}

	services, err := listServicesForPod(pod, mc.kubeController)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, nil
	}

	// Remove services that have been split into other services.
	// Filters out services referenced in TrafficSplit.spec.service
	services = mc.filterTrafficSplitServices(services)

	meshServices := kubernetesServicesToMeshServices(services)

	servicesForPod := strings.Join(listServiceNames(meshServices), ",")
	log.Trace().Msgf("Services associated with Pod with UID=%s Name=%s/%s: %+v",
		pod.ObjectMeta.UID, pod.Namespace, pod.Name, servicesForPod)

	return meshServices, nil
}

func kubernetesServicesToMeshServices(kubernetesServices []v1.Service) (meshServices []service.MeshService) {
	for _, svc := range kubernetesServices {
		meshServices = append(meshServices, service.MeshService{
			Namespace: svc.Namespace,
			Name:      svc.Name,
		})
	}
	return meshServices
}

func listServiceNames(meshServices []service.MeshService) (serviceNames []string) {
	for _, meshService := range meshServices {
		serviceNames = append(serviceNames, fmt.Sprintf("%s/%s", meshService.Namespace, meshService.Name))
	}
	return serviceNames
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

// listServicesForPod lists Kubernetes services whose selectors match pod labels
func listServicesForPod(pod *v1.Pod, kubeController k8s.Controller) ([]v1.Service, error) {
	var serviceList []v1.Service
	svcList := kubeController.ListServices()

	for _, svc := range svcList {
		if svc.Namespace != pod.Namespace {
			continue
		}
		svcRawSelector := svc.Spec.Selector
		// service has no selectors, we do not need to match against the pod label
		if len(svcRawSelector) == 0 {
			continue
		}
		selector := labels.Set(svcRawSelector).AsSelector()
		if selector.Matches(labels.Set(pod.Labels)) {
			serviceList = append(serviceList, *svc)
		}
	}

	return serviceList, nil
}

// NewCertCommonNameWithProxyID returns a newly generated CommonName for a certificate of the form: <ProxyUUID>.<serviceAccount>.<namespace>
func NewCertCommonNameWithProxyID(proxyUUID uuid.UUID, serviceAccount, namespace string) certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{proxyUUID.String(), serviceAccount, namespace}, constants.DomainDelimiter))
}
