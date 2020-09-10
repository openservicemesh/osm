package catalog

import (
	"context"
	"fmt"
	"strings"

	mapset "github.com/deckarep/golang-set"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
)

// GetServiceFromEnvoyCertificate returns the single service given Envoy is a member of based on the certificate provided, which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
func (mc *MeshCatalog) GetServiceFromEnvoyCertificate(cn certificate.CommonName) (*service.MeshService, error) {
	pod, err := GetPodFromCertificate(cn, mc.kubeClient)
	if err != nil {
		return nil, err
	}

	services, err := listServicesForPod(pod, mc.kubeClient)
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

	return &service.MeshService{
		Namespace: cnMeta.Namespace,
		Name:      services[0].Name,
	}, nil
}

func (mc *MeshCatalog) GetGatewaypods(searchName string) ([]string, error) {
	kubeClient := mc.kubeClient
	podList, err := kubeClient.CoreV1().Pods("default").List(context.Background(), v12.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error listing pods in namespace %s", "default")
		return nil, fmt.Errorf("error listing pod")
	}

	searchList := make([]string, 0)
	for _, pod := range podList.Items {
		if strings.Contains(pod.Name, searchName) && pod.Status.Phase == "Running" {
			log.Info().Msgf("pod.Name=%+v, pod.status=%+v \n", pod.Name, pod.Status.Phase)
			searchList = append(searchList, pod.Name)
		}
	}
	return searchList, nil
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

	for _, svc := range services {
		nsSvc := service.MeshService{
			Namespace: svc.Namespace,
			Name:      svc.Name,
		}
		if _, shouldSkip := excludeTheseServices[nsSvc]; shouldSkip {
			continue
		}
		filteredServices = append(filteredServices, svc)
	}

	return filteredServices
}

// GetPodFromCertificate returns the Kubernetes Pod object for a given certificate.
func GetPodFromCertificate(cn certificate.CommonName, kubeClient kubernetes.Interface) (*v1.Pod, error) {
	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return nil, err
	}

	log.Trace().Msgf("Looking for pod with label %q=%q", constants.EnvoyUniqueIDLabelName, cnMeta.ProxyID)

	podList, err := kubeClient.CoreV1().Pods(cnMeta.Namespace).List(context.Background(), v12.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error listing pods in namespace %s", cnMeta.Namespace)
		return nil, err
	}

	var pods []v1.Pod
	for _, pod := range podList.Items {
		for labelKey, labelValue := range pod.Labels {
			if labelKey == constants.EnvoyUniqueIDLabelName && labelValue == cnMeta.ProxyID {
				pods = append(pods, pod)
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

func mapStringStringToSet(m map[string]string) mapset.Set {
	stringSet := mapset.NewSet()
	for k, v := range m {
		stringSet.Add(fmt.Sprintf("%s:%s", k, v))
	}
	return stringSet
}

func listServicesForPod(pod *v1.Pod, kubeClient kubernetes.Interface) ([]v1.Service, error) {
	var serviceList []v1.Service
	svcList, err := kubeClient.CoreV1().Services(pod.Namespace).List(context.Background(), v12.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error listing pods in namespace %s", pod.Namespace)
		return nil, err
	}

	podLabels := mapStringStringToSet(pod.Labels)

	for _, svc := range svcList.Items {
		serviceLabelSet := mapStringStringToSet(svc.Spec.Selector)
		if serviceLabelSet.Intersect(podLabels).Cardinality() > 0 {
			serviceList = append(serviceList, svc)
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
