package registry

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
)

// ProxyServiceMapper knows how to map Envoy instances to services.
type ProxyServiceMapper interface {
	ListProxyServices(*envoy.Proxy) ([]service.MeshService, error)
}

// ExplicitProxyServiceMapper is a custom ProxyServiceMapper implementation.
type ExplicitProxyServiceMapper func(*envoy.Proxy) ([]service.MeshService, error)

// ListProxyServices executes the given mapping.
func (e ExplicitProxyServiceMapper) ListProxyServices(p *envoy.Proxy) ([]service.MeshService, error) {
	return e(p)
}

// KubeProxyServiceMapper maps an Envoy instance to services in a Kubernetes cluster.
type KubeProxyServiceMapper struct {
	KubeController k8s.Controller
}

// ListProxyServices maps an Envoy instance to a number of Kubernetes services.
func (k *KubeProxyServiceMapper) ListProxyServices(p *envoy.Proxy) ([]service.MeshService, error) {
	cn := p.GetCertificateCommonName()

	pod, err := envoy.GetPodFromCertificate(cn, k.KubeController)
	if err != nil {
		return nil, err
	}

	meshServices := listServicesForPod(pod, k.KubeController)

	servicesForPod := strings.Join(listServiceNames(meshServices), ",")
	log.Trace().Msgf("Services associated with Pod with UID=%s Name=%s/%s: %+v",
		pod.ObjectMeta.UID, pod.Namespace, pod.Name, servicesForPod)

	return meshServices, nil
}

func kubernetesServicesToMeshServices(kubeController k8s.Controller, kubernetesServices []v1.Service) (meshServices []service.MeshService) {
	for _, svc := range kubernetesServices {
		meshServices = append(meshServices, k8s.ServiceToMeshServices(svc, func(meshSvc service.MeshService) (*v1.Endpoints, error) {
			return kubeController.GetEndpoints(meshSvc)
		})...)
	}
	return meshServices
}

func listServiceNames(meshServices []service.MeshService) (serviceNames []string) {
	for _, meshService := range meshServices {
		serviceNames = append(serviceNames, fmt.Sprintf("%s/%s", meshService.Namespace, meshService.Name))
	}
	return serviceNames
}

// listServicesForPod lists Kubernetes services whose selectors match pod labels
func listServicesForPod(pod *v1.Pod, kubeController k8s.Controller) []service.MeshService {
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

	if len(serviceList) == 0 {
		return nil
	}

	meshServices := kubernetesServicesToMeshServices(kubeController, serviceList)
	// filter down meshServices to eliminate subdomains not from this pod (keeping empty subdomains)
	meshServices = service.FilterMeshServicesBySubdomain(meshServices, pod.GetName(), true)

	return meshServices
}

func listPodsForService(service *v1.Service, kubeController k8s.Controller) []v1.Pod {
	svcRawSelector := service.Spec.Selector
	// service has no selectors, we do not need to match against the pod label
	if len(svcRawSelector) == 0 {
		return nil
	}
	selector := labels.Set(svcRawSelector).AsSelector()

	allPods := kubeController.ListPods()

	var matchedPods []v1.Pod
	for _, pod := range allPods {
		if service.Namespace != pod.Namespace {
			continue
		}
		if selector.Matches(labels.Set(pod.Labels)) {
			matchedPods = append(matchedPods, *pod)
		}
	}

	return matchedPods
}

func getCertCommonNameForPod(pod v1.Pod) (certificate.CommonName, error) {
	proxyUIDStr, exists := pod.Labels[constants.EnvoyUniqueIDLabelName]
	if !exists {
		return "", errors.Errorf("no %s label", constants.EnvoyUniqueIDLabelName)
	}
	proxyUID, err := uuid.Parse(proxyUIDStr)
	if err != nil {
		return "", errors.Wrapf(err, "invalid UID value for %s label", constants.EnvoyUniqueIDLabelName)
	}
	cn := envoy.NewXDSCertCommonName(proxyUID, envoy.KindSidecar, pod.Spec.ServiceAccountName, pod.Namespace)
	return cn, nil
}
