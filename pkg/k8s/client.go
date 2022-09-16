package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	configv1alpha2Client "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	policyv1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	osminformers "github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// NewClient returns a new kubernetes.Controller which means to provide access to locally-cached k8s resources
func NewClient(osmNamespace, meshConfigName string, informerCollection *osminformers.InformerCollection, policyClient policyv1alpha1Client.Interface, configClient configv1alpha2Client.Interface, msgBroker *messaging.Broker, selectInformers ...InformerKey) *Client {
	// Initialize client object
	c := &Client{
		informers:      informerCollection,
		msgBroker:      msgBroker,
		policyClient:   policyClient,
		configClient:   configClient,
		osmNamespace:   osmNamespace,
		meshConfigName: meshConfigName,
	}

	// Initialize informers
	informerInitHandlerMap := map[InformerKey]func(){
		Namespaces:             c.initNamespaceMonitor,
		Services:               c.initServicesMonitor,
		ServiceAccounts:        c.initServiceAccountsMonitor,
		Pods:                   c.initPodMonitor,
		Endpoints:              c.initEndpointMonitor,
		MeshConfig:             c.initMeshConfigMonitor,
		MeshRootCertificate:    c.initMRCMonitor,
		Egress:                 c.initEgressMonitor,
		IngressBackend:         c.initIngressBackendMonitor,
		Retry:                  c.initRetryMonitor,
		UpstreamTrafficSetting: c.initUpstreamTrafficSettingMonitor,
	}

	// If specific informers are not selected to be initialized, initialize all informers
	if len(selectInformers) == 0 {
		selectInformers = []InformerKey{
			Namespaces, Services, ServiceAccounts, Pods, Endpoints, MeshConfig, MeshRootCertificate,
			Egress, IngressBackend, Retry, UpstreamTrafficSetting}
	}

	for _, informer := range selectInformers {
		informerInitHandlerMap[informer]()
	}

	return c
}

// Initializes Namespace monitoring
func (c *Client) initNamespaceMonitor() {
	// Add event handler to informer
	c.informers.AddEventHandler(osminformers.InformerKeyNamespace, GetEventHandlerFuncs(nil, c.msgBroker))
}

func (c *Client) initMeshConfigMonitor() {
	c.informers.AddEventHandler(osminformers.InformerKeyMeshConfig, GetEventHandlerFuncs(nil, c.msgBroker))
	c.informers.AddEventHandler(osminformers.InformerKeyMeshConfig, c.metricsHandler())
}

func (c *Client) initMRCMonitor() {
	c.informers.AddEventHandler(osminformers.InformerKeyMeshRootCertificate, GetEventHandlerFuncs(nil, c.msgBroker))
}

func (c *Client) initEgressMonitor() {
	c.informers.AddEventHandler(osminformers.InformerKeyEgress, GetEventHandlerFuncs(c.shouldObserve, c.msgBroker))
}

func (c *Client) initIngressBackendMonitor() {
	c.informers.AddEventHandler(osminformers.InformerKeyIngressBackend, GetEventHandlerFuncs(c.shouldObserve, c.msgBroker))
}

func (c *Client) initRetryMonitor() {
	c.informers.AddEventHandler(osminformers.InformerKeyRetry, GetEventHandlerFuncs(c.shouldObserve, c.msgBroker))
}

func (c *Client) initUpstreamTrafficSettingMonitor() {
	c.informers.AddEventHandler(osminformers.InformerKeyUpstreamTrafficSetting, GetEventHandlerFuncs(c.shouldObserve, c.msgBroker))
}

// Function to filter K8s meta Objects by OSM's isMonitoredNamespace
func (c *Client) shouldObserve(obj interface{}) bool {
	object, ok := obj.(metav1.Object)
	if !ok {
		return false
	}
	return c.IsMonitoredNamespace(object.GetNamespace())
}

// Initializes Service monitoring
func (c *Client) initServicesMonitor() {
	c.informers.AddEventHandler(osminformers.InformerKeyService, GetEventHandlerFuncs(c.shouldObserve, c.msgBroker))
}

// Initializes Service Account monitoring
func (c *Client) initServiceAccountsMonitor() {
	c.informers.AddEventHandler(osminformers.InformerKeyServiceAccount, GetEventHandlerFuncs(c.shouldObserve, c.msgBroker))
}

func (c *Client) initPodMonitor() {
	c.informers.AddEventHandler(osminformers.InformerKeyPod, GetEventHandlerFuncs(c.shouldObserve, c.msgBroker))
}

func (c *Client) initEndpointMonitor() {
	c.informers.AddEventHandler(osminformers.InformerKeyEndpoints, GetEventHandlerFuncs(c.shouldObserve, c.msgBroker))
}

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (c *Client) IsMonitoredNamespace(namespace string) bool {
	return c.informers.IsMonitoredNamespace(namespace)
}

func key(name, namespace string) string {
	return types.NamespacedName{Name: name, Namespace: namespace}.String()
}

// ListNamespaces returns all namespaces that the mesh is monitoring.
func (c *Client) ListNamespaces() ([]*corev1.Namespace, error) {
	var namespaces []*corev1.Namespace

	for _, ns := range c.informers.List(osminformers.InformerKeyNamespace) {
		namespace, ok := ns.(*corev1.Namespace)
		if !ok {
			log.Error().Err(errListingNamespaces).Msg("Failed to list monitored namespaces")
			continue
		}
		namespaces = append(namespaces, namespace)
	}
	return namespaces, nil
}

// GetService retrieves the Kubernetes Services resource for the given MeshService
func (c *Client) GetService(name, namespace string) *corev1.Service {
	// client-go cache uses <namespace>/<name> as key
	svcIf, exists, err := c.informers.GetByKey(osminformers.InformerKeyService, key(name, namespace))
	if exists && err == nil {
		svc := svcIf.(*corev1.Service)
		return svc
	}
	return nil
}

// ListServices returns a list of services that are part of monitored namespaces
func (c *Client) ListServices() []*corev1.Service {
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
func (c *Client) ListServiceAccounts() []*corev1.ServiceAccount {
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
func (c *Client) GetNamespace(ns string) *corev1.Namespace {
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
func (c *Client) ListPods() []*corev1.Pod {
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
func (c *Client) GetEndpoints(name, namespace string) (*corev1.Endpoints, error) {
	ep, exists, err := c.informers.GetByKey(osminformers.InformerKeyEndpoints, key(name, namespace))
	if err != nil {
		return nil, err
	}
	if exists {
		return ep.(*corev1.Endpoints), nil
	}
	return nil, nil
}

// UpdateIngressBackendStatus updates the status for the provided IngressBackend.
func (c *Client) UpdateIngressBackendStatus(obj *policyv1alpha1.IngressBackend) (*policyv1alpha1.IngressBackend, error) {
	return c.policyClient.PolicyV1alpha1().IngressBackends(obj.Namespace).UpdateStatus(context.Background(), obj, metav1.UpdateOptions{})
}

// UpdateUpstreamTrafficSettingStatus updates the status for the provided UpstreamTrafficSetting.
func (c *Client) UpdateUpstreamTrafficSettingStatus(obj *policyv1alpha1.UpstreamTrafficSetting) (*policyv1alpha1.UpstreamTrafficSetting, error) {
	return c.policyClient.PolicyV1alpha1().UpstreamTrafficSettings(obj.Namespace).UpdateStatus(context.Background(), obj, metav1.UpdateOptions{})
}

// IsHeadlessService determines whether or not a corev1.Service is a headless service
func IsHeadlessService(svc corev1.Service) bool {
	return len(svc.Spec.ClusterIP) == 0 || svc.Spec.ClusterIP == corev1.ClusterIPNone
}

// GetMeshConfig returns the current MeshConfig
func (c *Client) GetMeshConfig() configv1alpha2.MeshConfig {
	key := types.NamespacedName{Namespace: c.osmNamespace, Name: c.meshConfigName}.String()
	item, _, err := c.informers.GetByKey(informers.InformerKeyMeshConfig, key)
	if item != nil {
		return *item.(*configv1alpha2.MeshConfig)
	}
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMeshConfigFetchFromCache)).Msgf("Error getting MeshConfig from cache with key %s", key)
	} else {
		log.Warn().Msgf("MeshConfig %s does not exist. Default config values will be used.", key)
	}

	return configv1alpha2.MeshConfig{}
}

// GetPodForProxy returns the pod that the given proxy is attached to, based on the UUID and service identity.
// TODO(4863): move this to kube/client.go
func (c *Client) GetPodForProxy(proxy *envoy.Proxy) (*v1.Pod, error) {
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

// GetOSMNamespace returns the namespace in which the OSM controller pod resides.
func (c *Client) GetOSMNamespace() string {
	return c.osmNamespace
}

// ListEgressPolicies lists the all Egress policies
func (c *Client) ListEgressPolicies() []*policyv1alpha1.Egress {
	var policies []*policyv1alpha1.Egress

	for _, egressIface := range c.informers.List(osminformers.InformerKeyEgress) {
		egressPolicy := egressIface.(*policyv1alpha1.Egress)

		if !c.IsMonitoredNamespace(egressPolicy.Namespace) {
			continue
		}
		policies = append(policies, egressPolicy)
	}

	return policies
}

// ListIngressBackendPolicies lists the all IngressBackend policies
func (c *Client) ListIngressBackendPolicies() []*policyv1alpha1.IngressBackend {
	var backends []*policyv1alpha1.IngressBackend

	for _, ingressBackendIface := range c.informers.List(osminformers.InformerKeyIngressBackend) {
		backend := ingressBackendIface.(*policyv1alpha1.IngressBackend)
		if !c.IsMonitoredNamespace(backend.Namespace) {
			continue
		}

		backends = append(backends, backend)
	}

	return backends
}

// ListRetryPolicies returns the retry policies for the given source identity based on service accounts.
func (c *Client) ListRetryPolicies() []*policyv1alpha1.Retry {
	var retries []*policyv1alpha1.Retry

	for _, retryInterface := range c.informers.List(osminformers.InformerKeyRetry) {
		policy := retryInterface.(*policyv1alpha1.Retry)
		if !c.IsMonitoredNamespace(policy.Namespace) {
			continue
		}

		retries = append(retries, policy)
	}

	return retries
}

// ListUpstreamTrafficSettings returns the all UpstreamTrafficSetting resources
func (c *Client) ListUpstreamTrafficSettings() []*policyv1alpha1.UpstreamTrafficSetting {
	var settings []*policyv1alpha1.UpstreamTrafficSetting

	// Filter by MeshService
	for _, resource := range c.informers.List(osminformers.InformerKeyUpstreamTrafficSetting) {
		setting := resource.(*policyv1alpha1.UpstreamTrafficSetting)

		if !c.IsMonitoredNamespace(setting.Namespace) {
			continue
		}

		settings = append(settings, setting)
	}

	return settings
}

// GetUpstreamTrafficSetting returns the UpstreamTrafficSetting resources with namespaced name
func (c *Client) GetUpstreamTrafficSetting(namespace *types.NamespacedName) *policyv1alpha1.UpstreamTrafficSetting {
	resource, exists, err := c.informers.GetByKey(osminformers.InformerKeyUpstreamTrafficSetting, namespace.String())
	if exists && err == nil {
		return resource.(*policyv1alpha1.UpstreamTrafficSetting)
	}
	return nil
}

// GetMeshRootCertificate returns a MeshRootCertificate resource with namespaced name
func (c *Client) GetMeshRootCertificate(mrcName string) *configv1alpha2.MeshRootCertificate {
	key := types.NamespacedName{Namespace: c.osmNamespace, Name: mrcName}.String()
	resource, exists, err := c.informers.GetByKey(osminformers.InformerKeyMeshRootCertificate, key)
	if exists && err == nil {
		return resource.(*configv1alpha2.MeshRootCertificate)
	}
	return nil
}

// UpdateMeshRootCertificate updates a MeshRootCertificate.
func (c *Client) UpdateMeshRootCertificate(obj *configv1alpha2.MeshRootCertificate) (*configv1alpha2.MeshRootCertificate, error) {
	return c.configClient.ConfigV1alpha2().MeshRootCertificates(c.osmNamespace).Update(context.Background(), obj, metav1.UpdateOptions{})
}

// UpdateMeshRootCertificateStatus updates the status of a MeshRootCertificate.
func (c *Client) UpdateMeshRootCertificateStatus(obj *configv1alpha2.MeshRootCertificate) (*configv1alpha2.MeshRootCertificate, error) {
	return c.configClient.ConfigV1alpha2().MeshRootCertificates(c.osmNamespace).UpdateStatus(context.Background(), obj, metav1.UpdateOptions{})
}

// ListMeshRootCertificates returns the MRCs stored in the informerCollection's store
func (c *Client) ListMeshRootCertificates() ([]*configv1alpha2.MeshRootCertificate, error) {
	// informers return slice of generic, essentially untyped so we'll convert them to value types before returning
	mrcPtrs := c.informers.List(informers.InformerKeyMeshRootCertificate)
	var mrcs []*configv1alpha2.MeshRootCertificate
	for _, mrcPtr := range mrcPtrs {
		if mrcPtr == nil {
			continue
		}
		mrc, ok := mrcPtr.(*configv1alpha2.MeshRootCertificate)
		if !ok {
			continue
		}
		mrcs = append(mrcs, mrc)
	}

	return mrcs, nil
}

// AddMRCEventsHandler adds the event handler for MRCEvents
func (c *Client) AddMRCEventsHandler(handlerFuncs cache.ResourceEventHandlerFuncs) {
	c.informers.AddEventHandler(osminformers.InformerKeyMeshRootCertificate, handlerFuncs)
}
