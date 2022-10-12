package k8s

import (
	"context"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	mcs "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/models"
)

// NewClient returns a new kubernetes.Controller which means to provide access to locally-cached k8s resources
func NewClient(osmNamespace, meshConfigName string, broker *messaging.Broker, opts ...ClientOption) (*Client, error) {
	// Initialize client object
	c := &Client{
		informers:      map[informerKey]cache.SharedIndexInformer{},
		msgBroker:      broker,
		osmNamespace:   osmNamespace,
		meshConfigName: meshConfigName,
	}
	// Execute all of the given options (e.g. set clients, set custom stores, etc.)
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	if err := c.run(broker.Done()); err != nil {
		log.Error().Err(err).Msg("Could not start informer collection")
		return nil, err
	}

	return c, nil
}

func key(name, namespace string) string {
	return types.NamespacedName{Name: name, Namespace: namespace}.String()
}

// ListNamespaces returns all namespaces that the mesh is monitoring.
func (c *Client) ListNamespaces() ([]*corev1.Namespace, error) {
	var namespaces []*corev1.Namespace

	for _, ns := range c.list(informerKeyNamespace) {
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
	svcIf, exists, err := c.getByKey(informerKeyService, key(name, namespace))
	if exists && err == nil {
		svc := svcIf.(*corev1.Service)
		return svc
	}
	return nil
}

// GetSecret returns the secret for a given secret name and namespace
func (c *Client) GetSecret(name, namespace string) *models.Secret {
	secretIf, exists, err := c.getByKey(informerKeySecret, key(name, namespace))
	if exists && err == nil {
		corev1Secret, ok := secretIf.(*corev1.Secret)
		if !ok {
			return nil
		}
		return &models.Secret{
			Name:      corev1Secret.Name,
			Namespace: corev1Secret.Namespace,
			Data:      corev1Secret.Data,
		}
	}
	return nil
}

// ListSecrets returns a list of secrets
func (c *Client) ListSecrets() []*models.Secret {
	var secrets []*models.Secret

	for _, secretPtr := range c.list(informerKeySecret) {
		if secretPtr == nil {
			continue
		}
		secret, ok := secretPtr.(*corev1.Secret)
		if !ok {
			continue
		}

		secrets = append(secrets, &models.Secret{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Data:      secret.Data,
		})
	}

	return secrets
}

// UpdateSecret updates the given secret
func (c *Client) UpdateSecret(ctx context.Context, secret *models.Secret) error {
	corev1Secret, err := c.kubeClient.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	corev1Secret.Data = secret.Data
	_, err = c.kubeClient.CoreV1().Secrets(secret.Namespace).Update(ctx, corev1Secret, metav1.UpdateOptions{})
	return err
}

// ListServices returns a list of services that are part of monitored namespaces
func (c *Client) ListServices() []*corev1.Service {
	var services []*corev1.Service

	for _, serviceInterface := range c.list(informerKeyService) {
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

	for _, serviceInterface := range c.list(informerKeyServiceAccount) {
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
	nsIf, exists, err := c.getByKey(informerKeyNamespace, ns)
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

	for _, podInterface := range c.list(informerKeyPod) {
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
	ep, exists, err := c.getByKey(informerKeyEndpoints, key(name, namespace))
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
	item, _, err := c.getByKey(informerKeyMeshConfig, key)
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

// GetOSMNamespace returns the namespace in which the OSM controller pod resides.
func (c *Client) GetOSMNamespace() string {
	return c.osmNamespace
}

// ListEgressPolicies lists the all Egress policies
func (c *Client) ListEgressPolicies() []*policyv1alpha1.Egress {
	var policies []*policyv1alpha1.Egress

	for _, egressIface := range c.list(informerKeyEgress) {
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

	for _, ingressBackendIface := range c.list(informerKeyIngressBackend) {
		backend := ingressBackendIface.(*policyv1alpha1.IngressBackend)
		if !c.IsMonitoredNamespace(backend.Namespace) {
			continue
		}

		backends = append(backends, backend)
	}

	return backends
}

// GetExtensionService returns the extension service for the given service ref
func (c *Client) GetExtensionService(svc policyv1alpha1.ExtensionServiceRef) *configv1alpha2.ExtensionService {
	resource, exists, err := c.getByKey(informerKeyExtensionService, key(svc.Name, svc.Namespace))
	if exists && err == nil {
		return resource.(*configv1alpha2.ExtensionService)
	}
	return nil
}

// ListRetryPolicies returns the retry policies for the given source identity based on service accounts.
func (c *Client) ListRetryPolicies() []*policyv1alpha1.Retry {
	var retries []*policyv1alpha1.Retry

	for _, retryInterface := range c.list(informerKeyRetry) {
		policy := retryInterface.(*policyv1alpha1.Retry)
		if !c.IsMonitoredNamespace(policy.Namespace) {
			continue
		}

		retries = append(retries, policy)
	}

	return retries
}

// ListTelemetryPolicies returns all the telemetry policies.
func (c *Client) ListTelemetryPolicies() []*policyv1alpha1.Telemetry {
	var telemetryPolicies []*policyv1alpha1.Telemetry

	for _, resource := range c.list(informerKeyTelemetry) {
		t := resource.(*policyv1alpha1.Telemetry)

		if !c.IsMonitoredNamespace(t.Namespace) {
			continue
		}

		telemetryPolicies = append(telemetryPolicies, t)
	}

	return telemetryPolicies
}

// ListUpstreamTrafficSettings returns the all UpstreamTrafficSetting resources
func (c *Client) ListUpstreamTrafficSettings() []*policyv1alpha1.UpstreamTrafficSetting {
	var settings []*policyv1alpha1.UpstreamTrafficSetting

	// Filter by MeshService
	for _, resource := range c.list(informerKeyUpstreamTrafficSetting) {
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
	resource, exists, err := c.getByKey(informerKeyUpstreamTrafficSetting, namespace.String())
	if exists && err == nil {
		return resource.(*policyv1alpha1.UpstreamTrafficSetting)
	}
	return nil
}

// GetMeshRootCertificate returns a MeshRootCertificate resource with namespaced name
func (c *Client) GetMeshRootCertificate(mrcName string) *configv1alpha2.MeshRootCertificate {
	key := types.NamespacedName{Namespace: c.osmNamespace, Name: mrcName}.String()
	resource, exists, err := c.getByKey(informerKeyMeshRootCertificate, key)
	if exists && err == nil {
		return resource.(*configv1alpha2.MeshRootCertificate)
	}
	return nil
}

// ListTrafficSplits implements mesh.MeshSpec by returning the list of traffic splits.
func (c *Client) ListTrafficSplits() []*smiSplit.TrafficSplit {
	var trafficSplits []*smiSplit.TrafficSplit

	for _, splitIface := range c.list(informerKeyTrafficSplit) {
		trafficSplit := splitIface.(*smiSplit.TrafficSplit)

		if !c.IsMonitoredNamespace(trafficSplit.Namespace) {
			continue
		}
		trafficSplits = append(trafficSplits, trafficSplit)
	}
	return trafficSplits
}

// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
func (c *Client) ListHTTPTrafficSpecs() []*smiSpecs.HTTPRouteGroup {
	var httpTrafficSpec []*smiSpecs.HTTPRouteGroup
	for _, specIface := range c.list(informerKeyHTTPRouteGroup) {
		routeGroup := specIface.(*smiSpecs.HTTPRouteGroup)

		if !c.IsMonitoredNamespace(routeGroup.Namespace) {
			continue
		}
		httpTrafficSpec = append(httpTrafficSpec, routeGroup)
	}
	return httpTrafficSpec
}

// GetHTTPRouteGroup returns an SMI HTTPRouteGroup resource given its name of the form <namespace>/<name>
func (c *Client) GetHTTPRouteGroup(namespacedName string) *smiSpecs.HTTPRouteGroup {
	// client-go cache uses <namespace>/<name> as key
	routeIf, exists, err := c.getByKey(informerKeyHTTPRouteGroup, namespacedName)
	if exists && err == nil {
		route := routeIf.(*smiSpecs.HTTPRouteGroup)
		if !c.IsMonitoredNamespace(route.Namespace) {
			return nil
		}
		return route
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
	mrcPtrs := c.list(informerKeyMeshRootCertificate)
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

// ListTCPTrafficSpecs lists SMI TCPRoute resources
func (c *Client) ListTCPTrafficSpecs() []*smiSpecs.TCPRoute {
	var tcpRouteSpec []*smiSpecs.TCPRoute
	for _, specIface := range c.list(informerKeyTCPRoute) {
		tcpRoute := specIface.(*smiSpecs.TCPRoute)

		if !c.IsMonitoredNamespace(tcpRoute.Namespace) {
			continue
		}
		tcpRouteSpec = append(tcpRouteSpec, tcpRoute)
	}
	return tcpRouteSpec
}

// GetTCPRoute returns an SMI TCPRoute resource given its name of the form <namespace>/<name>
func (c *Client) GetTCPRoute(namespacedName string) *smiSpecs.TCPRoute {
	// client-go cache uses <namespace>/<name> as key
	routeIf, exists, err := c.getByKey(informerKeyTCPRoute, namespacedName)
	if exists && err == nil {
		route := routeIf.(*smiSpecs.TCPRoute)
		if !c.IsMonitoredNamespace(route.Namespace) {
			log.Warn().Msgf("TCPRoute %s found, but belongs to a namespace that is not monitored, ignoring it", namespacedName)
			return nil
		}
		return route
	}
	return nil
}

// ListTrafficTargets returns the list of traffic targets.
func (c *Client) ListTrafficTargets() []*smiAccess.TrafficTarget {
	var trafficTargets []*smiAccess.TrafficTarget

	for _, targetIface := range c.list(informerKeyTrafficTarget) {
		trafficTarget := targetIface.(*smiAccess.TrafficTarget)

		if !c.IsMonitoredNamespace(trafficTarget.Namespace) {
			continue
		}
		trafficTargets = append(trafficTargets, trafficTarget)
	}
	return trafficTargets
}

// ListServiceImports returns all ServiceImport resources
func (c *Client) ListServiceImports() []*mcs.ServiceImport {
	var serviceImports []*mcs.ServiceImport

	for _, resource := range c.list(informerKeyServiceImport) {
		serviceImport := resource.(*mcs.ServiceImport)
		if !c.IsMonitoredNamespace(serviceImport.Namespace) {
			continue
		}
		serviceImports = append(serviceImports, serviceImport)
	}

	return serviceImports
}

// ListServiceExports returns all ServiceExport resources
func (c *Client) ListServiceExports() []*mcs.ServiceExport {
	var serviceExports []*mcs.ServiceExport

	for _, resource := range c.list(informerKeyServiceExport) {
		serviceExport := resource.(*mcs.ServiceExport)
		if !c.IsMonitoredNamespace(serviceExport.Namespace) {
			continue
		}
		serviceExports = append(serviceExports, serviceExport)
	}

	return serviceExports
}

// AddMeshRootCertificateEventHandler adds an event handler specific to mesh root certificiates.
func (c *Client) AddMeshRootCertificateEventHandler(handler cache.ResourceEventHandler) {
	c.informers[informerKeyMeshRootCertificate].AddEventHandler(handler)
}
