package k8s

import (
	"context"
	"strconv"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	policyv1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewKubernetesController returns a new kubernetes.Controller which means to provide access to locally-cached k8s resources
func NewKubernetesController(kubeClient kubernetes.Interface, policyClient policyv1alpha1Client.Interface, meshName string, stop chan struct{}, selectInformers ...InformerKey) (Controller, error) {
	// Initialize client object
	c := client{
		kubeClient:   kubeClient,
		policyClient: policyClient,
		meshName:     meshName,
		informers:    informerCollection{},
	}

	// Initialize informers
	informerInitHandlerMap := map[InformerKey]func(){
		Namespaces:      c.initNamespaceMonitor,
		Services:        c.initServicesMonitor,
		ServiceAccounts: c.initServiceAccountsMonitor,
		Pods:            c.initPodMonitor,
		Endpoints:       c.initEndpointMonitor,
	}

	// If specific informers are not selected to be initialized, initialize all informers
	if len(selectInformers) == 0 {
		selectInformers = []InformerKey{Namespaces, Services, ServiceAccounts, Pods, Endpoints}
	}

	for _, informer := range selectInformers {
		informerInitHandlerMap[informer]()
	}

	if err := c.run(stop); err != nil {
		log.Error().Err(err).Msg("Could not start Kubernetes Namespaces client")
		return nil, err
	}

	return c, nil
}

// Initializes Namespace monitoring
func (c *client) initNamespaceMonitor() {
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
func (c *client) shouldObserve(obj interface{}) bool {
	object, ok := obj.(metav1.Object)
	if !ok {
		return false
	}
	return c.IsMonitoredNamespace(object.GetNamespace())
}

// Initializes Service monitoring
func (c *client) initServicesMonitor() {
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
func (c *client) initServiceAccountsMonitor() {
	informerFactory := informers.NewSharedInformerFactory(c.kubeClient, DefaultKubeEventResyncInterval)
	c.informers[ServiceAccounts] = informerFactory.Core().V1().ServiceAccounts().Informer()

	svcEventTypes := EventTypes{
		Add:    announcements.ServiceAccountAdded,
		Update: announcements.ServiceAccountUpdated,
		Delete: announcements.ServiceAccountDeleted,
	}
	c.informers[ServiceAccounts].AddEventHandler(GetKubernetesEventHandlers((string)(ServiceAccounts), providerName, c.shouldObserve, svcEventTypes))
}

func (c *client) initPodMonitor() {
	informerFactory := informers.NewSharedInformerFactory(c.kubeClient, DefaultKubeEventResyncInterval)
	c.informers[Pods] = informerFactory.Core().V1().Pods().Informer()

	podEventTypes := EventTypes{
		Add:    announcements.PodAdded,
		Update: announcements.PodUpdated,
		Delete: announcements.PodDeleted,
	}
	c.informers[Pods].AddEventHandler(GetKubernetesEventHandlers((string)(Pods), providerName, c.shouldObserve, podEventTypes))
}

func (c *client) initEndpointMonitor() {
	informerFactory := informers.NewSharedInformerFactory(c.kubeClient, DefaultKubeEventResyncInterval)
	c.informers[Endpoints] = informerFactory.Core().V1().Endpoints().Informer()

	eptEventTypes := EventTypes{
		Add:    announcements.EndpointAdded,
		Update: announcements.EndpointUpdated,
		Delete: announcements.EndpointDeleted,
	}
	c.informers[Endpoints].AddEventHandler(GetKubernetesEventHandlers((string)(Endpoints), providerName, c.shouldObserve, eptEventTypes))
}

func (c *client) run(stop <-chan struct{}) error {
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

	log.Info().Msgf("Caches for %v synced successfully", names)

	return nil
}

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (c client) IsMonitoredNamespace(namespace string) bool {
	_, exists, _ := c.informers[Namespaces].GetStore().GetByKey(namespace)
	return exists
}

// ListMonitoredNamespaces returns all namespaces that the mesh is monitoring.
func (c client) ListMonitoredNamespaces() ([]string, error) {
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
func (c client) GetService(svc service.MeshService) *corev1.Service {
	// client-go cache uses <namespace>/<name> as key
	svcIf, exists, err := c.informers[Services].GetStore().GetByKey(svc.NameWithoutCluster())
	if exists && err == nil {
		svc := svcIf.(*corev1.Service)
		return svc
	}
	return nil
}

// ListServices returns a list of services that are part of monitored namespaces
func (c client) ListServices() []*corev1.Service {
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
func (c client) ListServiceAccounts() []*corev1.ServiceAccount {
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
func (c client) GetNamespace(ns string) *corev1.Namespace {
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
func (c client) ListPods() []*corev1.Pod {
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
func (c client) GetEndpoints(svc service.MeshService) (*corev1.Endpoints, error) {
	ep, exists, err := c.informers[Endpoints].GetStore().GetByKey(svc.NameWithoutCluster())
	if err != nil {
		return nil, err
	}
	if exists {
		return ep.(*corev1.Endpoints), nil
	}
	return nil, nil
}

// ListServiceIdentitiesForService lists ServiceAccounts associated with the given service
func (c client) ListServiceIdentitiesForService(svc service.MeshService) ([]identity.K8sServiceAccount, error) {
	var svcAccounts []identity.K8sServiceAccount

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
			podSvcAccount := identity.K8sServiceAccount{
				Name:      pod.Spec.ServiceAccountName,
				Namespace: pod.Namespace, // ServiceAccount must belong to the same namespace as the pod
			}
			svcAccountsSet.Add(podSvcAccount)
		}
	}

	for svcAcc := range svcAccountsSet.Iter() {
		svcAccounts = append(svcAccounts, svcAcc.(identity.K8sServiceAccount))
	}
	return svcAccounts, nil
}

// IsMetricsEnabled returns true if the pod in the mesh is correctly annotated for prometheus scrapping
func (c client) IsMetricsEnabled(pod *corev1.Pod) bool {
	isScrapingEnabled := false
	prometheusScrapeAnnotation, ok := pod.Annotations[constants.PrometheusScrapeAnnotation]
	if !ok {
		return isScrapingEnabled
	}

	isScrapingEnabled, _ = strconv.ParseBool(prometheusScrapeAnnotation)
	return isScrapingEnabled
}

// UpdateStatus updates the status subresource for the given resource and GroupVersionKind
// The resource within the 'interface{}' must be a pointer to the underlying resource
func (c client) UpdateStatus(resource interface{}) (metav1.Object, error) {
	switch t := resource.(type) {
	case *policyv1alpha1.IngressBackend:
		obj := resource.(*policyv1alpha1.IngressBackend)
		return c.policyClient.PolicyV1alpha1().IngressBackends(obj.Namespace).UpdateStatus(context.Background(), obj, metav1.UpdateOptions{})

	default:
		return nil, errors.Errorf("Unsupported type: %T", t)
	}
}
