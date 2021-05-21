package registry

import (
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
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

	services := listServicesForPod(pod, k.KubeController)

	if len(services) == 0 {
		return nil, nil
	}

	meshServices := kubernetesServicesToMeshServices(services)

	servicesForPod := strings.Join(listServiceNames(meshServices), ",")
	log.Trace().Msgf("Services associated with Pod with UID=%s Name=%s/%s: %+v",
		pod.ObjectMeta.UID, pod.Namespace, pod.Name, servicesForPod)

	return meshServices, nil
}

// AsyncKubeProxyServiceMapper maps an Envoy instance to services in a
// Kubernetes cluster. It maintains a cache of the mapping updated in response
// to Kubernetes events.
type AsyncKubeProxyServiceMapper struct {
	kubeController k8s.Controller
	kubeEvents     chan interface{}
	servicesForCN  map[certificate.CommonName][]service.MeshService
	cnsForService  map[service.MeshService]map[certificate.CommonName]struct{}
	cacheLock      sync.RWMutex
}

// NewAsyncKubeProxyServiceMapper initializes a KubeProxyServiceMapper with an empty cache.
func NewAsyncKubeProxyServiceMapper(controller k8s.Controller) *AsyncKubeProxyServiceMapper {
	return &AsyncKubeProxyServiceMapper{
		kubeController: controller,
		servicesForCN:  make(map[certificate.CommonName][]service.MeshService),
		cnsForService:  make(map[service.MeshService]map[certificate.CommonName]struct{}),
		kubeEvents: events.GetPubSubInstance().Subscribe(
			announcements.PodAdded,
			announcements.PodUpdated,
			announcements.PodDeleted,
			announcements.ServiceAdded,
			announcements.ServiceUpdated,
			announcements.ServiceDeleted,
		),
	}
}

// Run starts updating the proxy-to-services cache based on k8s notifications.
func (k *AsyncKubeProxyServiceMapper) Run(stop <-chan struct{}) {
	// Populate the cache using the cluster's existing state. Otherwise if
	// existing resources are not modified, no events will come through to
	// update the cache.
	k.cacheLock.Lock()
	for _, pod := range k.kubeController.ListPods() {
		k.handlePodUpdate(pod)
	}
	k.cacheLock.Unlock()

	go func() {
		for {
			select {
			case <-stop:
				events.GetPubSubInstance().Unsub(k.kubeEvents)
				return
			case ev := <-k.kubeEvents:
				event, ok := ev.(events.PubSubMessage)
				if !ok {
					log.Error().Msgf("ignoring unexpected pubsub message type: %T %v", ev, ev)
					continue
				}
				k.cacheLock.Lock()
				switch event.AnnouncementType {
				case announcements.PodAdded, announcements.PodUpdated:
					pod := event.NewObj.(*v1.Pod)
					k.handlePodUpdate(pod)
				case announcements.PodDeleted:
					pod := event.OldObj.(*v1.Pod)
					k.handlePodDelete(pod)
				case announcements.ServiceAdded, announcements.ServiceUpdated:
					svc := event.NewObj.(*v1.Service)
					k.handleServiceUpdate(svc)
				case announcements.ServiceDeleted:
					svc := event.OldObj.(*v1.Service)
					k.handleServiceDelete(svc)
				}
				k.cacheLock.Unlock()
				events.GetPubSubInstance().Publish(events.PubSubMessage{
					AnnouncementType: announcements.ScheduleProxyBroadcast,
				})
			}
		}
	}()
}

func (k *AsyncKubeProxyServiceMapper) handlePodUpdate(pod *v1.Pod) {
	if pod == nil {
		return
	}
	cn, err := getCertCommonNameForPod(*pod)
	if err != nil {
		log.Error().Err(err).Msgf("ignoring updated pod %s/%s", pod.Namespace, pod.Name)
		return
	}

	services := listServicesForPod(pod, k.kubeController)

	meshServices := kubernetesServicesToMeshServices(services)

	servicesForPod := strings.Join(listServiceNames(meshServices), ",")
	log.Trace().Msgf("Services associated with Pod with UID=%s Name=%s/%s: %+v",
		pod.ObjectMeta.UID, pod.Namespace, pod.Name, servicesForPod)

	k.servicesForCN[cn] = meshServices

	for _, svc := range meshServices {
		if k.cnsForService[svc] == nil {
			k.cnsForService[svc] = make(map[certificate.CommonName]struct{})
		}
		k.cnsForService[svc][cn] = struct{}{}
	}
}

func (k *AsyncKubeProxyServiceMapper) handlePodDelete(pod *v1.Pod) {
	if pod == nil {
		return
	}
	cn, err := getCertCommonNameForPod(*pod)
	if err != nil {
		log.Error().Err(err).Msgf("ignoring deleted pod %s/%s", pod.Namespace, pod.Name)
		return
	}

	for _, svc := range k.servicesForCN[cn] {
		delete(k.cnsForService[svc], cn)
	}

	delete(k.servicesForCN, cn)
}

func (k *AsyncKubeProxyServiceMapper) handleServiceUpdate(svc *v1.Service) {
	if svc == nil {
		return
	}

	updatedSvc := service.MeshService{
		Name:      svc.Name,
		Namespace: svc.Namespace,
	}
	if k.cnsForService[updatedSvc] == nil {
		k.cnsForService[updatedSvc] = make(map[certificate.CommonName]struct{})
	}

	pods := listPodsForService(svc, k.kubeController)
	for _, pod := range pods {
		cn, err := getCertCommonNameForPod(pod)
		if err != nil {
			log.Error().Err(err)
			continue
		}
		alreadyCached := false
		for _, cachedSvc := range k.servicesForCN[cn] {
			if cachedSvc == updatedSvc {
				alreadyCached = true
				break
			}
		}
		if !alreadyCached {
			k.servicesForCN[cn] = append(k.servicesForCN[cn], updatedSvc)
		}

		k.cnsForService[updatedSvc][cn] = struct{}{}
	}
}

func (k *AsyncKubeProxyServiceMapper) handleServiceDelete(svc *v1.Service) {
	if svc == nil {
		return
	}

	deleted := service.MeshService{
		Name:      svc.Name,
		Namespace: svc.Namespace,
	}
	for cn := range k.cnsForService[deleted] {
		var rem []service.MeshService
		for _, s := range k.servicesForCN[cn] {
			if s == deleted {
				continue
			}
			rem = append(rem, s)
		}
		k.servicesForCN[cn] = rem
	}
	delete(k.cnsForService, deleted)
}

// ListProxyServices maps an Envoy instance to a number of Kubernetes services.
func (k *AsyncKubeProxyServiceMapper) ListProxyServices(p *envoy.Proxy) ([]service.MeshService, error) {
	k.cacheLock.RLock()
	defer k.cacheLock.RUnlock()
	return k.servicesForCN[p.GetCertificateCommonName()], nil
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

// listServicesForPod lists Kubernetes services whose selectors match pod labels
func listServicesForPod(pod *v1.Pod, kubeController k8s.Controller) []v1.Service {
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

	return serviceList
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
	cn := envoy.NewCertCommonName(proxyUID, envoy.KindSidecar, pod.Spec.ServiceAccountName, pod.Namespace)
	return cn, nil
}
