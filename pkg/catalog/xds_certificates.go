package catalog

import (
	"context"
	"fmt"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/open-service-mesh/osm/pkg/constants"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

const (
	domainDelimiter = "."
)

// GetServiceFromEnvoyCertificate returns the single service given Envoy is a member of based on the certificate provided, which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
func (mc *MeshCatalog) GetServiceFromEnvoyCertificate(cn certificate.CommonName) (*endpoint.NamespacedService, error) {
	service, err := getServiceFromCertificate(cn, mc.kubeClient)
	if err != nil {
		return nil, err
	}

	return service, nil
}

func getServiceFromCertificate(cn certificate.CommonName, kubeClient kubernetes.Interface) (*endpoint.NamespacedService, error) {
	pod, err := getPodFromCertificate(cn, kubeClient)
	if err != nil {
		return nil, err
	}

	services, err := listServicesForPod(pod, kubeClient)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		log.Error().Msgf("No services found for connected proxy ID %s", cn)
		return nil, errNoServicesFoundForCertificate
	}

	cnMeta, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return nil, err
	}

	return &endpoint.NamespacedService{
		Namespace: cnMeta.Namespace,
		Service:   services[0].Name,
	}, nil
}

func getPodFromCertificate(cn certificate.CommonName, kubeClient kubernetes.Interface) (*v1.Pod, error) {
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

	var pods []*v1.Pod
	for _, pod := range podList.Items {
		for labelKey, labelValue := range pod.Labels {
			if labelKey == constants.EnvoyUniqueIDLabelName && labelValue == cnMeta.ProxyID {
				pods = append(pods, &pod)
			}
		}
	}

	if len(pods) == 0 {
		log.Error().Msgf("Did not find pod with label %s = %s in namespace %s", constants.EnvoyUniqueIDLabelName, cnMeta.ProxyID, cnMeta.Namespace)
		return nil, errDidNotFindPodForCertificate
	}

	if len(pods) > 1 {
		log.Error().Msgf("Found more than one pod with label %s = %s in namespace %s; There should be only one!", constants.EnvoyUniqueIDLabelName, cnMeta.ProxyID, cnMeta.Namespace)
		return nil, errMoreThanOnePodForCertificate
	}

	pod := pods[0]

	// Ensure the Namespace encoded in the certificate matches that of the Pod
	if pod.Namespace != cnMeta.Namespace {
		log.Warn().Msgf("Pod %s belongs to Namespace %q while the pod's cert was issued for Namespace %q", pod.Name, pod.Namespace, cnMeta.Namespace)
		return nil, errNamespaceDoesNotMatchCertificate
	}

	// Ensure the ServiceAccount encoded in the certificate matches that of the Pod
	if pod.Spec.ServiceAccountName != cnMeta.ServiceAccount {
		// Since we search for the pod in the namespace we obtain from the certificate -- these namespaces will always matech.
		log.Warn().Msgf("Pod %s/%s belongs to ServiceAccount %q while the pod's cert was issued for ServiceAccount %q", pod.Namespace, pod.Name, pod.Spec.ServiceAccountName, cnMeta.ServiceAccount)
		return nil, errServiceAccountDoesNotMatchCertificate
	}

	return pod, nil
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

	for _, service := range svcList.Items {
		serviceLabelSet := mapStringStringToSet(service.Spec.Selector)
		if serviceLabelSet.Intersect(podLabels).Cardinality() > 0 {
			serviceList = append(serviceList, service)
		}
	}

	return serviceList, nil
}

func getCertificateCommonNameMeta(cn certificate.CommonName) (*certificateCommonNameMeta, error) {
	chunks := strings.Split(cn.String(), domainDelimiter)
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
	return certificate.CommonName(strings.Join([]string{proxyUUID, serviceAccount, namespace}, domainDelimiter))
}
