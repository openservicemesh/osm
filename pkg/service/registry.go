package service

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var log = logger.New("envoy/lds")

// GetServicesFromEnvoyCertificate returns a list of services the given Envoy is a member of based
// on the certificate provided, which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
func GetServicesFromEnvoyCertificate(cn certificate.CommonName, kubeController kubernetes.Controller) ([]MeshService, error) {
	pod, err := GetPodFromCertificate(cn, kubeController)
	if err != nil {
		return nil, err
	}

	services, err := listServicesForPod(pod, kubeController)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, nil
	}

	meshServices := kubernetesServicesToMeshServices(services)

	servicesForPod := strings.Join(listServiceNames(meshServices), ",")
	log.Trace().Msgf("Services associated with Pod with UID=%s Name=%s/%s: %+v",
		pod.ObjectMeta.UID, pod.Namespace, pod.Name, servicesForPod)

	return meshServices, nil
}

func listServiceNames(meshServices []MeshService) (serviceNames []string) {
	for _, meshService := range meshServices {
		meshNamespace := meshService.Namespace
		meshName := meshService.Name
		serviceNames = append(serviceNames, fmt.Sprintf("%s/%s", meshNamespace, meshName))
	}
	return serviceNames
}

func kubernetesServicesToMeshServices(kubernetesServices []v1.Service) (meshServices []MeshService) {
	for _, svc := range kubernetesServices {
		meshServices = append(meshServices, MeshService{
			Namespace: svc.Namespace,
			Name:      svc.Name,
		})
	}
	return meshServices
}

// listServicesForPod lists Kubernetes services whose selectors match pod labels
func listServicesForPod(pod *v1.Pod, kubeController kubernetes.Controller) ([]v1.Service, error) {
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
		if proxyUID, labelFound := pod.Labels[constants.EnvoyUniqueIDLabelName]; labelFound && proxyUID == cnMeta.ProxyUUID.String() {
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
	if pod.Spec.ServiceAccountName != cnMeta.ServiceAccount {
		// Since we search for the pod in the namespace we obtain from the certificate -- these namespaces will always match.
		log.Warn().Msgf("Pod with UID=%s belongs to ServiceAccount=%s. The pod's xDS certificate was issued for ServiceAccount=%s",
			pod.ObjectMeta.UID, pod.Spec.ServiceAccountName, cnMeta.ServiceAccount)
		return nil, ErrServiceAccountDoesNotMatchCertificate
	}

	return &pod, nil
}

func getCertificateCommonNameMeta(cn certificate.CommonName) (*certificateCommonNameMeta, error) {
	chunks := strings.Split(cn.String(), constants.DomainDelimiter)
	if len(chunks) < 3 {
		return nil, ErrInvalidCertificateCN
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

// certificateCommonNameMeta is the type that stores the metadata present in the CommonName field in a proxy's certificate
type certificateCommonNameMeta struct {
	ProxyUUID      uuid.UUID
	ServiceAccount string
	Namespace      string
}

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
