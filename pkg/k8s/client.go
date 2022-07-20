package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	policyv1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	osminformers "github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewKubernetesController returns a new kubernetes.Controller which means to provide access to locally-cached k8s resources
func NewKubernetesController(informerCollection *osminformers.InformerCollection, policyClient policyv1alpha1Client.Interface, msgBroker *messaging.Broker, selectInformers ...InformerKey) Controller {
	return newClient(informerCollection, policyClient, msgBroker, selectInformers...)
}

func newClient(informerCollection *osminformers.InformerCollection, policyClient policyv1alpha1Client.Interface, msgBroker *messaging.Broker, selectInformers ...InformerKey) *client {
	// Initialize client object
	c := &client{
		informers:    informerCollection,
		msgBroker:    msgBroker,
		policyClient: policyClient,
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

	return c
}

// Initializes Namespace monitoring
func (c *client) initNamespaceMonitor() {
	// Add event handler to informer
	nsEventTypes := EventTypes{
		Add:    announcements.NamespaceAdded,
		Update: announcements.NamespaceUpdated,
		Delete: announcements.NamespaceDeleted,
	}
	c.informers.AddEventHandler(osminformers.InformerKeyNamespace, GetEventHandlerFuncs(nil, nsEventTypes, c.msgBroker))
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
	svcEventTypes := EventTypes{
		Add:    announcements.ServiceAdded,
		Update: announcements.ServiceUpdated,
		Delete: announcements.ServiceDeleted,
	}
	c.informers.AddEventHandler(osminformers.InformerKeyService, GetEventHandlerFuncs(c.shouldObserve, svcEventTypes, c.msgBroker))
}

// Initializes Service Account monitoring
func (c *client) initServiceAccountsMonitor() {
	svcEventTypes := EventTypes{
		Add:    announcements.ServiceAccountAdded,
		Update: announcements.ServiceAccountUpdated,
		Delete: announcements.ServiceAccountDeleted,
	}
	c.informers.AddEventHandler(osminformers.InformerKeyServiceAccount, GetEventHandlerFuncs(c.shouldObserve, svcEventTypes, c.msgBroker))
}

func (c *client) initPodMonitor() {
	podEventTypes := EventTypes{
		Add:    announcements.PodAdded,
		Update: announcements.PodUpdated,
		Delete: announcements.PodDeleted,
	}
	c.informers.AddEventHandler(osminformers.InformerKeyPod, GetEventHandlerFuncs(c.shouldObserve, podEventTypes, c.msgBroker))
}

func (c *client) initEndpointMonitor() {
	eptEventTypes := EventTypes{
		Add:    announcements.EndpointAdded,
		Update: announcements.EndpointUpdated,
		Delete: announcements.EndpointDeleted,
	}
	c.informers.AddEventHandler(osminformers.InformerKeyEndpoints, GetEventHandlerFuncs(c.shouldObserve, eptEventTypes, c.msgBroker))
}

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (c client) IsMonitoredNamespace(namespace string) bool {
	return c.informers.IsMonitoredNamespace(namespace)
}

// ListMonitoredNamespaces returns all namespaces that the mesh is monitoring.
func (c client) ListMonitoredNamespaces() ([]string, error) {
	var namespaces []string

	for _, ns := range c.informers.List(osminformers.InformerKeyNamespace) {
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
	svcIf, exists, err := c.informers.GetByKey(osminformers.InformerKeyService, svc.NamespacedKey())
	if exists && err == nil {
		svc := svcIf.(*corev1.Service)
		return svc
	}
	return nil
}

// ListServices returns a list of services that are part of monitored namespaces
func (c client) ListServices() []*corev1.Service {
	var services []*corev1.Service

	for _, serviceInterface := range c.informers.List(osminformers.InformerKeyService) {
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

	for _, serviceInterface := range c.informers.List(osminformers.InformerKeyServiceAccount) {
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
	nsIf, exists, err := c.informers.GetByKey(osminformers.InformerKeyNamespace, ns)
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

	for _, podInterface := range c.informers.List(osminformers.InformerKeyPod) {
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
	ep, exists, err := c.informers.GetByKey(osminformers.InformerKeyEndpoints, svc.NamespacedKey())
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
		return nil, fmt.Errorf("Error fetching service %q: %s", svc, errServiceNotFound)
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
func IsMetricsEnabled(pod *corev1.Pod) bool {
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

	case *policyv1alpha1.UpstreamTrafficSetting:
		obj := resource.(*policyv1alpha1.UpstreamTrafficSetting)
		return c.policyClient.PolicyV1alpha1().UpstreamTrafficSettings(obj.Namespace).UpdateStatus(context.Background(), obj, metav1.UpdateOptions{})

	default:
		return nil, fmt.Errorf("Unsupported type: %T", t)
	}
}

// ServiceToMeshServices translates a k8s service with one or more ports to one or more
// MeshService objects per port.
func ServiceToMeshServices(c Controller, svc corev1.Service) []service.MeshService {
	var meshServices []service.MeshService

	for _, portSpec := range svc.Spec.Ports {
		meshSvc := service.MeshService{
			Namespace: svc.Namespace,
			Name:      svc.Name,
			Port:      uint16(portSpec.Port),
		}

		// attempt to parse protocol from port name
		// Order of Preference is:
		// 1. port.appProtocol field
		// 2. protocol prefixed to port name (e.g. tcp-my-port)
		// 3. default to http
		protocol := constants.ProtocolHTTP
		for _, p := range constants.SupportedProtocolsInMesh {
			if strings.HasPrefix(portSpec.Name, p+"-") {
				protocol = p
				break
			}
		}

		// use port.appProtocol if specified, else use port protocol
		meshSvc.Protocol = pointer.StringDeref(portSpec.AppProtocol, protocol)

		// The endpoints for the kubernetes service carry information that allows
		// us to retrieve the TargetPort for the MeshService.
		endpoints, _ := c.GetEndpoints(meshSvc)
		if endpoints != nil {
			meshSvc.TargetPort = GetTargetPortFromEndpoints(portSpec.Name, *endpoints)
		} else {
			log.Warn().Msgf("k8s service %s/%s does not have endpoints but is being represented as a MeshService", svc.Namespace, svc.Name)
		}

		if !IsHeadlessService(svc) || endpoints == nil {
			meshServices = append(meshServices, meshSvc)
			continue
		}

		for _, subset := range endpoints.Subsets {
			for _, address := range subset.Addresses {
				if address.Hostname == "" {
					continue
				}
				meshServices = append(meshServices, service.MeshService{
					Namespace:  svc.Namespace,
					Name:       fmt.Sprintf("%s.%s", address.Hostname, svc.Name),
					Port:       meshSvc.Port,
					TargetPort: meshSvc.TargetPort,
					Protocol:   meshSvc.Protocol,
				})
			}
		}
	}

	return meshServices
}

// GetTargetPortFromEndpoints returns the endpoint port corresponding to the given endpoint name and endpoints
func GetTargetPortFromEndpoints(endpointName string, endpoints corev1.Endpoints) (endpointPort uint16) {
	// Per https://pkg.go.dev/k8s.io/api/core/v1#ServicePort and
	// https://pkg.go.dev/k8s.io/api/core/v1#EndpointPort, if a service has multiple
	// ports, then ServicePort.Name must match EndpointPort.Name when considering
	// matching endpoints for the service's port. ServicePort.Name and EndpointPort.Name
	// can be unset when the service has a single port exposed, in which case we are
	// guaranteed to have the same port specified in the list of EndpointPort.Subsets.
	//
	// The logic below works as follows:
	// If the service has multiple ports, retrieve the matching endpoint port using
	// the given ServicePort.Name specified by `endpointName`.
	// Otherwise, simply return the only port referenced in EndpointPort.Subsets.
	for _, subset := range endpoints.Subsets {
		for _, port := range subset.Ports {
			if endpointName == "" || len(subset.Ports) == 1 {
				// ServicePort.Name is not passed or a single port exists on the service.
				// Both imply that this service has a single ServicePort and EndpointPort.
				endpointPort = uint16(port.Port)
				return
			}

			// If more than 1 port is specified
			if port.Name == endpointName {
				endpointPort = uint16(port.Port)
				return
			}
		}
	}
	return
}

func (c client) GetPodForProxy(proxy *envoy.Proxy) (*v1.Pod, error) {
	proxyUUID, svcAccount := proxy.UUID.String(), proxy.Identity.ToK8sServiceAccount()
	log.Trace().Msgf("Looking for pod with label %q=%q", constants.EnvoyUniqueIDLabelName, proxyUUID)
	podList := c.ListPods()
	var pods []v1.Pod

	for _, pod := range podList {
		if uuid, labelFound := pod.Labels[constants.EnvoyUniqueIDLabelName]; labelFound && uuid == proxyUUID {
			pods = append(pods, *pod)
		}
	}

	if len(pods) == 0 {
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingPodFromCert)).
			Msgf("Did not find Pod with label %s = %s in namespace %s",
				constants.EnvoyUniqueIDLabelName, proxyUUID, svcAccount.Namespace)
		return nil, errDidNotFindPodForUUID
	}

	// Each pod is assigned a unique UUID at the time of sidecar injection.
	// The certificate's CommonName encodes this UUID, and we lookup the pod
	// whose label matches this UUID.
	// Only 1 pod must match the UUID encoded in the given certificate. If multiple
	// pods match, it is an error.
	if len(pods) > 1 {
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrPodBelongsToMultipleServices)).
			Msgf("Found more than one pod with label %s = %s in namespace %s. There can be only one!",
				constants.EnvoyUniqueIDLabelName, proxyUUID, svcAccount.Namespace)
		return nil, errMoreThanOnePodForUUID
	}

	pod := pods[0]
	log.Trace().Msgf("Found Pod with UID=%s for proxyID %s", pod.ObjectMeta.UID, proxyUUID)

	if pod.Namespace != svcAccount.Namespace {
		log.Warn().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingPodFromCert)).
			Msgf("Pod with UID=%s belongs to Namespace %s. The pod's xDS certificate was issued for Namespace %s",
				pod.ObjectMeta.UID, pod.Namespace, svcAccount.Namespace)
		return nil, errNamespaceDoesNotMatchProxy
	}

	// Ensure the Name encoded in the certificate matches that of the Pod
	// TODO(draychev): check that the Kind matches too! [https://github.com/openservicemesh/osm/issues/3173]
	if pod.Spec.ServiceAccountName != svcAccount.Name {
		// Since we search for the pod in the namespace we obtain from the certificate -- these namespaces will always match.
		log.Warn().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingPodFromCert)).
			Msgf("Pod with UID=%s belongs to ServiceAccount=%s. The pod's xDS certificate was issued for ServiceAccount=%s",
				pod.ObjectMeta.UID, pod.Spec.ServiceAccountName, svcAccount)
		return nil, errServiceAccountDoesNotMatchProxy
	}

	return &pod, nil
}

// GetTargetPortForServicePort returns the TargetPort corresponding to the Port used by clients
// to communicate with it.
func (c client) GetTargetPortForServicePort(namespacedSvc types.NamespacedName, port uint16) (uint16, error) {
	// Lookup the k8s service corresponding to the given service name.
	// The k8s service is necessary to lookup the TargetPort from the Endpoint whose name
	// matches the name of the port on the k8s Service object.
	svcIf, exists, err := c.informers.GetByKey(osminformers.InformerKeyService, namespacedSvc.String())
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, fmt.Errorf("service %s not found in cache", namespacedSvc)
	}

	svc := svcIf.(*corev1.Service)
	var portName string
	for _, portSpec := range svc.Spec.Ports {
		if uint16(portSpec.Port) == port {
			portName = portSpec.Name
			break
		}
	}

	// Lookup the endpoint port (TargetPort) that matches the given service and 'portName'
	ep, exists, err := c.informers.GetByKey(osminformers.InformerKeyEndpoints, namespacedSvc.String())
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, fmt.Errorf("endpoint for service %s not found in cache", namespacedSvc)
	}
	endpoint := ep.(*corev1.Endpoints)

	for _, subset := range endpoint.Subsets {
		for _, portSpec := range subset.Ports {
			if portSpec.Name == portName {
				return uint16(portSpec.Port), nil
			}
		}
	}

	return 0, fmt.Errorf("error finding port name %s for endpoint %s", portName, namespacedSvc)
}
