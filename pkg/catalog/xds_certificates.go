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
	pod, err := GetPodFromCertificate(cn, mc.kubeController)
	if err != nil {
		return nil, err
	}

	services, err := listServicesForPod(pod, mc.kubeController)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return makeSyntheticServiceForPod(pod, cn), nil
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

func makeSyntheticServiceForPod(pod *v1.Pod, proxyCommonName certificate.CommonName) []service.MeshService {
	svcAccount := service.K8sServiceAccount{
		Namespace: pod.Namespace,
		Name:      pod.Spec.ServiceAccountName,
	}
	syntheticService := svcAccount.GetSyntheticService()
	log.Debug().Msgf("Creating synthetic service %s since no actual services found for Envoy with xDS CN %s",
		syntheticService, proxyCommonName)
	return []service.MeshService{syntheticService}
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

	log.Trace().Msgf("Looking for pod with label %q=%q", constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID)
	podList := kubecontroller.ListPods()
	var pods []v1.Pod
	for _, pod := range podList {
		if pod.Namespace != cnMeta.Namespace {
			continue
		}
		for labelKey, labelValue := range pod.Labels {
			if labelKey == constants.EnvoyUniqueIDLabelName && labelValue == cnMeta.ProxyUUID.String() {
				pods = append(pods, *pod)
			}
		}
	}

	if len(pods) == 0 {
		log.Error().Msgf("Did not find Pod with label %s = %s in namespace %s",
			constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID, cnMeta.Namespace)
		return nil, errDidNotFindPodForCertificate
	}

	// --- CONVENTION ---
	// By Open Service Mesh convention the number of services a pod can belong to is 1
	// This is a limitation we set in place in order to make the mesh easy to understand and reason about.
	// When a pod belongs to more than one service XDS will not program the Envoy proxy, leaving it out of the mesh.
	if len(pods) > 1 {
		log.Error().Msgf("Found more than one pod with label %s = %s in namespace %s. There can be only one!",
			constants.EnvoyUniqueIDLabelName, cnMeta.ProxyUUID, cnMeta.Namespace)
		return nil, errMoreThanOnePodForCertificate
	}

	pod := pods[0]
	log.Trace().Msgf("Found Pod with UID=%s for proxyID %s", pod.ObjectMeta.UID, cnMeta.ProxyUUID)

	// Ensure the Namespace encoded in the certificate matches that of the Pod
	if pod.Namespace != cnMeta.Namespace {
		log.Warn().Msgf("Pod with UID=%s belongs to Namespace %s. The pod's xDS certificate was issued for Namespace %s",
			pod.ObjectMeta.UID, pod.Namespace, cnMeta.Namespace)
		return nil, errNamespaceDoesNotMatchCertificate
	}

	// Ensure the Name encoded in the certificate matches that of the Pod
	if pod.Spec.ServiceAccountName != cnMeta.ServiceAccount {
		// Since we search for the pod in the namespace we obtain from the certificate -- these namespaces will always match.
		log.Warn().Msgf("Pod with UID=%s belongs to ServiceAccount=%s. The pod's xDS certificate was issued for ServiceAccount=%s",
			pod.ObjectMeta.UID, pod.Spec.ServiceAccountName, cnMeta.ServiceAccount)
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
	proxyUUID, err := uuid.Parse(chunks[0])
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing %s into uuid.UUID", chunks[0])
		return nil, err
	}

	return &certificateCommonNameMeta{
		ProxyUUID:      proxyUUID,
		ServiceAccount: chunks[1],
		Namespace:      chunks[2],
	}, nil
}

// NewCertCommonNameWithProxyID returns a newly generated CommonName for a certificate of the form: <ProxyUUID>.<serviceAccount>.<namespace>
func NewCertCommonNameWithProxyID(proxyUUID uuid.UUID, serviceAccount, namespace string) certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{proxyUUID.String(), serviceAccount, namespace}, constants.DomainDelimiter))
}

// GetServiceAccountFromProxyCertificate returns the ServiceAccount information encoded in the certificate CN
func GetServiceAccountFromProxyCertificate(cn certificate.CommonName) (service.K8sServiceAccount, error) {
	var svcAccount service.K8sServiceAccount
	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return svcAccount, err
	}

	svcAccount.Name = cnMeta.ServiceAccount
	svcAccount.Namespace = cnMeta.Namespace

	return svcAccount, nil
}
