package kubernetes

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

// GetServicesForProxy returns a list of services the given Envoy is a member of based
// on its certificate, which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
func (c Client) GetServicesForProxy(p *envoy.Proxy) ([]service.MeshService, error) {
	cn := p.GetCertificateCommonName()

	pod, err := c.GetPodFromCertificate(cn)
	if err != nil {
		return nil, err
	}

	services, err := c.listServicesForPod(pod)
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

func listServiceNames(meshServices []service.MeshService) (serviceNames []string) {
	for _, meshService := range meshServices {
		serviceNames = append(serviceNames, fmt.Sprintf("%s/%s", meshService.Namespace, meshService.Name))
	}
	return serviceNames
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

// listServicesForPod lists Kubernetes services whose selectors match pod labels
func (c Client) listServicesForPod(pod *v1.Pod) ([]v1.Service, error) {
	var serviceList []v1.Service
	svcList := c.ListServices()

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
