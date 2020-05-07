package ingress

import (
	"reflect"
	"strings"

	extensionsV1beta "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	k8s "github.com/open-service-mesh/osm/pkg/kubernetes"
	"github.com/open-service-mesh/osm/pkg/namespace"
)

// NewIngressClient implements ingress.Monitor and creates the Kubernetes client to monitor Ingress resources.
func NewIngressClient(kubeConfig *rest.Config, namespaceController namespace.Controller, stop chan struct{}) (Monitor, error) {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	informerFactory := informers.NewSharedInformerFactory(kubeClient, k8s.DefaultKubeEventResyncInterval)
	informer := informerFactory.Extensions().V1beta1().Ingresses().Informer()

	client := Client{
		informer:            informer,
		cache:               informer.GetStore(),
		cacheSynced:         make(chan interface{}),
		announcements:       make(chan interface{}),
		namespaceController: namespaceController,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return namespaceController.IsMonitoredNamespace(ns)
	}
	informer.AddEventHandler(k8s.GetKubernetesEventHandlers("Ingress", "Kubernetes", client.announcements, shouldObserve))

	if err := client.run(stop); err != nil {
		log.Error().Err(err).Msg("Could not start Kubernetes Ingress client")
		return nil, err
	}

	return client, nil
}

// run executes informer collection.
func (c *Client) run(stop <-chan struct{}) error {
	log.Info().Msg("Ingress client started")

	if c.informer == nil {
		return errInitInformers
	}

	go c.informer.Run(stop)
	log.Info().Msgf("Waiting for Ingress informer cache sync")
	if !cache.WaitForCacheSync(stop, c.informer.HasSynced) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	log.Info().Msgf("Cache sync finished for Ingress informer")
	return nil
}

// GetAnnouncementsChannel returns the announcement channel for the Ingress client
func (c Client) GetAnnouncementsChannel() <-chan interface{} {
	return c.announcements
}

// GetIngressResources returns the ingress resources whose backends correspond to the service
func (c Client) GetIngressResources(service endpoint.NamespacedService) ([]*extensionsV1beta.Ingress, error) {
	var ingressResources []*extensionsV1beta.Ingress
	for _, ingressInterface := range c.cache.List() {
		ingress, ok := ingressInterface.(*extensionsV1beta.Ingress)
		if !ok {
			log.Error().Msg("Failed type assertion for Ingress in ingress cache")
			continue
		}
		// TODO(check if needed): Check if the ingress resource belongs to the overall list of monitored namespaces
		if !c.namespaceController.IsMonitoredNamespace(ingress.Namespace) {
			continue
		}
		// Check if the ingress resource belongs to the same namespace as the service
		if ingress.Namespace != service.Namespace {
			// The ingress resource does not belong to the namespace of the service
			continue
		}
		// Check if the ingress resources is annotated for monitor
		if enabled, err := isMonitoredResource(ingress); !enabled || err != nil {
			continue
		}
		if backend := ingress.Spec.Backend; backend != nil && backend.ServiceName == service.Service {
			// Default backend service
			ingressResources = append(ingressResources, ingress)
			continue
		}

	ingressRule:
		for _, rule := range ingress.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.ServiceName == service.Service {
					ingressResources = append(ingressResources, ingress)
					break ingressRule
				}
			}
		}
	}
	return ingressResources, nil
}

func isMonitoredResource(ingress *extensionsV1beta.Ingress) (bool, error) {
	monitor := strings.ToLower(ingress.ObjectMeta.Annotations[constants.OSMKubeResourceMonitorAnnotation])
	if monitor != "" {
		switch monitor {
		case "enabled", "yes", "true":
			return true, nil

		case "disabled", "no", "false":
			return false, nil

		default:
			log.Error().Err(errInvalidMonitorAnnotation).Msgf("Invalid annotation specified: %s", monitor)
			return false, errInvalidMonitorAnnotation
		}
	}
	return false, nil
}
