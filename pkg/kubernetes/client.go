package kubernetes

import (
	"reflect"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewKubernetesController returns a new kubernetes.Controller which means to provide access to locally-cached k8s resources
func NewKubernetesController(kubeClient kubernetes.Interface, meshName string, stop chan struct{}, selectInformers ...InformerKey) (Controller, error) {
	// Initialize client object
	client := Client{
		kubeClient:  kubeClient,
		meshName:    meshName,
		informers:   informerCollection{},
		cacheSynced: make(chan interface{}),
	}

	// Initialize informers
	informerInitHandlerMap := map[InformerKey]func(){
		Namespaces:      client.initNamespaceMonitor,
		Services:        client.initServicesMonitor,
		ServiceAccounts: client.initServiceAccountsMonitor,
		Pods:            client.initPodMonitor,
		Endpoints:       client.initEndpointMonitor,
	}

	// If specific informers are not selected to be initialized, initialize all informers
	if len(selectInformers) == 0 {
		selectInformers = []InformerKey{Namespaces, Services, ServiceAccounts, Pods, Endpoints}
	}

	for _, informer := range selectInformers {
		informerInitHandlerMap[informer]()
	}

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

	// Add event handler to informer
	nsEventTypes := EventTypes{
		Add:    announcements.NamespaceAdded,
		Update: announcements.NamespaceUpdated,
		Delete: announcements.NamespaceDeleted,
	}
	c.informers[Namespaces].AddEventHandler(GetKubernetesEventHandlers((string)(Namespaces), providerName, nil, nsEventTypes))
}

// Function to filter K8s meta Objects by OSM's isMonitoredNamespace
func (c *Client) shouldObserve(obj interface{}) bool {
	ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
	return c.IsMonitoredNamespace(ns)
}

// Initializes Service monitoring
func (c *Client) initServicesMonitor() {
	informerFactory := informers.NewSharedInformerFactory(c.kubeClient, DefaultKubeEventResyncInterval)
	c.informers[Services] = informerFactory.Core().V1().Services().Informer()

	svcEventTypes := EventTypes{
		Add:    announcements.ServiceAdded,
		Update: announcements.ServiceUpdated,
		Delete: announcements.ServiceDeleted,
	}
	c.informers[Services].AddEventHandler(GetKubernetesEventHandlers((string)(Services), providerName, c.shouldObserve, svcEventTypes))
}

// Initializes Service Account monitoring
func (c *Client) initServiceAccountsMonitor() {
	informerFactory := informers.NewSharedInformerFactory(c.kubeClient, DefaultKubeEventResyncInterval)
	c.informers[ServiceAccounts] = informerFactory.Core().V1().ServiceAccounts().Informer()

	svcEventTypes := EventTypes{
		Add:    announcements.ServiceAccountAdded,
		Update: announcements.ServiceAccountUpdated,
		Delete: announcements.ServiceAccountDeleted,
	}
	c.informers[ServiceAccounts].AddEventHandler(GetKubernetesEventHandlers((string)(ServiceAccounts), providerName, c.shouldObserve, svcEventTypes))
}

func (c *Client) initPodMonitor() {
	informerFactory := informers.NewSharedInformerFactory(c.kubeClient, DefaultKubeEventResyncInterval)
	c.informers[Pods] = informerFactory.Core().V1().Pods().Informer()

	podEventTypes := EventTypes{
		Add:    announcements.PodAdded,
		Update: announcements.PodUpdated,
		Delete: announcements.PodDeleted,
	}
	c.informers[Pods].AddEventHandler(GetKubernetesEventHandlers((string)(Pods), providerName, c.shouldObserve, podEventTypes))
}

func (c *Client) initEndpointMonitor() {
	informerFactory := informers.NewSharedInformerFactory(c.kubeClient, DefaultKubeEventResyncInterval)
	c.informers[Endpoints] = informerFactory.Core().V1().Endpoints().Informer()

	eptEventTypes := EventTypes{
		Add:    announcements.EndpointAdded,
		Update: announcements.EndpointUpdated,
		Delete: announcements.EndpointDeleted,
	}
	c.informers[Endpoints].AddEventHandler(GetKubernetesEventHandlers((string)(Endpoints), providerName, c.shouldObserve, eptEventTypes))
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
		log.Info().Msgf("Waiting for %s informer cache sync...", name)
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

// ListServiceAccounts returns a list of service accounts that are part of monitored namespaces
func (c Client) ListServiceAccounts() []*corev1.ServiceAccount {
	var serviceAccounts []*corev1.ServiceAccount

	for _, serviceInterface := range c.informers[ServiceAccounts].GetStore().List() {
		sa := serviceInterface.(*corev1.ServiceAccount)

		if !c.IsMonitoredNamespace(sa.Namespace) {
			continue
		}
		serviceAccounts = append(serviceAccounts, sa)
	}
	return serviceAccounts
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

// GetEndpoints returns the endpoint for a given service, otherwise returns nil if not found
// or error if the API errored out.
func (c Client) GetEndpoints(svc service.MeshService) (*corev1.Endpoints, error) {
	ep, exists, err := c.informers[Endpoints].GetStore().GetByKey(svc.String())
	if err != nil {
		return nil, err
	}
	if exists {
		return ep.(*corev1.Endpoints), nil
	}
	return nil, nil
}

// ListServiceAccountsForService lists ServiceAccounts associated with the given service
func (c Client) ListServiceAccountsForService(svc service.MeshService) ([]service.K8sServiceAccount, error) {
	var svcAccounts []service.K8sServiceAccount

	k8sSvc := c.GetService(svc)
	if k8sSvc == nil {
		return nil, errors.Errorf("Error fetching service %q: %s", svc, errServiceNotFound)
	}

	svcAccountsSet := mapset.NewSet()
	pods := c.ListPods()
	for _, pod := range pods {
		svcRawSelector := k8sSvc.Spec.Selector
		selector := labels.Set(svcRawSelector).AsSelector()
		// service has no selectors, we do not need to match against the pod label
		if len(svcRawSelector) == 0 {
			continue
		}
		if selector.Matches(labels.Set(pod.Labels)) {
			podSvcAccount := service.K8sServiceAccount{
				Name:      pod.Spec.ServiceAccountName,
				Namespace: pod.Namespace, // ServiceAccount must belong to the same namespace as the pod
			}
			svcAccountsSet.Add(podSvcAccount)
		}
	}

	for svcAcc := range svcAccountsSet.Iter() {
		svcAccounts = append(svcAccounts, svcAcc.(service.K8sServiceAccount))
	}
	return svcAccounts, nil
}
