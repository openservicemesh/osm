package subm

import (
	"fmt"
	"net"
	"reflect"
	"strings"

	mapset "github.com/deckarep/golang-set"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"

	"github.com/pkg/errors"
	// appsv1 "k8s.io/api/apps/v1"
	// corev1 "k8s.io/api/core/v1"
	v2alpha1 "github.com/openservicemesh/osm/pkg/apis/submresource/v2alpha1"
	informers "github.com/openservicemesh/osm/pkg/subm_client/informers/externalversions"
	serviceimportclienteset "github.com/openservicemesh/osm/pkg/subm_client/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/smi"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	// "github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/namespace"
)

const namespaceSelectorLabel = "app"
const submNamespace = "submariner-operator"

// NewProvider implements mesh.EndpointsProvider, which creates a new Kubernetes cluster/compute provider.
func NewProvider(kubeClient kubernetes.Interface, clusterId string, stop chan struct{}, meshSpec smi.MeshSpec, providerIdent string, config *rest.Config) (*Client, error) {
	submClient := serviceimportclienteset.NewForConfigOrDie(config)
	submNamespaceController := namespace.NewNamespaceController(kubeClient, submNamespace, stop)

	informerFactory := informers.NewSharedInformerFactory(submClient, k8s.DefaultKubeEventResyncInterval)

	informerCollection := InformerCollection{
		// Endpoints:   informerFactory.Core().V1().Endpoints().Informer(),
		ServiceImports: informerFactory.Lighthouse().V2alpha1().ServiceImports().Informer(),
	}

	cacheCollection := CacheCollection{
		ServiceImports: informerCollection.ServiceImports.GetStore(),
	}

	client := Client{
		providerIdent:       providerIdent,
		clusterId:           clusterId,
		kubeClient:          kubeClient,
		meshSpec:            meshSpec,
		informers:           &informerCollection,
		caches:              &cacheCollection,
		cacheSynced:         make(chan interface{}),
		announcements:       make(chan interface{}),
		namespaceController: submNamespaceController,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return submNamespaceController.IsMonitoredNamespace(ns)
	}
	informerCollection.ServiceImports.AddEventHandler(k8s.GetKubernetesEventHandlers("ServiceImports", client.providerIdent, client.announcements, shouldObserve))

	if err := client.run(stop); err != nil {
		return nil, errors.Errorf("Failed to start Submariner EndpointProvider client: %+v", err)
	}
	log.Info().Msgf("[NewProvider] started Submariner provider")

	return &client, nil
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interface: EndpointsProvider
func (c *Client) GetID() string {
	return c.providerIdent
}

func (c *Client) filterTargetList(svc string) []*target.TrafficTarget {
	var filteredTargets []*target.TrafficTarget
	trafficTargets := c.meshSpec.ListTrafficTargets()
	for _, trafficTarget := range trafficTargets {
		if trafficTarget.Spec.Destination.Name != svc {
			continue
		}
		filteredTargets = append(filteredTargets, trafficTarget)
	}
	return filteredTargets
}

// ListEndpointsForService retrieves the list of IP addresses for the given service
func (c Client) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	stripClusterId := func(name string) string {
		// It is assumed that clusterId doesn't have any '-'
		// but other words can have "-". 
		// ex: dhcp-watch-default-hq should return dhcp-watch-default
		words := strings.Split(name, "-")
		n := len(words)
		suffix := "-" + words[n-1]
		return strings.TrimSuffix(name, suffix)
	}

	log.Info().Msgf("[%s] Getting Endpoints for service %s on Submariner", c.providerIdent, svc)
	var endpoints []endpoint.Endpoint = []endpoint.Endpoint{}

	s := strings.Split(svc.String(), "/") // svc.String() -> "default/gateway"

	destTargetList := c.filterTargetList(s[1])
	svcNsName := fmt.Sprintf("%s-%s", s[1], s[0])

	serviceImportInterfaces := c.caches.ServiceImports.List()
	log.Info().Msgf("[%s] imports : %+v", c.providerIdent, serviceImportInterfaces)
	for _, si := range serviceImportInterfaces {
		serviceImport := si.(*v2alpha1.ServiceImport)
		if serviceImport == nil ||
		   len(serviceImport.Status.Clusters) == 0 ||
		   len(serviceImport.Status.Clusters[0].IPs) == 0 {
			continue
		}
		siNsName := stripClusterId(serviceImport.ObjectMeta.Name)
		log.Info().Msgf("[%s] siNsName:%s svcNsName:%s", c.providerIdent, siNsName, svcNsName)
		if siNsName != svcNsName {
			// match except the cluster-name
			continue
		}

		for _, target := range destTargetList {
			ept := endpoint.Endpoint{
				IP:   net.ParseIP(serviceImport.Status.Clusters[0].IPs[0]),
				Port: endpoint.Port(*target.Spec.Destination.Port),
			}
			endpoints = append(endpoints, ept)
		}
		break
	}
	log.Info().Msgf("[%s] Endpoints for service %s on Submariner endpoints:%+v", c.providerIdent, svc.String(), endpoints)

	return endpoints
}

// GetServiceForServiceAccount retrieves the service for the given service account
func (c Client) GetServiceForServiceAccount(svcAccount service.K8sServiceAccount) (service.MeshService, error) {
	log.Info().Msgf("[%s] Getting Services for service account %s on Submariner", c.providerIdent, svcAccount)
	services := mapset.NewSet()
	serviceImportInterfaces := c.caches.ServiceImports.List()

	for _, deployments := range serviceImportInterfaces {
		serviceImport := deployments.(*v2alpha1.ServiceImport)
		if serviceImport != nil {
			/*
			if !c.namespaceController.IsMonitoredNamespace(serviceImport.Namespace) {
				// Doesn't belong to namespaces we are observing
				continue
			}
			*/
			strs := strings.Split(serviceImport.ObjectMeta.Name, "-")
			importSvc, importNamespace, importClusterId := strs[0], strs[1], strs[2]
			if c.clusterId != importClusterId &&
			   svcAccount.Name == importSvc &&
			   svcAccount.Namespace == importNamespace {
				namespacedService := service.MeshService{
					Namespace: importNamespace,
					Name:      importSvc,
				}
				services.Add(namespacedService)
			}
		}
	}

	if services.Cardinality() == 0 {
		log.Error().Msgf("Did not find any service with serviceAccount = %s in namespace %s", svcAccount.Name, svcAccount.Namespace)
		return service.MeshService{}, errDidNotFindServiceForServiceAccount
	}

	// --- CONVENTION ---
	// By Open Service Mesh convention the number of services for a service account is 1
	// This is a limitation we set in place in order to make the mesh easy to understand and reason about.
	// When a service account has more than one service XDS will not apply any SMI policy for that service, leaving it out of the mesh.
	if services.Cardinality() > 1 {
		log.Error().Msgf("Found more than one service for serviceAccount %s in namespace %s; There should be only one!", svcAccount.Name, svcAccount.Namespace)
		return service.MeshService{}, errMoreThanServiceForServiceAccount
	}

	log.Info().Msgf("[%s] Services %v observed on service account %s on Kubernetes", c.providerIdent, services, svcAccount)
	svc := services.Pop().(service.MeshService)
	log.Trace().Msgf("Found service %s for serviceAccount %s in namespace %s", svc.Name, svcAccount.Name, svcAccount.Namespace)
	return svc, nil
}

// GetAnnouncementsChannel returns the announcement channel for the Kubernetes endpoints provider.
func (c Client) GetAnnouncementsChannel() <-chan interface{} {
	return c.announcements
}

func (c *Client) run(stop <-chan struct{}) error {
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[string]cache.SharedInformer{
		"ServiceImports":   c.informers.ServiceImports,
	}

	var names []string
	for name, informer := range sharedInformers {
		// Depending on the use-case, some Informers from the collection may not have been initialized.
		if informer == nil {
			continue
		}
		names = append(names, name)
		log.Debug().Msgf("Starting informer %s", name)
		go informer.Run(stop)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	log.Info().Msgf("Waiting for informer's cache to sync: %+v", strings.Join(names, ", "))
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	log.Info().Msgf("Cache sync finished for %+v", names)
	return nil
}
