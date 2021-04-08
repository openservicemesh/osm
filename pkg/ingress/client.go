package ingress

import (
	"reflect"

	networkingV1 "k8s.io/api/networking/v1"
	networkingV1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/configurator"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// ingressKind denotes the Kind attribute of the Ingress k8s resource
	ingressKind = "Ingress"
)

var candidateVersions = []string{networkingV1.SchemeGroupVersion.String(), networkingV1beta1.SchemeGroupVersion.String()}

// NewIngressClient implements ingress.Monitor and creates the Kubernetes client to monitor Ingress resources.
func NewIngressClient(kubeClient kubernetes.Interface, kubeController k8s.Controller, stop chan struct{}, cfg configurator.Configurator) (Monitor, error) {
	supportedIngressVersions, err := getSupportedIngressVersions(kubeClient.Discovery())
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving ingress API versions supported by k8s API server")
		return nil, err
	}

	// Initialize the version specific ingress informers and caches
	informerFactory := informers.NewSharedInformerFactory(kubeClient, k8s.DefaultKubeEventResyncInterval)
	ingressEventTypes := k8s.EventTypes{
		Add:    announcements.IngressAdded,
		Update: announcements.IngressUpdated,
		Delete: announcements.IngressDeleted,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return kubeController.IsMonitoredNamespace(ns)
	}

	client := Client{
		cacheSynced:    make(chan interface{}),
		kubeController: kubeController,
	}

	if v1Supported, ok := supportedIngressVersions[networkingV1.SchemeGroupVersion.String()]; ok && v1Supported {
		client.informerV1 = informerFactory.Networking().V1().Ingresses().Informer()
		client.cacheV1 = client.informerV1.GetStore()
		client.informerV1.AddEventHandler(k8s.GetKubernetesEventHandlers("IngressV1", "Kubernetes", shouldObserve, ingressEventTypes))
	}

	if v1beta1Supported, ok := supportedIngressVersions[networkingV1beta1.SchemeGroupVersion.String()]; ok && v1beta1Supported {
		client.informerV1beta1 = informerFactory.Networking().V1beta1().Ingresses().Informer()
		client.cacheV1Beta1 = client.informerV1beta1.GetStore()
		client.informerV1beta1.AddEventHandler(k8s.GetKubernetesEventHandlers("IngressV1beta1", "Kubernetes", shouldObserve, ingressEventTypes))
	}

	if err := client.run(stop); err != nil {
		log.Error().Err(err).Msg("Could not start Kubernetes Ingress client")
		return nil, err
	}

	return client, nil
}

// run executes informer collection.
func (c *Client) run(stop <-chan struct{}) error {
	log.Info().Msg("Ingress client started")

	if c.informerV1 == nil && c.informerV1beta1 == nil {
		log.Error().Err(errInitInformers).Msgf("k8s API must support at least one of %v, but does not support any ingress API", candidateVersions)
		return errInitInformers
	}

	informerCollection := map[string]cache.SharedIndexInformer{
		"IngressV1":      c.informerV1,
		"IngressV1beta1": c.informerV1beta1,
	}

	var pendingCacheSync []cache.InformerSynced
	for name, informer := range informerCollection {
		// Depending on k8s server version, an informer may or may not be initialized
		if informer == nil {
			continue
		}
		log.Info().Msgf("Starting ingress informer: %s", name)
		go informer.Run(stop)
		pendingCacheSync = append(pendingCacheSync, informer.HasSynced)
	}

	log.Info().Msgf("Waiting for ingress informer's cache to sync")
	if !cache.WaitForCacheSync(stop, pendingCacheSync...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that caches have been synced.
	close(c.cacheSynced)

	log.Info().Msgf("Cache sync finished for ingress informer")
	return nil
}

// GetIngressNetworkingV1beta1 returns the networking.k8s.io/v1beta1 ingress resources whose backends correspond to the service
func (c Client) GetIngressNetworkingV1beta1(meshService service.MeshService) ([]*networkingV1beta1.Ingress, error) {
	if c.cacheV1Beta1 == nil {
		// The v1beta1 version is not served by the controller, return an empty list
		return nil, nil
	}

	var ingressResources []*networkingV1beta1.Ingress
	for _, ingressInterface := range c.cacheV1Beta1.List() {
		ingress, ok := ingressInterface.(*networkingV1beta1.Ingress)
		if !ok {
			log.Error().Msg("Failed type assertion for Ingress in ingress cache")
			continue
		}

		// Extra safety - make sure we do not pay attention to Ingresses outside of observed namespaces
		if !c.kubeController.IsMonitoredNamespace(ingress.Namespace) {
			continue
		}

		// Check if the ingress resource belongs to the same namespace as the service
		if ingress.Namespace != meshService.Namespace {
			// The ingress resource does not belong to the namespace of the service
			continue
		}
		if backend := ingress.Spec.Backend; backend != nil && backend.ServiceName == meshService.Name {
			// Default backend service
			ingressResources = append(ingressResources, ingress)
			continue
		}

	ingressRule:
		for _, rule := range ingress.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.ServiceName == meshService.Name {
					ingressResources = append(ingressResources, ingress)
					break ingressRule
				}
			}
		}
	}
	return ingressResources, nil
}

// GetIngressNetworkingV1 returns the networking.k8s.io/v1 ingress resources whose backends correspond to the service
func (c Client) GetIngressNetworkingV1(meshService service.MeshService) ([]*networkingV1.Ingress, error) {
	if c.cacheV1 == nil {
		// The v1 version is not served by the controller, return an empty list
		return nil, nil
	}

	var ingressResources []*networkingV1.Ingress
	for _, ingressInterface := range c.cacheV1.List() {
		ingress, ok := ingressInterface.(*networkingV1.Ingress)
		if !ok {
			log.Error().Msg("Failed type assertion for Ingress in ingress cache")
			continue
		}

		// Extra safety - make sure we do not pay attention to Ingresses outside of observed namespaces
		if !c.kubeController.IsMonitoredNamespace(ingress.Namespace) {
			continue
		}

		// Check if the ingress resource belongs to the same namespace as the service
		if ingress.Namespace != meshService.Namespace {
			// The ingress resource does not belong to the namespace of the service
			continue
		}
		if backend := ingress.Spec.DefaultBackend; backend != nil && backend.Service.Name == meshService.Name {
			// Default backend service
			ingressResources = append(ingressResources, ingress)
			continue
		}

	ingressRule:
		for _, rule := range ingress.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service.Name == meshService.Name {
					ingressResources = append(ingressResources, ingress)
					break ingressRule
				}
			}
		}
	}
	return ingressResources, nil
}

// getSupportedIngressVersions returns a map comprising of keys matching candidate ingress API versions
// and corresponding values indidicating if they are supported by the k8s API server or not. An error
// is returned in case this cannot be determined.
// Example return values:
// - only networking.k8s.io/v1 is supported: {'networking.k8s.io/v1': true, 'networking.k8s.io/v1beta1': false}, nil
// - only networking.k8s.io/v1beta1 is supported: {'networking.k8s.io/v1': false, 'networking.k8s.io/v1beta1': true}, nil
// - both networking.k8s.io/v1 and networking.k8s.io/v1beta1 are supported: {'networking.k8s.io/v1': true, 'networking.k8s.io/v1beta1': true}, nil
// - on error: nil, error
func getSupportedIngressVersions(client discovery.ServerResourcesInterface) (map[string]bool, error) {
	versions := make(map[string]bool)

	for _, groupVersion := range candidateVersions {
		// Initialize version support to false before checking with the k8s API server
		versions[groupVersion] = false

		list, err := client.ServerResourcesForGroupVersion(groupVersion)
		if err != nil {
			return nil, err
		}

		for _, elem := range list.APIResources {
			if elem.Kind == ingressKind {
				versions[groupVersion] = true
				break
			}
		}
	}

	return versions, nil
}
