package kubernetes

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewKubernetesController returns a new kubernetes.Controller which means to provide access to locally-cached k8s resources
func NewKubernetesController(kubeClient kubernetes.Interface, meshName string, stop chan struct{}) (Controller, error) {
	// Initialize client object
	client := Client{
		kubeClient:    kubeClient,
		meshName:      meshName,
		informers:     InformerCollection{},
		announcements: make(map[InformerKey]chan interface{}),
		cacheSynced:   make(chan interface{}),
	}

	// Initialize resources here
	client.initNamespaceMonitor()
	client.initServicesMonitor()
	client.initPodMonitor()

	if err := client.run(stop); err != nil {
		log.Error().Err(err).Msg("Could not start Kubernetes Namespaces client")
		return nil, err
	}

	return client, nil
}

// Initializes Namespace monitoring
func (c *Client) initNamespaceMonitor() {
	monitorNamespaceLabel := map[string]string{constants.OSMKubeResourceMonitorAnnotation: c.meshName}

	labelSelector := fields.SelectorFromSet(monitorNamespaceLabel).String()
	option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
		opt.LabelSelector = labelSelector
	})

	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.kubeClient, DefaultKubeEventResyncInterval, option)

	// Add informer
	c.informers[Namespaces] = informerFactory.Core().V1().Namespaces().Informer()

	// Announcement channel for Namespaces
	c.announcements[Namespaces] = make(chan interface{})

	// Add event handler to informer
	c.informers[Namespaces].AddEventHandler(GetKubernetesEventHandlers((string)(Namespaces), ProviderName, c.announcements[Namespaces], nil))
}

// Initializes Service monitoring
func (c *Client) initServicesMonitor() {
	informerFactory := informers.NewSharedInformerFactory(c.kubeClient, DefaultKubeEventResyncInterval)
	c.informers[Services] = informerFactory.Core().V1().Services().Informer()

	// Function to filter Services by Namespace
	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return c.IsMonitoredNamespace(ns)
	}

	// Announcement channel for Services
	c.announcements[Services] = make(chan interface{})

	c.informers[Services].AddEventHandler(GetKubernetesEventHandlers((string)(Services), ProviderName, c.announcements[Services], shouldObserve))
}

func (c *Client) initPodMonitor() {
	informerFactory := informers.NewSharedInformerFactory(c.kubeClient, DefaultKubeEventResyncInterval)
	c.informers[Pods] = informerFactory.Core().V1().Pods().Informer()

	// Function to filter Pods by Namespace
	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return c.IsMonitoredNamespace(ns)
	}

	// Announcement channel for Pods
	c.announcements[Pods] = make(chan interface{})

	c.informers[Pods].AddEventHandler(GetKubernetesEventHandlers((string)(Pods), ProviderName, c.announcements[Services], shouldObserve))
}

func (c *Client) run(stop <-chan struct{}) error {
	log.Info().Msg("Namespace controller client started")
	var hasSynced []cache.InformerSynced
	var names []string

	if c.informers == nil {
		return errInitInformers
	}

	for name, informer := range c.informers {
		if informer == nil {
			continue
		}

		go informer.Run(stop)
		names = append(names, (string)(name))
		log.Info().Msgf("Waiting informer for %s cache sync...", name)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that caches have synced.
	close(c.cacheSynced)
	log.Info().Msgf("Caches for %+s synced successfully", names)

	return nil
}

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (c Client) IsMonitoredNamespace(namespace string) bool {
	_, exists, _ := c.informers[Namespaces].GetStore().GetByKey(namespace)
	return exists
}

// ListMonitoredNamespaces returns all namespaces that the mesh is monitoring.
func (c Client) ListMonitoredNamespaces() ([]string, error) {
	var namespaces []string

	for _, ns := range c.informers[Namespaces].GetStore().List() {
		namespace, ok := ns.(*corev1.Namespace)
		if !ok {
			log.Error().Err(errListingNamespaces).Msg("Failed to list monitored namespaces")
			continue
		}
		namespaces = append(namespaces, namespace.Name)
	}
	return namespaces, nil
}

// GetService retrieves the Kubernetes Services resource for the given MeshService
func (c Client) GetService(svc service.MeshService) *corev1.Service {
	// client-go cache uses <namespace>/<name> as key
	svcIf, exists, err := c.informers[Services].GetStore().GetByKey(svc.String())
	if exists && err == nil {
		svc := svcIf.(*corev1.Service)
		return svc
	}
	return nil
}

// ListServices returns a list of services that are part of monitored namespaces
func (c Client) ListServices() []*corev1.Service {
	var services []*corev1.Service

	for _, serviceInterface := range c.informers[Services].GetStore().List() {
		svc := serviceInterface.(*corev1.Service)

		if !c.IsMonitoredNamespace(svc.Namespace) {
			continue
		}
		services = append(services, svc)
	}
	return services
}

// GetAnnouncementsChannel gets the Announcements channel back
func (c Client) GetAnnouncementsChannel(informerID InformerKey) <-chan interface{} {
	return c.announcements[informerID]
}

// GetNamespace returns a Namespace resource if found, nil otherwise.
func (c Client) GetNamespace(ns string) *corev1.Namespace {
	nsIf, exists, err := c.informers[Namespaces].GetStore().GetByKey(ns)
	if exists && err == nil {
		ns := nsIf.(*corev1.Namespace)
		return ns
	}
	return nil
}

// ListPods returns a list of pods part of the mesh
// Kubecontroller does not currently segment pod notifications, hence it receives notifications
// for all k8s Pods.
func (c Client) ListPods() []*corev1.Pod {
	var pods []*corev1.Pod

	for _, podInterface := range c.informers[Pods].GetStore().List() {
		pod := podInterface.(*corev1.Pod)
		if !c.IsMonitoredNamespace(pod.Namespace) {
			continue
		}
		pods = append(pods, pod)
	}
	return pods
}
